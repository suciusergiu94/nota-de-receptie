package store

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"nota-de-receptie/internal/model"
)

// schemaV1 is the shape the application shipped with, before the imported
// rows needed a column of their own. The migration has to carry a file in
// this shape forward without losing anything: by the time it runs, it holds
// documents nobody is going to retype.
const schemaV1 = `
CREATE TABLE settings (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  unitate_nume TEXT NOT NULL DEFAULT '',
  next_nr INTEGER NOT NULL DEFAULT 1,
  cota_tva REAL NOT NULL DEFAULT 11
);
CREATE TABLE products (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  denumire TEXT NOT NULL,
  um TEXT NOT NULL DEFAULT 'Kg.',
  pret_vanzare REAL NOT NULL DEFAULT 0,
  cota_tva REAL NOT NULL DEFAULT 11,
  ordine INTEGER NOT NULL
);
CREATE UNIQUE INDEX idx_products_denumire ON products(denumire COLLATE NOCASE);
CREATE TABLE furnizori (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  nume TEXT NOT NULL
);
CREATE UNIQUE INDEX idx_furnizori_nume ON furnizori(nume COLLATE NOCASE);
CREATE TABLE documents (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  nr INTEGER NOT NULL,
  data TEXT NOT NULL,
  unitate TEXT NOT NULL DEFAULT '',
  document_livrare TEXT NOT NULL DEFAULT '',
  document_livrare_nr TEXT NOT NULL DEFAULT '',
  document_livrare_data TEXT NOT NULL DEFAULT '',
  furnizor TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE TABLE document_rows (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  document_id INTEGER NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  product_id INTEGER REFERENCES products(id) ON DELETE SET NULL,
  pozitie INTEGER NOT NULL,
  denumire TEXT NOT NULL,
  um TEXT NOT NULL DEFAULT 'Kg.',
  cantitate REAL NOT NULL DEFAULT 0,
  pret_fara_tva REAL NOT NULL DEFAULT 0,
  cota_tva REAL NOT NULL DEFAULT 11,
  pret_vanzare REAL NOT NULL DEFAULT 0
);
PRAGMA user_version = 1;
`

