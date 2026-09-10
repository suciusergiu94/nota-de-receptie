# Import proces verbal de transare — plan de implementare

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Un rând al notei de recepție poate primi valoarea la preț de vânzare din totalul „ce iese” al unui proces verbal de transare, citit read-only din baza de date a aplicației surori.

**Architecture:** Un câmp nullable nou pe `model.Rand` (`ValoareVanzareImpusa`) pe care `calc.ValoriRand` îl preferă în locul înmulțirii cantitate × preț; prezența lui blochează rândul în formular și îl trimite la finalul tabelului. Un pachet nou `internal/pvt` este singurul loc care știe de aplicația soră: deschide fișierul ei `data.db` cu `query_only(1)`, citește, închide. Importul este un instantaneu — după el nota nu mai depinde de nimic.

**Tech Stack:** Go 1.25, Wails v2, `modernc.org/sqlite`, `github.com/go-pdf/fpdf`, TypeScript vanilla + vitest.

**Spec:** `docs/superpowers/specs/2026-09-10-import-proces-verbal-design.md`

## Global Constraints

- **Nu se scrie niciodată în baza aplicației surori.** Handle propriu cu
  `_pragma=query_only(1)`, niciodată `store.Open` — acela rulează `migrate`,
  care ar ștampila `user_version = 1` peste baza celuilalt proiect și ar
  declanșa `dropAllTables` la următoarea lui pornire.
- **Migrarea bazei proprii nu are voie să șteargă date.** `ALTER TABLE`, nu
  drop-and-recreate; aplicația a fost livrată.
- Mesajele de eroare din Go sunt în română **fără diacritice**, ca
  `store.ErrNotFound`. Textele din interfață (TypeScript) sunt în română **cu
  diacritice**.
- Comentariile în cod sunt în engleză, ca în tot restul proiectului. Numele
  testelor Go sunt în română fără diacritice (`TestNumeleTestului`).
- Nu se adaugă dependențe noi.
- Verificare după fiecare task: `go test ./...` și, unde s-a atins frontendul,
  `cd frontend && npm test`.

---

### Task 1: Valoarea de vânzare impusă în modelul și aritmetica Go

**Files:**
- Modify: `internal/model/model.go` (structura `Rand`)
- Modify: `internal/calc/calc.go` (`ValoriRand`)
- Test: `internal/calc/calc_test.go`

**Interfaces:**
- Consumes: nimic (primul task)
- Produces: `model.Rand.ValoareVanzareImpusa *float64` cu tag JSON
  `valoareVanzareImpusa,omitempty`. `calc.ValoriRand` întoarce
  `Valori.ValoareVanzare` egal cu valoarea impusă când aceasta există.

- [ ] **Step 1: Write the failing tests**

Adaugă la finalul lui `internal/calc/calc_test.go`:

```go
func TestValoareVanzareImpusaInlocuiesteInmultirea(t *testing.T) {
	// Carcasa de 162.2 Kg cumparata cu 12.30 lei/Kg, transata intr-un proces
	// verbal al carui tabel "ce iese" totalizeaza 2501.35 lei. Pretul unitar
	// rotunjit la doua zecimale (15.42) inmultit inapoi ar da 2501.12; nota
	// trebuie sa arate totalul procesului verbal, la ban.
	impusa := 2501.35
	v := ValoriRand(model.Rand{
		Cantitate:            162.2,
		PretFaraTVA:          12.30,
		CotaTVA:              11,
		PretVanzare:          15.42,
		ValoareVanzareImpusa: &impusa,
	})
	aproape(t, "ValoareFaraTVA", v.ValoareFaraTVA, 1995.06)
	aproape(t, "ValoareCuTVA", v.ValoareCuTVA, 2214.52)
	aproape(t, "ValoareVanzare", v.ValoareVanzare, 2501.35)
	aproape(t, "Adaos", v.Adaos, 286.83)
	aproape(t, "AdaosProcent", v.AdaosProcent, 12.95)
}

func TestValoareImpusaZeroNuEAcelasiLucruCuLipsaEi(t *testing.T) {
	// Un proces verbal din care iese numai deseu totalizeaza 0. Zero impus si
	// "nimic impus" trebuie sa dea rezultate diferite, altfel campul nu poate
	// exprima primul caz.
	zero := 0.0
	impus := ValoriRand(model.Rand{
		Cantitate: 10, PretFaraTVA: 5, CotaTVA: 11, PretVanzare: 8,
		ValoareVanzareImpusa: &zero,
	})
	aproape(t, "ValoareVanzare impusa", impus.ValoareVanzare, 0)

	normal := ValoriRand(model.Rand{
		Cantitate: 10, PretFaraTVA: 5, CotaTVA: 11, PretVanzare: 8,
	})
	aproape(t, "ValoareVanzare normala", normal.ValoareVanzare, 80)
}

func TestTotaluriInsumeazaSiRandurileCuValoareImpusa(t *testing.T) {
	impusa := 2501.35
	randuri := []model.Rand{
		{Cantitate: 10, PretFaraTVA: 20, CotaTVA: 11, PretVanzare: 30},
		{Cantitate: 162.2, PretFaraTVA: 12.30, CotaTVA: 11, PretVanzare: 15.42,
			ValoareVanzareImpusa: &impusa},
	}
	tot := Totaluri(randuri)
	aproape(t, "ValoareVanzare", tot.ValoareVanzare, 2801.35)
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/calc/ -run ValoareImpusa -v`
Expected: FAIL la compilare — `unknown field ValoareVanzareImpusa in struct literal of type model.Rand`.

- [ ] **Step 3: Add the field to the model**

În `internal/model/model.go`, în structura `Rand`, imediat după `PretVanzare`:

```go
	// ValoareVanzareImpusa, when set, is the row's valoare la preț de vânzare
	// as it came from a proces verbal de transare, rather than the cantitate ×
	// pret de vânzare the other rows derive it from. Its presence is also what
	// makes the row read-only in the form: the figure it carries only means
	// anything beside the quantity and price it was worked out from.
	//
	// A pointer rather than a zero sentinel: a proces verbal whose "ce iese"
	// table comes to 0 — nothing but waste — is a valid document, and an
	// imposed zero has to be distinguishable from nothing imposed at all.
	ValoareVanzareImpusa *float64 `json:"valoareVanzareImpusa,omitempty"`
```

- [ ] **Step 4: Add the branch to ValoriRand**

În `internal/calc/calc.go`, în `ValoriRand`, între linia care calculează
`vanzare` și `return`:

```go
	vanzare := Round2(r.Cantitate * r.PretVanzare)
	// A row imported from a proces verbal carries the figure its "ce iese"
	// table came to. Re-deriving it from the rounded unit price would miss it
	// by up to a leu on a whole carcass, and the two documents have to agree.
	if r.ValoareVanzareImpusa != nil {
		vanzare = Round2(*r.ValoareVanzareImpusa)
	}
	return valoriDin(faraTVA, cuTVA, vanzare)
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `go test ./internal/calc/ -v`
Expected: PASS, inclusiv testele existente.

- [ ] **Step 6: Commit**

```bash
git add internal/model/model.go internal/calc/calc.go internal/calc/calc_test.go
git commit -m "Valoare de vanzare impusa pe rand, folosita de calc"
```

---

### Task 2: Oglinda din frontend a aceleiași reguli

**Files:**
- Modify: `frontend/wailsjs/go/models.ts` (clasa `model.Rand`)
- Modify: `frontend/src/calc.ts` (`RandCalculabil`, `valoriRand`)
- Test: `frontend/src/calc.test.ts`

**Interfaces:**
- Consumes: `model.Rand.ValoareVanzareImpusa` din Task 1
- Produces: `RandCalculabil.valoareVanzareImpusa?: number | null`;
  `valoriRand` întoarce `valoareVanzare` egal cu valoarea impusă când aceasta
  nu e `null`/`undefined`.

Câmpul se verifică împotriva **și** lui `null`, **și** lui `undefined`: `omitempty`
îl scoate din JSON când e nil, dar un `Rand` construit în frontend îl poate
purta explicit `null`. Ambele înseamnă „nimic impus”.

- [ ] **Step 1: Write the failing tests**

Adaugă în `frontend/src/calc.test.ts`, în blocul `describe('valoriRand', ...)`:

```ts
  it('foloseste valoarea de vanzare impusa in locul inmultirii', () => {
    const v = valoriRand({
      cantitate: 162.2,
      pretFaraTva: 12.3,
      cotaTva: 11,
      pretVanzare: 15.42,
      valoareVanzareImpusa: 2501.35,
    });
    expect(v.valoareFaraTva).toBe(1995.06);
    expect(v.valoareCuTva).toBe(2214.52);
    expect(v.valoareVanzare).toBe(2501.35);
    expect(v.adaos).toBe(286.83);
    expect(v.adaosProcent).toBe(12.95);
  });

  it('trateaza null si undefined ca "nimic impus"', () => {
    const cuNull = valoriRand({
      cantitate: 10, pretFaraTva: 5, cotaTva: 11, pretVanzare: 8,
      valoareVanzareImpusa: null,
    });
    expect(cuNull.valoareVanzare).toBe(80);

    const fara = valoriRand({ cantitate: 10, pretFaraTva: 5, cotaTva: 11, pretVanzare: 8 });
    expect(fara.valoareVanzare).toBe(80);
  });

  it('deosebeste zero impus de lipsa valorii impuse', () => {
    const v = valoriRand({
      cantitate: 10, pretFaraTva: 5, cotaTva: 11, pretVanzare: 8,
      valoareVanzareImpusa: 0,
    });
    expect(v.valoareVanzare).toBe(0);
  });
