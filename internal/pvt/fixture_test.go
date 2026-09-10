package pvt

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// schemaSora is the shape of the sibling application's database at its schema
// version 5, copied here on purpose. It is the contract between the two
// applications; when the other project changes it, a test in this package
// failing is exactly the warning we want.
const schemaSora = `
CREATE TABLE settings (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  unitate_nume TEXT NOT NULL DEFAULT '',
  next_nr INTEGER NOT NULL DEFAULT 1,
  cota_tva REAL NOT NULL DEFAULT 11,
  gestiune TEXT NOT NULL DEFAULT ''
);
CREATE TABLE templates (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  nume TEXT NOT NULL,
  ordine INTEGER NOT NULL
);
CREATE TABLE products (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  template_id INTEGER NOT NULL REFERENCES templates(id) ON DELETE CASCADE,
  denumire TEXT NOT NULL,
  um TEXT NOT NULL DEFAULT 'Kg',
  pret_cu_tva REAL NOT NULL DEFAULT 0,
  procent_din_intrare REAL NOT NULL DEFAULT 0,
  ordine INTEGER NOT NULL
);
CREATE TABLE documents (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  template_id INTEGER REFERENCES templates(id) ON DELETE SET NULL,
  nr INTEGER NOT NULL,
  data TEXT NOT NULL,
  gestiune TEXT NOT NULL,
  document_referinta TEXT NOT NULL DEFAULT '',
  diferenta_tip TEXT NOT NULL DEFAULT '',
  diferenta_valoare REAL NOT NULL DEFAULT 0,
  incarca_descarca_tip TEXT NOT NULL DEFAULT '',
  incarca_descarca_valoare REAL NOT NULL DEFAULT 0,
  gestionar TEXT NOT NULL DEFAULT '',
  calculator TEXT NOT NULL DEFAULT '',
  vizat_compartiment_productie TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE TABLE document_intrare_rows (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  document_id INTEGER NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  pozitie INTEGER NOT NULL,
  denumire TEXT NOT NULL DEFAULT '',
  um TEXT NOT NULL DEFAULT '',
  cantitate REAL NOT NULL DEFAULT 0,
  pret_fara_tva REAL NOT NULL DEFAULT 0,
  pret_cu_tva REAL NOT NULL DEFAULT 0,
  cota_tva REAL NOT NULL DEFAULT 11
);
CREATE TABLE document_iesire_rows (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  document_id INTEGER NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  product_id INTEGER REFERENCES products(id) ON DELETE SET NULL,
  pozitie INTEGER NOT NULL,
  denumire TEXT NOT NULL,
  um TEXT NOT NULL,
  pret_cu_tva REAL NOT NULL,
  cantitate REAL NOT NULL DEFAULT 0,
  pret_fara_tva REAL NOT NULL DEFAULT 0,
  cota_tva REAL NOT NULL DEFAULT 11
);
`

// fixtura writes a sibling-shaped database holding two proces verbal
// documents and returns its path.
//
// Document 1 is the worked example the plan and the spec both use: a 162.2 Kg
// carcass bought at 12.30 lei/Kg, cut into pieces whose "ce iese" table comes
// to 2501.35 lei with TVA.
//
// Document 2 has two "ce intra" rows, so the proportional split has something
// to split.
func fixtura(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "data.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(schemaSora); err != nil {
		t.Fatalf("schema sora: %v", err)
	}
	if _, err := db.Exec(`PRAGMA user_version = 5`); err != nil {
		t.Fatalf("user_version: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO settings (id, unitate_nume, next_nr, cota_tva, gestiune)
		 VALUES (1, 'S.C. Largiana Carn S.R.L.', 3, 11, 'Magazin Bradet')`,
	); err != nil {
		t.Fatalf("settings: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO documents (id, nr, data, gestiune, created_at, updated_at)
		 VALUES (1, 1, '2026-09-01', 'Magazin Bradet', '2026-09-01T10:00:00Z', '2026-09-01T10:00:00Z'),
		        (2, 2, '2026-09-05', 'Magazin Centru', '2026-09-05T10:00:00Z', '2026-09-05T10:00:00Z')`,
	); err != nil {
		t.Fatalf("documents: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO document_intrare_rows (document_id, pozitie, denumire, um,
		        cantitate, pret_fara_tva, pret_cu_tva, cota_tva)
		 VALUES (1, 0, 'Carcasa', 'Kg', 162.2, 12.30, 13.653, 11),
		        (2, 0, 'Carcasa', 'Kg', 100, 10, 11.1, 11),
		        (2, 1, 'Pulpa vita Angus', 'Kg', 100, 10, 11.1, 11)`,
	); err != nil {
		t.Fatalf("intrare: %v", err)
	}
	// Document 1's "ce iese" comes to 2501.35: 100 x 15 = 1500, plus
	// Round2(50.0675 x 20) = 1001.35.
	// Document 2's comes to 100.
	if _, err := db.Exec(
		`INSERT INTO document_iesire_rows (document_id, pozitie, denumire, um,
		        pret_cu_tva, cantitate, pret_fara_tva, cota_tva)
		 VALUES (1, 0, 'Pulpa fara os', 'Kg', 15, 100, 13.51, 11),
		        (1, 1, 'Cotlet cu os', 'Kg', 20, 50.0675, 18.02, 11),
		        (2, 0, 'Ceva', 'Kg', 10, 10, 9.01, 11)`,
	); err != nil {
		t.Fatalf("iesire: %v", err)
	}
	return path
}