// bazaV1 writes a version 1 file holding one document with two rows, and
// returns its path.
func bazaV1(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "v1.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(schemaV1); err != nil {
		t.Fatalf("schema v1: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO settings (id, unitate_nume, next_nr, cota_tva)
		 VALUES (1, 'S.C. Largiana Carn S.R.L.', 8, 11)`,
	); err != nil {
		t.Fatalf("settings v1: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO documents (id, nr, data, unitate, document_livrare,
		        document_livrare_nr, document_livrare_data, furnizor,
		        created_at, updated_at)
		 VALUES (1, 7, '2026-09-01', 'S.C. Largiana Carn S.R.L.', 'Factura',
		        '1234', '2026-08-31', 'Alfa SRL', '2026-09-01T10:00:00Z',
		        '2026-09-01T10:00:00Z')`,
	); err != nil {
		t.Fatalf("document v1: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO document_rows (document_id, pozitie, denumire, um, cantitate,
		        pret_fara_tva, cota_tva, pret_vanzare)
		 VALUES (1, 0, 'Pulpa fara os', 'Kg.', 10, 20, 11, 30),
		        (1, 1, 'Oua', 'Buc.', 30, 0.9, 11, 1.5)`,
	); err != nil {
		t.Fatalf("randuri v1: %v", err)
	}
	return path
}

func TestMigrareaDeLaV1PastreazaDocumentele(t *testing.T) {
	path := bazaV1(t)

	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open pe o baza v1: %v", err)
	}
	defer s.Close()

	doc, err := s.GetDocument(1)
	if err != nil {
		t.Fatalf("GetDocument dupa migrare: %v", err)
	}
	if doc.Nr != 7 || doc.Furnizor != "Alfa SRL" {
		t.Errorf("documentul s-a schimbat: nr = %d, furnizor = %q", doc.Nr, doc.Furnizor)
	}
	if len(doc.Randuri) != 2 {
		t.Fatalf("randuri = %d, vrem 2", len(doc.Randuri))
	}
	if doc.Randuri[0].Denumire != "Pulpa fara os" || doc.Randuri[1].Denumire != "Oua" {
		t.Errorf("randurile s-au schimbat: %q, %q",
			doc.Randuri[0].Denumire, doc.Randuri[1].Denumire)
	}
	if doc.Randuri[0].ValoareVanzareImpusa != nil {
		t.Error("un rand vechi a capatat o valoare impusa")
	}

	var versiune int
	if err := s.db.QueryRow(`PRAGMA user_version`).Scan(&versiune); err != nil {
		t.Fatalf("PRAGMA user_version: %v", err)
	}
	if versiune != schemaVersion {
		t.Errorf("user_version = %d, vrem %d", versiune, schemaVersion)
	}
}

func TestMigrareaEsteIdempotenta(t *testing.T) {
	// A fresh database already has the column, though its user_version is 0
	// on the way in; a second Open must not try to add it again.
	path := filepath.Join(t.TempDir(), "test.db")
	for i := 0; i < 3; i++ {
		s, err := Open(path)
		if err != nil {
			t.Fatalf("Open #%d: %v", i+1, err)
		}
		s.Close()
	}
}

func TestValoareaImpusaFaceDusIntorsPrinBaza(t *testing.T) {
	s := deschide(t)
	impusa := 2501.35
	doc := docExemplu(nil)
	doc.Randuri = append(doc.Randuri, model.Rand{
		Pozitie: 2, Denumire: "Carcasa", UM: "Kg.", Cantitate: 162.2,
		PretFaraTVA: 12.30, CotaTVA: 11, PretVanzare: 15.42,
		ValoareVanzareImpusa: &impusa,
	})

	salvat, err := s.SaveDocument(doc)
	if err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}
	citit, err := s.GetDocument(salvat.ID)
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}

	var cuValoare int
	for _, r := range citit.Randuri {
		if r.ValoareVanzareImpusa == nil {
			continue
		}
		cuValoare++
		if *r.ValoareVanzareImpusa != 2501.35 {
			t.Errorf("ValoareVanzareImpusa = %v, vrem 2501.35", *r.ValoareVanzareImpusa)
		}
	}
	if cuValoare != 1 {
		t.Errorf("randuri cu valoare impusa = %d, vrem 1", cuValoare)
	}
}

// TestValoareaImpusaZeroNuSeConfundaCuNimic verifies the reason the column is
// nullable at all: an imposed value of exactly 0 is a real, distinct
// statement ("someone imposed a sale value of zero") and must not collapse
// to nil ("nothing imposed") on the way through the database, nor be
// confused with a sibling row that genuinely has nothing imposed.
func TestValoareaImpusaZeroNuSeConfundaCuNimic(t *testing.T) {
	s := deschide(t)
	zero := 0.0
	doc := docExemplu(nil)
	doc.Randuri = append(doc.Randuri, model.Rand{
		Pozitie: 2, Denumire: "Carcasa", UM: "Kg.", Cantitate: 162.2,
		PretFaraTVA: 12.30, CotaTVA: 11, PretVanzare: 15.42,
		ValoareVanzareImpusa: &zero,
	})

	salvat, err := s.SaveDocument(doc)
	if err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}
	citit, err := s.GetDocument(salvat.ID)
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}
	if len(citit.Randuri) != 3 {
		t.Fatalf("randuri = %d, vrem 3", len(citit.Randuri))
	}

	// The two rows from docExemplu carry no imposed value at all.
	for i, r := range citit.Randuri[:2] {
		if r.ValoareVanzareImpusa != nil {
			t.Errorf("randul %d: ValoareVanzareImpusa = %v, vrem nil", i, *r.ValoareVanzareImpusa)
		}
	}

	// The third row imposes exactly 0, which must survive as a non-nil zero.
	impusaCitita := citit.Randuri[2].ValoareVanzareImpusa
	if impusaCitita == nil {
		t.Fatal("ValoareVanzareImpusa = nil, vrem un pointer catre 0")
	}
	if *impusaCitita != 0 {
		t.Errorf("ValoareVanzareImpusa = %v, vrem 0", *impusaCitita)
	}
}