```

Și în blocul `describe('totaluri', ...)`:

```ts
  it('insumeaza si randurile cu valoare impusa', () => {
    const tot = totaluri([
      { cantitate: 10, pretFaraTva: 20, cotaTva: 11, pretVanzare: 30 },
      {
        cantitate: 162.2, pretFaraTva: 12.3, cotaTva: 11, pretVanzare: 15.42,
        valoareVanzareImpusa: 2501.35,
      },
    ]);
    expect(tot.valoareVanzare).toBe(2801.35);
  });
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd frontend && npx vitest run src/calc.test.ts`
Expected: FAIL — `valoareVanzare` iese 2501.12 (nu 2501.35), iar TypeScript
semnalează că `valoareVanzareImpusa` nu există pe `RandCalculabil`.

- [ ] **Step 3: Extend RandCalculabil and valoriRand**

În `frontend/src/calc.ts`, în interfața `RandCalculabil`, după `pretVanzare`:

```ts
  /**
   * The row's valoare la preț de vânzare when it came from a proces verbal de
   * transare, instead of the cantitate × preț this file derives for every
   * other row. Null and undefined both mean "nothing imposed": the Go side
   * omits the field when it is nil, but a row built here may carry it null.
   */
  valoareVanzareImpusa?: number | null;
```

Și în `valoriRand`:

```ts
export function valoriRand(r: RandCalculabil): Valori {
  const faraTva = round2(r.cantitate * r.pretFaraTva);
  const cuTva = round2(faraTva * (1 + r.cotaTva / 100));
  const vanzare =
    r.valoareVanzareImpusa === undefined || r.valoareVanzareImpusa === null
      ? round2(r.cantitate * r.pretVanzare)
      : round2(r.valoareVanzareImpusa);
  return valoriDin(faraTva, cuTva, vanzare);
}
```

- [ ] **Step 4: Add the field to the generated model**

`frontend/wailsjs/go/models.ts` este generat de Wails, dar comis în depozit.
În clasa `model.Rand`, după `pretVanzare: number;`:

```ts
	    valoareVanzareImpusa?: number;
```

și în constructor, după linia `this.pretVanzare = ...`:

```ts
	        this.valoareVanzareImpusa = source["valoareVanzareImpusa"];
```

(Indentarea din acest fișier este cu tab urmat de patru spații — copiaz-o de la
liniile vecine. Dacă ai `wails` instalat, `wails generate module` produce exact
același rezultat.)

- [ ] **Step 5: Run the tests and the typecheck**

Run: `cd frontend && npm test && npm run build`
Expected: PASS la teste, `tsc` fără erori.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/calc.ts frontend/src/calc.test.ts frontend/wailsjs/go/models.ts
git commit -m "Oglinda din frontend a valorii de vanzare impuse"
```

---

### Task 3: Migrarea bazei la versiunea 2

**Files:**
- Modify: `internal/store/schema.go` (`schemaVersion`, `schemaSQL`, `migrate`)
- Modify: `internal/store/documents.go` (`randuri`, `SaveDocument`)
- Test: `internal/store/schema_test.go` (creat)

**Interfaces:**
- Consumes: `model.Rand.ValoareVanzareImpusa` din Task 1
- Produces: coloana `valoare_vanzare_impusa REAL` (nullable) în
  `document_rows`; `SaveDocument`/`GetDocument` o scriu și o citesc.

- [ ] **Step 1: Write the failing tests**

Creează `internal/store/schema_test.go`:

```go
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
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/store/ -run "Migrarea|ValoareaImpusa" -v`
Expected: FAIL — `TestMigrareaDeLaV1PastreazaDocumentele` pică la `GetDocument`
(`no such column: valoare_vanzare_impusa`), iar `TestValoareaImpusaFaceDusIntors`
găsește 0 rânduri cu valoare.

- [ ] **Step 3: Bump the schema and add the column**

În `internal/store/schema.go`:

```go
const schemaVersion = 2
```

În `schemaSQL`, în `CREATE TABLE IF NOT EXISTS document_rows`, după
`pret_vanzare REAL NOT NULL DEFAULT 0`, adaugă o virgulă pe linia aceea și apoi:

```sql
  valoare_vanzare_impusa REAL
```

Coloana este nullable și fără `DEFAULT`: `NULL` înseamnă „valoarea se derivă ca
la orice rând”, iar un `0` scris acolo ar fi o afirmație complet diferită.

- [ ] **Step 4: Write the migration**

În `internal/store/schema.go`, înlocuiește primele două instrucțiuni din
`migrate` cu:

```go
	if _, err := db.Exec(schemaSQL); err != nil {
		return err
	}
	if err := adaugaValoareVanzareImpusa(db); err != nil {
		return err
	}
	if _, err := db.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, schemaVersion)); err != nil {
		return err
	}
```

și adaugă, sub `migrate`:

```go
// adaugaValoareVanzareImpusa brings a version 1 file up to the version 2
// shape. It is the project's first real migration: the application has
// shipped, so the column is added to the table that exists rather than the
// table being dropped and rebuilt.
//
// The guard is the column's absence, not the version number. A brand new
// database already has the column — schemaSQL just created it — while its
// user_version is still 0, so a version-driven migration would try to add it a
// second time and fail on every first start.
func adaugaValoareVanzareImpusa(db *sql.DB) error {
	are, err := areColoana(db, "document_rows", "valoare_vanzare_impusa")
	if err != nil {
		return err
	}
	if are {
		return nil
	}
	if _, err := db.Exec(
		`ALTER TABLE document_rows ADD COLUMN valoare_vanzare_impusa REAL`,
	); err != nil {
		return fmt.Errorf("adaugare coloana valoare_vanzare_impusa: %w", err)
	}
	return nil
}

// areColoana reports whether a table already has a column.
func areColoana(db *sql.DB, tabel, coloana string) (bool, error) {
	var n int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`, tabel, coloana,
	).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("citire forma tabelului %s: %w", tabel, err)
	}
	return n > 0, nil
}
```

Verifică importurile din `schema.go`: are nevoie de `database/sql` și `fmt`.

`pragma_table_info(?)` este funcția-tabel a lui SQLite, disponibilă din 3.16 și
singurul apel din tot planul care ar putea să se comporte altfel sub driver.
Dacă `areColoana` dă eroare de sintaxă, înlocuiește interogarea cu un
`PRAGMA table_info(document_rows)` parcurs cu `db.Query` — numele coloanei este
al doilea câmp al fiecărui rând — și păstrează aceeași semnătură. Aceeași notă
este valabilă pentru `verificaForma` din Task 5.

- [ ] **Step 5: Read and write the column**

În `internal/store/documents.go`, în `randuri`, extinde interogarea și scanarea:

```go
	rows, err := s.db.Query(
		`SELECT id, product_id, pozitie, denumire, um, cantitate, pret_fara_tva,
		        cota_tva, pret_vanzare, valoare_vanzare_impusa
		 FROM document_rows WHERE document_id = ? ORDER BY pozitie, id`, documentID,
	)
```

```go
	for rows.Next() {
		var r model.Rand
		var productID sql.NullInt64
		var impusa sql.NullFloat64
		if err := rows.Scan(&r.ID, &productID, &r.Pozitie, &r.Denumire, &r.UM,
			&r.Cantitate, &r.PretFaraTVA, &r.CotaTVA, &r.PretVanzare, &impusa); err != nil {
			return nil, fmt.Errorf("citire rand: %w", err)
		}
		if productID.Valid {
			id := productID.Int64
			r.ProductID = &id
		}
		if impusa.Valid {
			v := impusa.Float64
			r.ValoareVanzareImpusa = &v
		}
		out = append(out, r)
	}
```

Și în `SaveDocument`, în bucla de inserare:

```go
		var productID any
		if r.ProductID != nil {
			productID = *r.ProductID
		}
		var impusa any
		if r.ValoareVanzareImpusa != nil {
			impusa = *r.ValoareVanzareImpusa
		}
		res, err := tx.Exec(
			`INSERT INTO document_rows (document_id, product_id, pozitie, denumire, um,
			        cantitate, pret_fara_tva, cota_tva, pret_vanzare, valoare_vanzare_impusa)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			doc.ID, productID, r.Pozitie, r.Denumire, r.UM, r.Cantitate,
			r.PretFaraTVA, r.CotaTVA, r.PretVanzare, impusa,
		)
```

`any` nil se scrie ca `NULL`, exact ca `productID` mai sus.

- [ ] **Step 6: Run the tests to verify they pass**

Run: `go test ./internal/store/ -v`
Expected: PASS, inclusiv toate testele existente.

- [ ] **Step 7: Commit**

```bash
git add internal/store/schema.go internal/store/documents.go internal/store/schema_test.go
git commit -m "Migrare la schema v2: coloana valoare_vanzare_impusa"
```

---

### Task 4: Rândurile importate stau la finalul tabelului

**Files:**
- Modify: `internal/store/documents.go` (`SaveDocument`)
- Test: `internal/store/documents_test.go`

**Interfaces:**
- Consumes: `model.Rand.ValoareVanzareImpusa` din Task 1, persistența din Task 3
- Produces: `SaveDocument` întoarce și scrie rândurile cu cele importate la
  final, ordinea relativă din fiecare grup păstrată.

- [ ] **Step 1: Write the failing test**

Adaugă în `internal/store/documents_test.go`:

```go
func TestRandurileImportateAjungLaFinal(t *testing.T) {
	// Un proces verbal adaugat, apoi inca un produs tastat manual: pe hartie
	// randul importat trebuie sa ramana ultimul, indiferent in ce ordine le-a
	// pus formularul.
	s := deschide(t)
	impusa := 2501.35
	doc := model.Document{
		Nr: 1, Data: "2026-09-09", Unitate: "S.C. Largiana Carn S.R.L.",
		Randuri: []model.Rand{
			{Denumire: "Pulpa fara os", UM: "Kg.", Cantitate: 10, PretFaraTVA: 20,
				CotaTVA: 11, PretVanzare: 30},
			{Denumire: "Carcasa", UM: "Kg.", Cantitate: 162.2, PretFaraTVA: 12.30,
				CotaTVA: 11, PretVanzare: 15.42, ValoareVanzareImpusa: &impusa},
			{Denumire: "Oua", UM: "Buc.", Cantitate: 30, PretFaraTVA: 0.9,
				CotaTVA: 11, PretVanzare: 1.5},
		},
	}

	salvat, err := s.SaveDocument(doc)
	if err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}

	vrem := []string{"Pulpa fara os", "Oua", "Carcasa"}
	for i, denumire := range vrem {
		if salvat.Randuri[i].Denumire != denumire {
			t.Errorf("documentul intors, randul %d = %q, vrem %q",
				i, salvat.Randuri[i].Denumire, denumire)
		}
		if salvat.Randuri[i].Pozitie != i {
			t.Errorf("randul %d are pozitia %d", i, salvat.Randuri[i].Pozitie)
		}
	}

	citit, err := s.GetDocument(salvat.ID)
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}
	for i, denumire := range vrem {
		if citit.Randuri[i].Denumire != denumire {
			t.Errorf("citit din baza, randul %d = %q, vrem %q",
				i, citit.Randuri[i].Denumire, denumire)
		}
	}
}

func TestOrdineaDintreRandurileImportateSePastreaza(t *testing.T) {
	s := deschide(t)
	unu, doi := 100.0, 200.0
	doc := model.Document{
		Nr: 1, Data: "2026-09-09", Unitate: "S.C. Largiana Carn S.R.L.",
		Randuri: []model.Rand{
			{Denumire: "Carcasa porc", UM: "Kg.", Cantitate: 1, PretFaraTVA: 1,
				CotaTVA: 11, PretVanzare: 1, ValoareVanzareImpusa: &unu},
			{Denumire: "Oua", UM: "Buc.", Cantitate: 30, PretFaraTVA: 0.9,
				CotaTVA: 11, PretVanzare: 1.5},
			{Denumire: "Pulpa vita", UM: "Kg.", Cantitate: 1, PretFaraTVA: 1,
				CotaTVA: 11, PretVanzare: 1, ValoareVanzareImpusa: &doi},
		},
	}

	salvat, err := s.SaveDocument(doc)
	if err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}
	vrem := []string{"Oua", "Carcasa porc", "Pulpa vita"}
	for i, denumire := range vrem {
		if salvat.Randuri[i].Denumire != denumire {
			t.Errorf("randul %d = %q, vrem %q", i, salvat.Randuri[i].Denumire, denumire)
		}
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/store/ -run "Randurile|Ordinea" -v`
Expected: FAIL — rândul „Carcasa” rămâne pe poziția 1.

- [ ] **Step 3: Impose the order in SaveDocument**

În `internal/store/documents.go`, adaugă înainte de `func (s *Store) SaveDocument`:

```go
// asazaImportateleLaFinal returns the rows with every row imported from a
// proces verbal moved to the end, each group keeping the order it came in.
//
// The rule is imposed here rather than in the form because this is where
// position becomes durable: whatever the frontend sends, what gets written —
// and what the PDF later prints — has the imported rows last.
func asazaImportateleLaFinal(randuri []model.Rand) []model.Rand {
	out := make([]model.Rand, 0, len(randuri))
	for _, r := range randuri {
		if r.ValoareVanzareImpusa == nil {
			out = append(out, r)
		}
	}
	for _, r := range randuri {
		if r.ValoareVanzareImpusa != nil {
			out = append(out, r)
		}
	}
	return out
}
```

Și în `SaveDocument`, imediat înaintea lui
`if _, err := tx.Exec(`DELETE FROM document_rows WHERE document_id = ?`, ...)`:

```go
	doc.Randuri = asazaImportateleLaFinal(doc.Randuri)
```

Bucla de mai jos atribuie deja `r.Pozitie = i`, deci pozițiile ies corecte, iar
documentul întors poartă ordinea corectată înapoi în formular.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/store/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/store/documents.go internal/store/documents_test.go
git commit -m "Randurile importate din proces verbal stau la finalul notei"
```

---

### Task 5: Pachetul `internal/pvt` — deschidere read-only și listare

**Files:**
- Create: `internal/pvt/pvt.go`
- Create: `internal/pvt/fixture_test.go`
- Create: `internal/pvt/pvt_test.go`

**Interfaces:**
- Consumes: `calc.Round2` din `internal/calc`
- Produces:
  - `pvt.Sumar{ID int64; Nr int; Data, Gestiune string; Total float64}`
  - `pvt.Lista{Disponibil bool; Procese []Sumar}`
  - `pvt.DBPath() (string, error)`
  - `pvt.List() (Lista, error)` și forma testabilă `pvt.ListaDin(path string) (Lista, error)`
  - erorile `pvt.ErrIndisponibil`, `pvt.ErrVersiuneVeche`, `pvt.ErrForma`, `pvt.ErrJurnal`

`List` și `Import` (Task 7) sunt înveliși subțiri peste `ListaDin`/`ImportDin`,
care primesc calea. Fără asta, testele ar trebui să scrie în directorul real de
configurare al utilizatorului care rulează testele.

- [ ] **Step 1: Write the fixture builder**

Creează `internal/pvt/fixture_test.go`:

```go
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
```

Verifică cifrele documentului 1 înainte să mergi mai departe: `100 × 15 = 1500`
plus `Round2(50.0675 × 20) = 1001.35` dau `2501.35` — exact valoarea pe care o
așteaptă testele din Task 1, Task 7 și Task 10. Dacă schimbi fixtura, schimbă-le
și pe ele.

- [ ] **Step 2: Write the failing tests**

Creează `internal/pvt/pvt_test.go`:

```go
package pvt

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestListaCiteseProceseleVerbale(t *testing.T) {
	lista, err := ListaDin(fixtura(t))
	if err != nil {
		t.Fatalf("ListaDin: %v", err)
	}
	if !lista.Disponibil {
		t.Fatal("Disponibil = false pe o baza care exista")
	}
	if len(lista.Procese) != 2 {
		t.Fatalf("procese = %d, vrem 2", len(lista.Procese))
	}
	// Cel mai recent primul.
	if lista.Procese[0].Nr != 2 {
		t.Errorf("primul proces are nr = %d, vrem 2", lista.Procese[0].Nr)
	}
	if lista.Procese[0].Gestiune != "Magazin Centru" {
		t.Errorf("Gestiune = %q", lista.Procese[0].Gestiune)
	}
	if lista.Procese[1].Total != 2501.35 {
		t.Errorf("Total = %v, vrem 2501.35", lista.Procese[1].Total)
	}
	if lista.Procese[1].Data != "2026-09-01" {
		t.Errorf("Data = %q", lista.Procese[1].Data)
	}
}

func TestListaSpuneCaAplicatiaLipseste(t *testing.T) {
	lista, err := ListaDin(filepath.Join(t.TempDir(), "nu-exista.db"))
	if err != nil {
		t.Fatalf("ListaDin pe un fisier absent nu trebuie sa dea eroare: %v", err)
	}
	if lista.Disponibil {
		t.Error("Disponibil = true pentru un fisier care nu exista")
	}
	if len(lista.Procese) != 0 {
		t.Errorf("procese = %d, vrem 0", len(lista.Procese))
	}
}

func TestListaGoalaNuEAcelasiLucruCuAplicatiaLipsa(t *testing.T) {
	path := fixtura(t)
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	if _, err := db.Exec(`DELETE FROM documents`); err != nil {
		t.Fatalf("golire: %v", err)
	}
	db.Close()

	lista, err := ListaDin(path)
	if err != nil {
		t.Fatalf("ListaDin: %v", err)
	}
	if !lista.Disponibil {
		t.Error("Disponibil = false desi aplicatia e instalata")
	}
	if len(lista.Procese) != 0 {
		t.Errorf("procese = %d, vrem 0", len(lista.Procese))
	}
}

func TestRefuzaOBazaPreRelease(t *testing.T) {
	path := fixtura(t)
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	if _, err := db.Exec(`PRAGMA user_version = 4`); err != nil {
		t.Fatalf("user_version: %v", err)
	}
	db.Close()

	if _, err := ListaDin(path); !errors.Is(err, ErrVersiuneVeche) {
		t.Errorf("err = %v, vrem ErrVersiuneVeche", err)
	}
}

func TestRefuzaOBazaCareAPierdutOColoana(t *testing.T) {
	path := fixtura(t)
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	if _, err := db.Exec(`DROP TABLE document_iesire_rows`); err != nil {
		t.Fatalf("drop: %v", err)
	}
	db.Close()

	if _, err := ListaDin(path); !errors.Is(err, ErrForma) {
		t.Errorf("err = %v, vrem ErrForma", err)
	}
}

// amprenta is everything about a file that a write would disturb.
type amprentaFisier struct {
	marime   int64
	modificat string
	versiune int
	continut string
}

func amprenta(t *testing.T, path string) amprentaFisier {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	octeti, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()
	var versiune int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&versiune); err != nil {
		t.Fatalf("PRAGMA user_version: %v", err)
	}
	return amprentaFisier{
		marime:    info.Size(),
		modificat: info.ModTime().UTC().Format("2006-01-02T15:04:05.000000000Z"),
		versiune:  versiune,
		continut:  string(octeti),
	}
}

func TestListaNuAtingeFisierulCeluilaltProiect(t *testing.T) {
	// Testul care conteaza cel mai mult. Daca acest pachet ajunge vreodata sa
	// deschida baza sora altfel decat read-only, aplicatia aceea isi sterge
	// singura tot continutul la urmatoarea pornire.
	path := fixtura(t)
	inainte := amprenta(t, path)

	if _, err := ListaDin(path); err != nil {
		t.Fatalf("ListaDin: %v", err)
	}

	if dupa := amprenta(t, path); dupa != inainte {
		t.Errorf("fisierul s-a schimbat dupa o citire:\ninainte = %+v\ndupa    = %+v",
			inainte, dupa)
	}
}
```

- [ ] **Step 3: Run the tests to verify they fail**

Run: `go test ./internal/pvt/ -v`
Expected: FAIL la compilare — pachetul `pvt` nu există încă.

- [ ] **Step 4: Write the package**

Creează `internal/pvt/pvt.go`:

```go
// Package pvt reads the database of the sibling application "proces verbal de
// transare". It is the only place in this project that knows that application
// exists.
//
// It never writes to it. The connection is opened query_only, and store.Open
// is deliberately not used: that runs this project's migrate, which would
// stamp user_version = 1 onto the other project's file and create this
// project's tables inside it. The other application would then see a version
// below its own on its next start and drop every table it has.
package pvt

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"

	"nota-de-receptie/internal/calc"
)

// dirName is the sibling application's folder inside the user's config
// directory — on Windows %APPDATA%, the same place internal/appdir puts this
// application's own file.
const dirName = "proces-verbal-transare"

// versiuneMinima is the sibling's schema version this package knows how to
// read. Below it, the file was written by a pre-release build whose products
// table has no template_id and which has no templates table at all — a shape
// the other application itself drops and reseeds on sight.
const versiuneMinima = 5

var (
	ErrIndisponibil = errors.New("aplicatia Proces verbal de transare nu este instalata pe acest calculator, sau nu a fost pornita niciodata")
	ErrVersiuneVeche = errors.New("baza aplicatiei Proces verbal de transare are o forma mai veche, dinaintea primei versiuni publicate; deschide o data acea aplicatie si incearca din nou")
	ErrForma         = errors.New("baza aplicatiei Proces verbal de transare are alta forma decat cea asteptata; probabil a fost actualizata la o versiune mai noua")
	ErrJurnal        = errors.New("baza aplicatiei Proces verbal de transare a ramas cu un jurnal neincheiat dupa o inchidere fortata; deschide o data acea aplicatie si incearca din nou")
	ErrFaraIntrare   = errors.New("procesul verbal ales nu are niciun rand in tabelul \"ce intra\"")
)

// Sumar is one proces verbal as it appears in the picker.
type Sumar struct {
	ID       int64   `json:"id"`
	Nr       int     `json:"nr"`
	Data     string  `json:"data"` // ISO YYYY-MM-DD
	Gestiune string  `json:"gestiune"`
	// Total is what the document's "ce iese" table comes to with TVA — the
	// figure that will land on the notă, shown before the choice rather than
	// after it.
	Total float64 `json:"total"`
}

// Lista is what the picker needs to know.
//
// Disponibil rides along because "the other application is not installed" and
// "it is installed but has no proces verbal yet" are different things needing
// different sentences, and an empty slice alone cannot tell them apart.
type Lista struct {
	Disponibil bool    `json:"disponibil"`
	Procese    []Sumar `json:"procese"`
}

// DBPath is where the sibling application keeps its database. Unlike
// internal/appdir it does not create the directory: this project has no
// business creating anything inside another application's folder, and an
// absent directory is the answer, not a problem to fix.
func DBPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, dirName, "data.db"), nil
}

// List reads the sibling application's proces verbal documents.
func List() (Lista, error) {
	path, err := DBPath()
	if err != nil {
		return Lista{}, err
	}
	return ListaDin(path)
}

// ListaDin is List against an explicit file, so tests do not have to write
// into the config directory of whoever is running them.
func ListaDin(path string) (Lista, error) {
	db, err := deschide(path)
	if errors.Is(err, ErrIndisponibil) {
		return Lista{Disponibil: false, Procese: []Sumar{}}, nil
	}
	if err != nil {
		return Lista{}, err
	}
	defer db.Close()

	totaluri, err := totaluriIesire(db)
	if err != nil {
		return Lista{}, err
	}

	rows, err := db.Query(
		`SELECT id, nr, data, gestiune FROM documents ORDER BY data DESC, nr DESC, id DESC`,
	)
	if err != nil {
		return Lista{}, fmt.Errorf("citire procese verbale: %w", err)
	}
	defer rows.Close()

	out := []Sumar{}
	for rows.Next() {
		var s Sumar
		if err := rows.Scan(&s.ID, &s.Nr, &s.Data, &s.Gestiune); err != nil {
			return Lista{}, fmt.Errorf("citire proces verbal: %w", err)
		}
		s.Total = totaluri[s.ID]
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return Lista{}, fmt.Errorf("citire procese verbale: %w", err)
	}
	return Lista{Disponibil: true, Procese: out}, nil
}

// totaluriIesire is every document's "ce iese" total with TVA, by document id.
//
// The sum is worked out here rather than in SQL so it goes through the same
// Round2 the rest of this application uses: row by row, then the sum, exactly
// as the sibling's own calc.TotalsIesire does it.
func totaluriIesire(db *sql.DB) (map[int64]float64, error) {
	rows, err := db.Query(
		`SELECT document_id, cantitate, pret_cu_tva FROM document_iesire_rows`,
	)
	if err != nil {
		return nil, fmt.Errorf("citire tabel \"ce iese\": %w", err)
	}
	defer rows.Close()

	out := map[int64]float64{}
	for rows.Next() {
		var id int64
		var cantitate, pret float64
		if err := rows.Scan(&id, &cantitate, &pret); err != nil {
			return nil, fmt.Errorf("citire rand \"ce iese\": %w", err)
		}
		out[id] += calc.Round2(cantitate * pret)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("citire tabel \"ce iese\": %w", err)
	}
	for id, total := range out {
		out[id] = calc.Round2(total)
	}
	return out, nil
}

// deschide opens the sibling's file read-only and checks it is a shape this
// package can read. The caller closes the handle.
func deschide(path string) (*sql.DB, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, ErrIndisponibil
	}
	db, err := sql.Open("sqlite", path+"?_pragma=query_only(1)&_pragma=busy_timeout(3000)")
	if err != nil {
		return nil, fmt.Errorf("deschidere baza Proces verbal de transare: %w", err)
	}
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		db.Close()
		// A file left with a hot journal by a forced shutdown cannot be rolled
		// forward by a read-only connection, and SQLite says so rather than
		// reading stale pages. Opening the other application once fixes it.
		if strings.Contains(strings.ToUpper(err.Error()), "READONLY") {
			return nil, ErrJurnal
		}
		return nil, fmt.Errorf("conectare la baza Proces verbal de transare: %w", err)
	}
	if err := verificaForma(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// verificaForma refuses a database this package cannot read correctly, which
// is better than reading it wrongly onto an accounting document.
func verificaForma(db *sql.DB) error {
	var versiune int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&versiune); err != nil {
		return fmt.Errorf("citire versiune baza Proces verbal de transare: %w", err)
	}
	if versiune < versiuneMinima {
		return ErrVersiuneVeche
	}
	necesare := map[string][]string{
		"documents":             {"id", "nr", "data", "gestiune"},
		"document_intrare_rows": {"document_id", "pozitie", "denumire", "um", "cantitate", "pret_fara_tva", "cota_tva"},
		"document_iesire_rows":  {"document_id", "cantitate", "pret_cu_tva"},
	}
	for tabel, coloane := range necesare {
		for _, coloana := range coloane {
			var n int
			err := db.QueryRow(
				`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`, tabel, coloana,
			).Scan(&n)
			if err != nil {
				return fmt.Errorf("citire forma bazei Proces verbal de transare: %w", err)
			}
			if n == 0 {
				return ErrForma
			}
		}
	}
	return nil
}
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `go test ./internal/pvt/ -v`
Expected: PASS, inclusiv `TestListaNuAtingeFisierulCeluilaltProiect`.

- [ ] **Step 6: Commit**

```bash
git add internal/pvt/
git commit -m "Pachetul pvt: citire read-only din baza aplicatiei surori"
```

---

### Task 6: Împărțirea proporțională a totalului

**Files:**
- Create: `internal/pvt/cote.go`
- Create: `internal/pvt/cote_test.go`

**Interfaces:**
- Consumes: `calc.Round2`
- Produces: `pvt.Cote(valori []float64, total float64) []float64`, cu garanția
  `suma(rezultat) == Round2(total)` exact.

- [ ] **Step 1: Write the failing tests**

Creează `internal/pvt/cote_test.go`:

```go
package pvt

import (
	"math"
	"testing"

	"nota-de-receptie/internal/calc"
)

func aproape(t *testing.T, nume string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("%s = %v, vrem %v", nume, got, want)
	}
}

func suma(v []float64) float64 {
	var s float64
	for _, x := range v {
		s += x
	}
	return calc.Round2(s)
}

func TestCoteCuUnSingurRand(t *testing.T) {
	// Cazul obisnuit: o carcasa, tot totalul pe randul ei.
	got := Cote([]float64{1995.06}, 2501.35)
	if len(got) != 1 {
		t.Fatalf("cote = %d, vrem 1", len(got))
	}
	aproape(t, "cota", got[0], 2501.35)
}

func TestCoteImpartProportional(t *testing.T) {
	got := Cote([]float64{300, 100}, 400)
	aproape(t, "cota[0]", got[0], 300)
	aproape(t, "cota[1]", got[1], 100)
	aproape(t, "suma", suma(got), 400)
}

func TestCoteDauSumaExactaCandImpartireaNuIese(t *testing.T) {
	// 100 impartit in trei parti egale nu se poate scrie in bani. Restul
	// merge la randul cel mai mare, si suma trebuie sa fie exact 100.
	got := Cote([]float64{100, 100, 100}, 100)
	aproape(t, "suma", suma(got), 100)
	aproape(t, "cota[1]", got[1], 33.33)
	aproape(t, "cota[2]", got[2], 33.33)
	aproape(t, "cota[0]", got[0], 33.34)
}

func TestRestulMergeLaRandulCelMaiMare(t *testing.T) {
	got := Cote([]float64{1, 1000}, 100)
	aproape(t, "suma", suma(got), 100)
	// Randul mic primeste cota lui rotunjita; restul cade pe cel mare.
	aproape(t, "cota[0]", got[0], 0.1)
	aproape(t, "cota[1]", got[1], 99.9)
}

func TestCoteCuValoriZero(t *testing.T) {
	// Intrari trecute cu pretul zero: nu exista proportie de urmat, si tot
	// totalul merge pe primul rand in loc sa se piarda.
	got := Cote([]float64{0, 0}, 50)
	aproape(t, "cota[0]", got[0], 50)
	aproape(t, "cota[1]", got[1], 0)
	aproape(t, "suma", suma(got), 50)
}

func TestCoteFaraValori(t *testing.T) {
	if got := Cote(nil, 100); len(got) != 0 {
		t.Errorf("cote = %v, vrem gol", got)
	}
}

func TestCoteCuTotalZero(t *testing.T) {
	got := Cote([]float64{10, 20}, 0)
	aproape(t, "suma", suma(got), 0)
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/pvt/ -run Cote -v`
Expected: FAIL la compilare — `undefined: Cote`.

- [ ] **Step 3: Write Cote**

Creează `internal/pvt/cote.go`:

```go
package pvt

import "nota-de-receptie/internal/calc"

// Cote splits total across len(valori) shares, in proportion to valori, so
// that the shares add up to exactly Round2(total).
//
// Shares rounded independently miss the total by a ban or two, and here the
// total is the whole point: it is what the proces verbal came to, and the notă
// has to carry that figure and not a near miss. So every share but one is the
// rounded quotient, and the remaining one takes whatever is left. That one is
// the largest, where a couple of bani weigh proportionally least — on a 2500
// lei carcass they vanish; on a 30 lei row they would show.
//
// With all values zero there is no proportion to follow, and the whole total
// goes on the first share rather than being lost.
func Cote(valori []float64, total float64) []float64 {
	out := make([]float64, len(valori))
	if len(valori) == 0 {
		return out
	}
	total = calc.Round2(total)

	var sumaValori float64
	for _, v := range valori {
		sumaValori += v
	}
	if sumaValori == 0 {
		out[0] = total
		return out
	}

	mare := indexulCelMaiMare(valori)
	rest := total
	for i, v := range valori {
		if i == mare {
			continue
		}
		out[i] = calc.Round2(total * v / sumaValori)
		rest = calc.Round2(rest - out[i])
	}
	out[mare] = rest
	return out
}

// indexulCelMaiMare is the position of the largest value, the first of them if
// several tie.
func indexulCelMaiMare(valori []float64) int {
	mare := 0
	for i, v := range valori {
		if v > valori[mare] {
			mare = i
		}
	}
	return mare
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/pvt/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/pvt/cote.go internal/pvt/cote_test.go
git commit -m "Impartirea proportionala a totalului ce iese, cu suma exacta"
```

---

### Task 7: `pvt.Import` — un proces verbal devine rânduri de notă

**Files:**
- Modify: `internal/pvt/pvt.go`
- Modify: `internal/pvt/pvt_test.go`

**Interfaces:**
- Consumes: `Cote` din Task 6, `deschide`/`totaluriIesire` din Task 5,
  `model.Rand` din Task 1
- Produces: `pvt.Import(id int64) ([]model.Rand, error)` și forma testabilă
  `pvt.ImportDin(path string, id int64) ([]model.Rand, error)`

- [ ] **Step 1: Write the failing tests**

Adaugă în `internal/pvt/pvt_test.go`:

```go
func TestImportulUneiCarcase(t *testing.T) {
	randuri, err := ImportDin(fixtura(t), 1)
	if err != nil {
		t.Fatalf("ImportDin: %v", err)
	}
	if len(randuri) != 1 {
		t.Fatalf("randuri = %d, vrem 1", len(randuri))
	}
	r := randuri[0]
	if r.Denumire != "Carcasa" {
		t.Errorf("Denumire = %q, vrem \"Carcasa\"", r.Denumire)
	}
	if r.UM != "Kg." {
		t.Errorf("UM = %q, vrem \"Kg.\" (normalizat din \"Kg\")", r.UM)
	}
	if r.ProductID != nil {
		t.Error("randul importat nu trebuie sa fie legat de un produs din catalog")
	}
	aproape(t, "Cantitate", r.Cantitate, 162.2)
	aproape(t, "PretFaraTVA", r.PretFaraTVA, 12.30)
	aproape(t, "CotaTVA", r.CotaTVA, 11)
	if r.ValoareVanzareImpusa == nil {
		t.Fatal("ValoareVanzareImpusa = nil")
	}
	aproape(t, "ValoareVanzareImpusa", *r.ValoareVanzareImpusa, 2501.35)
	// 2501.35 / 162.2 = 15.4214..., informativ, rotunjit ca tot formularul.
	aproape(t, "PretVanzare", r.PretVanzare, 15.42)
}

func TestImportulImparteIntreDouaRanduriDeIntrare(t *testing.T) {
	randuri, err := ImportDin(fixtura(t), 2)
	if err != nil {
		t.Fatalf("ImportDin: %v", err)
	}
	if len(randuri) != 2 {
		t.Fatalf("randuri = %d, vrem 2", len(randuri))
	}
	if randuri[0].Denumire != "Carcasa" || randuri[1].Denumire != "Pulpa vita Angus" {
		t.Errorf("ordinea randurilor de intrare nu s-a pastrat: %q, %q",
			randuri[0].Denumire, randuri[1].Denumire)
	}
	var total float64
	for _, r := range randuri {
		if r.ValoareVanzareImpusa == nil {
			t.Fatalf("randul %q nu are valoare impusa", r.Denumire)
		}
		total += *r.ValoareVanzareImpusa
	}
	// Cele doua intrari sunt identice ca valoare, deci impartirea e la egalitate.
	aproape(t, "total impartit", calc.Round2(total), 100)
	aproape(t, "cota[0]", *randuri[0].ValoareVanzareImpusa, 50)
	aproape(t, "cota[1]", *randuri[1].ValoareVanzareImpusa, 50)
}

func TestImportulRefuzaUnProcesVerbalFaraIntrari(t *testing.T) {
	path := fixtura(t)
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	if _, err := db.Exec(`DELETE FROM document_intrare_rows WHERE document_id = 1`); err != nil {
		t.Fatalf("golire intrare: %v", err)
	}
	db.Close()

	if _, err := ImportDin(path, 1); !errors.Is(err, ErrFaraIntrare) {
		t.Errorf("err = %v, vrem ErrFaraIntrare", err)
	}
}

func TestImportulRefuzaUnIdInexistent(t *testing.T) {
	if _, err := ImportDin(fixtura(t), 99); err == nil {
		t.Error("un id inexistent trebuie sa dea eroare")
	}
}

func TestImportulNuAtingeFisierulCeluilaltProiect(t *testing.T) {
	path := fixtura(t)
	inainte := amprenta(t, path)

	if _, err := ImportDin(path, 1); err != nil {
		t.Fatalf("ImportDin: %v", err)
	}

	if dupa := amprenta(t, path); dupa != inainte {
		t.Errorf("fisierul s-a schimbat dupa un import:\ninainte = %+v\ndupa    = %+v",
			inainte, dupa)
	}
}
```

Adaugă `"nota-de-receptie/internal/calc"` la importurile din `pvt_test.go`.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/pvt/ -run Import -v`
Expected: FAIL la compilare — `undefined: ImportDin`.

- [ ] **Step 3: Write Import**

Adaugă în `internal/pvt/pvt.go` (și `"nota-de-receptie/internal/model"` la
importuri):

```go
// intrare is one row of a proces verbal's "ce intra" table — the only shape
// this package reads out of the sibling's documents.
type intrare struct {
	Denumire    string
	UM          string
	Cantitate   float64
	PretFaraTVA float64
	PretCuTVA   float64
	CotaTVA     float64
}

// Import turns one proces verbal into rows ready to append to a notă.
//
// The figures are copied, not linked: from here on the notă owns them, and
// editing the proces verbal later leaves a filed document alone — the same
// rule its rows already follow towards the product catalogue.
func Import(id int64) ([]model.Rand, error) {
	path, err := DBPath()
	if err != nil {
		return nil, err
	}
	return ImportDin(path, id)
}

// ImportDin is Import against an explicit file.
func ImportDin(path string, id int64) ([]model.Rand, error) {
	db, err := deschide(path)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var nr int
	if err := db.QueryRow(`SELECT nr FROM documents WHERE id = ?`, id).Scan(&nr); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("procesul verbal cerut nu mai exista")
		}
		return nil, fmt.Errorf("citire proces verbal: %w", err)
	}

	intrari, err := citesteIntrare(db, id)
	if err != nil {
		return nil, err
	}
	if len(intrari) == 0 {
		return nil, ErrFaraIntrare
	}

	totaluri, err := totaluriIesire(db)
	if err != nil {
		return nil, err
	}
	cote := coteleIntrarilor(intrari, totaluri[id])

	out := make([]model.Rand, len(intrari))
	for i, in := range intrari {
		cota := cote[i]
		var pretVanzare float64
		if in.Cantitate != 0 {
			pretVanzare = calc.Round2(cota / in.Cantitate)
		}
		valoare := cota
		out[i] = model.Rand{
			Denumire:             denumireSau(in.Denumire, nr),
			UM:                   umNota(in.UM),
			Cantitate:            in.Cantitate,
			PretFaraTVA:          in.PretFaraTVA,
			CotaTVA:              in.CotaTVA,
			PretVanzare:          pretVanzare,
			ValoareVanzareImpusa: &valoare,
		}
	}
	return out, nil
}

// coteleIntrarilor splits the "ce iese" total across the "ce intra" rows by
// what each of them cost. Rows all priced at zero carry no proportion, and the
// quantities are the next best thing to share it out by.
func coteleIntrarilor(intrari []intrare, total float64) []float64 {
	valori := make([]float64, len(intrari))
	var suma float64
	for i, in := range intrari {
		valori[i] = calc.Round2(in.Cantitate * in.PretCuTVA)
		suma += valori[i]
	}
	if suma == 0 {
		for i, in := range intrari {
			valori[i] = in.Cantitate
		}
	}
	return Cote(valori, total)
}

func citesteIntrare(db *sql.DB, id int64) ([]intrare, error) {
	rows, err := db.Query(
		`SELECT denumire, um, cantitate, pret_fara_tva, pret_cu_tva, cota_tva
		 FROM document_intrare_rows WHERE document_id = ? ORDER BY pozitie, id`, id,
	)
	if err != nil {
		return nil, fmt.Errorf("citire tabel \"ce intra\": %w", err)
	}
	defer rows.Close()

	out := []intrare{}
	for rows.Next() {
		var in intrare
		if err := rows.Scan(&in.Denumire, &in.UM, &in.Cantitate,
			&in.PretFaraTVA, &in.PretCuTVA, &in.CotaTVA); err != nil {
			return nil, fmt.Errorf("citire rand \"ce intra\": %w", err)
		}
		out = append(out, in)
	}
	return out, rows.Err()
}

// umNota translates the sibling's unit into the two this form writes. The
// other application types "Kg"; this one writes "Kg." everywhere, and a notă
// carrying both spellings would be a notă with two units.
func umNota(um string) string {
	switch strings.TrimSpace(um) {
	case "Kg", "kg", "Kg.":
		return "Kg."
	case "Buc", "buc", "Buc.":
		return "Buc."
	default:
		return um
	}
}

// denumireSau falls back to the document's number when the "ce intra" row was
// left unnamed — that column allows an empty string, and a blank row on a
// reception is worse than a plain one.
func denumireSau(denumire string, nr int) string {
	if strings.TrimSpace(denumire) == "" {
		return fmt.Sprintf("Proces verbal nr. %d", nr)
	}
	return denumire
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/pvt/ -v`
Expected: PASS, inclusiv ambele teste de „fișierul nu s-a schimbat”.

- [ ] **Step 5: Commit**

```bash
git add internal/pvt/pvt.go internal/pvt/pvt_test.go
git commit -m "pvt.Import: un proces verbal devine randuri de nota de receptie"
```

---

### Task 8: Legăturile Wails

**Files:**
- Modify: `app.go`
- Modify: `frontend/wailsjs/go/main/App.js`
- Modify: `frontend/wailsjs/go/main/App.d.ts`
- Modify: `frontend/wailsjs/go/models.ts`
- Modify: `frontend/src/api.ts`

**Interfaces:**
- Consumes: `pvt.List`, `pvt.Import` din Task 5 și 7
- Produces: `ListProceseVerbale(): Promise<pvt.Lista>` și
  `ImportProcesVerbal(id: number): Promise<model.Rand[]>` în
  `frontend/src/api.ts`, plus tipurile `ProcesVerbalSumar` și `ProceseVerbale`.

- [ ] **Step 1: Add the bindings in Go**

În `app.go`, adaugă `"nota-de-receptie/internal/pvt"` la importuri și, după
`ListFurnizori`/`DeleteFurnizor`:

```go
// ListProceseVerbale lists the proces verbal documents of the sibling
// application, or reports that it is not installed.
func (a *App) ListProceseVerbale() (pvt.Lista, error) { return pvt.List() }

// ImportProcesVerbal turns one proces verbal into rows ready to append to the
// note on screen.
func (a *App) ImportProcesVerbal(id int64) ([]model.Rand, error) { return pvt.Import(id) }
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./... && go test ./...`
Expected: fără erori.

- [ ] **Step 3: Update the generated bindings**

Fișierele din `frontend/wailsjs` sunt generate de Wails dar comise. Dacă ai
`wails` instalat, rulează `wails generate module` și sari peste editarea
manuală. Altfel, adaugă exact:

În `frontend/wailsjs/go/main/App.js`, în ordine alfabetică (după `ImportProcesVerbal`
vine înainte de `ListDocuments`):

```js
export function ImportProcesVerbal(arg1) {
  return window['go']['main']['App']['ImportProcesVerbal'](arg1);
}

export function ListProceseVerbale() {
  return window['go']['main']['App']['ListProceseVerbale']();
}
```

În `frontend/wailsjs/go/main/App.d.ts`, schimbă prima linie de import și adaugă
declarațiile:

```ts
import {model} from '../models';
import {pvt} from '../models';

export function ImportProcesVerbal(arg1:number):Promise<Array<model.Rand>>;

export function ListProceseVerbale():Promise<pvt.Lista>;
```

În `frontend/wailsjs/go/models.ts`, adaugă un namespace nou după închiderea lui
`export namespace model { ... }`:

```ts
export namespace pvt {
	
	export class Sumar {
	    id: number;
	    nr: number;
	    data: string;
	    gestiune: string;
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new Sumar(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.nr = source["nr"];
	        this.data = source["data"];
	        this.gestiune = source["gestiune"];
	        this.total = source["total"];
	    }
	}
	export class Lista {
	    disponibil: boolean;
	    procese: Sumar[];
	
	    static createFrom(source: any = {}) {
	        return new Lista(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.disponibil = source["disponibil"];
	        this.procese = this.convertValues(source["procese"], Sumar);
	    }
	
	    convertValues(a: any, classs: any, asMap: boolean = false): any {
	        if (!a) {
	            return a;
	        }
	        if (a.slice && a.map) {
	            return (a as any[]).map((elem) => this.convertValues(elem, classs));
	        } else if ("object" === typeof a) {
	            if (asMap) {
	                for (const key of Object.keys(a)) {
	                    a[key] = new classs(a[key]);
	                }
	                return a;
	            }
	            return new classs(a);
	        }
	        return a;
	    }
	}

}
```

- [ ] **Step 4: Export them from api.ts**

În `frontend/src/api.ts`, adaugă `ImportProcesVerbal` și `ListProceseVerbale` în
ambele liste (importul din `../wailsjs/go/main/App` și blocul `export {}`,
păstrând ordinea alfabetică), adaugă `pvt` la importul de modele și tipurile:

```ts
import { model, pvt } from '../wailsjs/go/models';

export type ProcesVerbalSumar = pvt.Sumar;
export type ProceseVerbale = pvt.Lista;
```

- [ ] **Step 5: Verify the typecheck passes**

Run: `cd frontend && npm run build && npm test`
Expected: `tsc` fără erori, testele trec.

- [ ] **Step 6: Commit**

```bash
git add app.go frontend/wailsjs frontend/src/api.ts
git commit -m "Legaturi Wails pentru listarea si importul proceselor verbale"
```

---

### Task 9: Butonul, lista de alegere și rândurile blocate

**Files:**
- Modify: `frontend/src/nota.ts` (helper `esteImportat`)
- Modify: `frontend/src/nota.test.ts`
- Modify: `frontend/src/dialog.ts` (`showPicker`)
- Modify: `frontend/src/views/document.ts`
- Modify: `frontend/src/style.css`

**Interfaces:**
- Consumes: `ListProceseVerbale`, `ImportProcesVerbal`, `ProcesVerbalSumar` din
  Task 8; `valoriRand` din Task 2
- Produces: `nota.esteImportat(r: Rand): boolean`;
  `dialog.showPicker(message, optiuni): Promise<string | undefined>`

- [ ] **Step 1: Write the failing test for the helper**

Adaugă în `frontend/src/nota.test.ts` (adaugă `esteImportat` la importul din
`./nota`):

```ts
describe('esteImportat', () => {
  // The cast is the one nota.ts already uses for a hand-built row: the
  // generated Rand is a class, and `null` is not in its declared type.
  const rand = (impusa?: number | null): Rand =>
    ({
      id: 0, productId: undefined, pozitie: 0, denumire: 'Carcasa', um: 'Kg.',
      cantitate: 162.2, pretFaraTva: 12.3, cotaTva: 11, pretVanzare: 15.42,
      valoareVanzareImpusa: impusa,
    }) as unknown as Rand;

  it('recunoaste un rand venit dintr-un proces verbal', () => {
    expect(esteImportat(rand(2501.35))).toBe(true);
  });

  it('recunoaste zero impus ca tot rand importat', () => {
    expect(esteImportat(rand(0))).toBe(true);
  });

  it('trateaza null si undefined ca rand obisnuit', () => {
    expect(esteImportat(rand(null))).toBe(false);
    expect(esteImportat(rand())).toBe(false);
  });
});
```

`Rand` se importă cu `import type { Rand } from './api';` dacă fișierul nu îl
are deja.

- [ ] **Step 2: Run it to verify it fails**

Run: `cd frontend && npx vitest run src/nota.test.ts`
Expected: FAIL — `esteImportat` nu este exportat din `./nota`.

- [ ] **Step 3: Write the helper**

În `frontend/src/nota.ts`, după `randDinProdus`:

```ts
/**
 * Whether the row came from a proces verbal de transare.
 *
 * The imposed selling value is the only marker there is, and it is enough: a
 * row has one exactly when it was imported. Zero is a real imposed value — a
 * proces verbal yielding nothing but waste — so the check is against null and
 * undefined, not against falsiness.
 */
export function esteImportat(r: Rand): boolean {
  const impusa = (r as { valoareVanzareImpusa?: number | null }).valoareVanzareImpusa;
  return impusa !== undefined && impusa !== null;
}
```

- [ ] **Step 4: Run it to verify it passes**

Run: `cd frontend && npx vitest run src/nota.test.ts`
Expected: PASS.

- [ ] **Step 5: Add the picker dialog**

În `frontend/src/dialog.ts`, dă lui `openModal` un al treilea parametru
opțional și inserează-l între text și butoane:

```ts
function openModal(
  message: string,
  buttons: HTMLButtonElement[],
  continut?: HTMLElement,
): Promise<string> {
```

și, în corpul funcției, după `dialog.appendChild(text);`:

```ts
    if (continut !== undefined) dialog.appendChild(continut);
```

Apoi, la finalul fișierului:

```ts
/** One choice in a picker: what it returns, and the two lines it shows. */
export interface OptiunePicker {
  valoare: string;
  eticheta: string;
  detaliu: string;
}

/**
 * Asks the user to pick one of `optiuni`, and resolves with its `valoare`, or
 * undefined if they backed out. Esc and "Renunță" both close with an empty
 * returnValue, which is the same answer either way.
 */
export function showPicker(
  message: string,
  optiuni: OptiunePicker[],
): Promise<string | undefined> {
  const lista = document.createElement('div');
  lista.className = 'modal-lista';
  optiuni.forEach((o) => {
    const button = modalButton('', o.valoare, 'btn modal-optiune');
    const eticheta = document.createElement('span');
    eticheta.className = 'modal-optiune-eticheta';
    eticheta.textContent = o.eticheta;
    const detaliu = document.createElement('span');
    detaliu.className = 'modal-optiune-detaliu';
    detaliu.textContent = o.detaliu;
    button.append(eticheta, detaliu);
    lista.appendChild(button);
  });

  const renunta = modalButton('Renunță', '', 'btn');
  return openModal(message, [renunta], lista).then((v) => (v === '' ? undefined : v));
}
```

- [ ] **Step 6: Style the picker**

Adaugă la finalul lui `frontend/src/style.css`:

```css
/* Lista de alegere dintr-un modal (procesele verbale). */
.modal-lista {
  display: flex;
  flex-direction: column;
  gap: 6px;
  max-height: 50vh;
  overflow-y: auto;
  margin: 12px 0;
}

.modal-optiune {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 2px;
  text-align: left;
  width: 100%;
}

.modal-optiune-eticheta {
  font-weight: 600;
}

.modal-optiune-detaliu {
  font-size: 0.85em;
  opacity: 0.75;
}

/* Randurile venite dintr-un proces verbal: blocate, si vizibil asa. */
.tabel-produse tr.rand-importat input {
  background: rgba(0, 0, 0, 0.04);
  cursor: default;
}
```

- [ ] **Step 7: Wire the form**

În `frontend/src/views/document.ts`:

Adaugă la importuri `ImportProcesVerbal` și `ListProceseVerbale` din `../api`,
`showPicker` din `../dialog`, și `esteImportat` din `../nota`.

Butonul, în `.table-actions`:

```html
        <button class="btn" id="add-rand">+ Adaugă rând</button>
        <button class="btn" id="add-pv">+ Adaugă din proces verbal</button>
```

În `tabelProduse()`, la începutul construirii fiecărui rând, calculează starea și
folosește-o pe casete:

```ts
      const blocat = esteImportat(r);
      const ro = blocat ? ' readonly' : '';
```

`<tr>`-ul primește clasa: `<tr data-rand="${i}" class="${blocat ? 'rand-importat' : ''}">`,
iar fiecare `<input>` din rând primește `${ro}` înaintea lui `/>`. Selectorul de
U/M, dacă e un `<select>`, primește `${blocat ? ' disabled' : ''}`.

În `wireEvents()`, sari peste combo pentru rândurile blocate:

```ts
      const denumire = rand.querySelector<HTMLInputElement>('input.denumire')!;
      if (!esteImportat(doc.randuri[index])) {
        denumire.addEventListener('input', () => deschideCombo(index, denumire));
        denumire.addEventListener('focus', () => deschideCombo(index, denumire));
        denumire.addEventListener('keydown', (e) => navigheazaCombo(e, index, denumire));
        denumire.addEventListener('blur', () =>
          window.setTimeout(() => {
            if (comboRand === index) inchideCombo();
          }, 150),
        );
      }
```

„+ Adaugă rând” inserează deasupra rândurilor importate:

```ts
    outlet.querySelector<HTMLButtonElement>('#add-rand')!.addEventListener('click', () => {
      // Imported rows stay at the end, so a row typed in now belongs above
      // them: the form shows what will be printed, with nothing jumping on
      // save.
      const primulImportat = doc.randuri.findIndex((r) => esteImportat(r));
      const pozitie = primulImportat === -1 ? doc.randuri.length : primulImportat;
      doc.randuri.splice(pozitie, 0, randGol(cotaImplicita));
      render();
      const inputuri = outlet.querySelectorAll<HTMLInputElement>('input.denumire');
      inputuri[pozitie]?.focus();
    });
```

Și handlerul nou, lângă el:

```ts
    outlet.querySelector<HTMLButtonElement>('#add-pv')!.addEventListener('click', async () => {
      let lista;
      try {
        lista = await ListProceseVerbale();
      } catch (err) {
        showError('Nu am putut citi procesele verbale', err);
        return;
      }
      if (!lista.disponibil) {
        await showAlert(
          'Nu am găsit aplicația „Proces verbal de transare” pe acest calculator. ' +
            'Instaleaz-o și deschide-o o dată, apoi încearcă din nou.',
        );
        return;
      }
      if (lista.procese.length === 0) {
        await showAlert('Aplicația „Proces verbal de transare” nu are încă niciun proces verbal salvat.');
        return;
      }

      const ales = await showPicker(
        'Alege procesul verbal de adăugat pe notă:',
        lista.procese.map((p) => ({
          valoare: String(p.id),
          eticheta: `Nr. ${p.nr} din ${formatDateRO(p.data)}`,
          detaliu: `${p.gestiune} — ${formatLei(p.total)}`,
        })),
      );
      if (ales === undefined) return;

      try {
        const randuri = await ImportProcesVerbal(Number(ales));
        doc.randuri.push(...randuri);
        render();
      } catch (err) {
        showError('Nu am putut adăuga procesul verbal', err);
      }
    });
```

- [ ] **Step 8: Verify the typecheck and tests pass**

Run: `cd frontend && npm run build && npm test`
Expected: `tsc` fără erori, toate testele trec.

- [ ] **Step 9: Commit**

```bash
git add frontend/src
git commit -m "Buton, lista de alegere si randuri blocate pentru procesele verbale"
```

---

### Task 10: PDF, documentație și verificarea finală

**Files:**
- Test: `internal/pdfdoc/pdf_test.go`
- Modify: `README.md`

**Interfaces:**
- Consumes: tot ce s-a construit până aici
- Produces: nimic nou — acest task confirmă că PDF-ul nu are nevoie de
  modificări și încheie lucrarea.

- [ ] **Step 1: Write the failing test**

Adaugă în `internal/pdfdoc/pdf_test.go`:

```go
func TestTabelulTipareteValoareaImpusaSiTotalulEi(t *testing.T) {
	// Randul importat dintr-un proces verbal isi poarta propria valoare la pret
	// de vanzare. Inmultirea cantitate x pret ar da 2501.12; pe hartie trebuie
	// sa apara totalul procesului verbal, si el trebuie sa intre ca atare in
	// randul TOTAL.
	impusa := 2501.35
	d := model.Document{
		Nr: 7, Data: "2026-09-09", Unitate: "S.C. Largiana Carn S.R.L.",
		Randuri: []model.Rand{
			{Pozitie: 0, Denumire: "Oua", UM: "Buc.", Cantitate: 30,
				PretFaraTVA: 0.9, CotaTVA: 11, PretVanzare: 1.5},
			{Pozitie: 1, Denumire: "Carcasa", UM: "Kg.", Cantitate: 162.2,
				PretFaraTVA: 12.30, CotaTVA: 11, PretVanzare: 15.42,
				ValoareVanzareImpusa: &impusa},
		},
	}

	celule := randCells(1, d.Randuri[1])
	if celule[8] != "2.501,35" {
		t.Errorf("valoarea la pret de vanzare = %q, vrem \"2.501,35\"", celule[8])
	}
	if celule[7] != "15,42" {
		t.Errorf("pretul de vanzare = %q, vrem \"15,42\"", celule[7])
	}

	total := randTotal(d)
	if total[8] != "2.546,35" {
		t.Errorf("totalul la pret de vanzare = %q, vrem \"2.546,35\"", total[8])
	}
}
```

(30 × 1,5 = 45 lei pe rândul de ouă, plus 2.501,35 = 2.546,35.)

- [ ] **Step 2: Run it**

Run: `go test ./internal/pdfdoc/ -run Impusa -v`
Expected: PASS din prima. `randCells` citește deja `calc.ValoriRand`, care de la
Task 1 întoarce valoarea impusă — acesta este testul care confirmă că
`internal/pdfdoc` nu are nevoie de nicio modificare. Dacă pică, nu repara
PDF-ul: înseamnă că Task 1 nu e complet.

- [ ] **Step 3: Document the feature**

Adaugă în `README.md`, după secțiunea „Teste”:

```markdown
## Import din „Proces verbal de transare”

Când carcasa de pe factură a fost transată, valoarea ei de vânzare se ia din
procesul verbal întocmit în aplicația soră, cu butonul **„+ Adaugă din proces
verbal”** din formularul notei.

Cele două aplicații nu comunică între ele: nota citește o singură dată fișierul
celeilalte, read-only, și copiază cifrele. Aplicația soră trebuie instalată și
pornită cel puțin o dată pe același cont de utilizator Windows, fiindcă baza ei
de date se creează la prima pornire:

- Windows: `%APPDATA%\proces-verbal-transare\data.db`
- macOS: `~/Library/Application Support/proces-verbal-transare/data.db`

Rândurile importate nu se pot edita — poartă valoarea din procesul verbal, care
înseamnă ceva doar alături de cantitatea și prețul din care a fost calculată —
și stau întotdeauna la finalul tabelului. Se pot șterge și reimporta.
```

- [ ] **Step 4: Run the whole suite**

Run: `make test`
Expected: `go test ./...` și `npm test` trec, toate.

- [ ] **Step 5: Run the application by hand**

Run: `wails dev`

Verifică, în ordine:
1. O notă existentă se deschide și arată la fel ca înainte (migrarea nu a stricat nimic).
2. „+ Adaugă din proces verbal” listează procesele verbale, cu totalul lor.
3. Un import adaugă rândul, blocat, la finalul tabelului, cu valoarea la preț de
   vânzare egală cu totalul din listă.
4. „+ Adaugă rând” după import inserează deasupra rândului importat.
5. Salvarea păstrează ordinea, iar PDF-ul tipărește aceleași cifre ca ecranul.
6. Redenumește temporar folderul aplicației surori și verifică mesajul „Nu am
   găsit aplicația…”, apoi pune-l la loc.

- [ ] **Step 6: Commit**

```bash
git add internal/pdfdoc/pdf_test.go README.md
git commit -m "Test PDF pentru valoarea impusa si documentatia importului"
```
