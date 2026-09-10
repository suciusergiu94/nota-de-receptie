package store

import (
	"database/sql"
	"fmt"
)

const schemaVersion = 2

const schemaSQL = `
CREATE TABLE IF NOT EXISTS settings (
  id           INTEGER PRIMARY KEY CHECK (id = 1),
  unitate_nume TEXT    NOT NULL DEFAULT '',
  next_nr      INTEGER NOT NULL DEFAULT 1,
  cota_tva     REAL    NOT NULL DEFAULT 11
);

CREATE TABLE IF NOT EXISTS products (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  denumire     TEXT    NOT NULL,
  um           TEXT    NOT NULL DEFAULT 'Kg.',
  pret_vanzare REAL    NOT NULL DEFAULT 0,
  cota_tva     REAL    NOT NULL DEFAULT 11,
  ordine       INTEGER NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_products_denumire
  ON products(denumire COLLATE NOCASE);

CREATE TABLE IF NOT EXISTS furnizori (
  id   INTEGER PRIMARY KEY AUTOINCREMENT,
  nume TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_furnizori_nume
  ON furnizori(nume COLLATE NOCASE);

CREATE TABLE IF NOT EXISTS documents (
  id                    INTEGER PRIMARY KEY AUTOINCREMENT,
  nr                    INTEGER NOT NULL,
  data                  TEXT    NOT NULL,
  unitate               TEXT    NOT NULL DEFAULT '',
  document_livrare      TEXT    NOT NULL DEFAULT '',
  document_livrare_nr   TEXT    NOT NULL DEFAULT '',
  document_livrare_data TEXT    NOT NULL DEFAULT '',
  furnizor              TEXT    NOT NULL DEFAULT '',
  created_at            TEXT    NOT NULL,
  updated_at            TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS document_rows (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  document_id   INTEGER NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  product_id    INTEGER          REFERENCES products(id)  ON DELETE SET NULL,
  pozitie       INTEGER NOT NULL,
  denumire      TEXT    NOT NULL,
  um            TEXT    NOT NULL DEFAULT 'Kg.',
  cantitate     REAL    NOT NULL DEFAULT 0,
  pret_fara_tva REAL    NOT NULL DEFAULT 0,
  cota_tva      REAL    NOT NULL DEFAULT 11,
  pret_vanzare  REAL    NOT NULL DEFAULT 0,
  valoare_vanzare_impusa REAL
);
CREATE INDEX IF NOT EXISTS idx_rows_document ON document_rows(document_id, pozitie);
`

// defaultUnitate is the company the paper form is printed for.
const defaultUnitate = "S.C. Largiana Carn S.R.L."

// defaultCotaTVA is the standard Romanian TVA rate for food, in percent.
const defaultCotaTVA = 11.0

// migrate brings the database to schemaVersion and writes the first-run
// settings row. It is safe to call on every startup: the row is written only
// when it is absent.
//
// There is only one schema version so far. When the shape has to change on a
// database that has shipped, this is where a real migration goes — never a
// drop-and-recreate, since by then the file holds documents nobody can retype.
func migrate(db *sql.DB) error {
	if _, err := db.Exec(schemaSQL); err != nil {
		return err
	}
	if err := adaugaValoareVanzareImpusa(db); err != nil {
		return err
	}
	if _, err := db.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, schemaVersion)); err != nil {
		return err
	}

	var seeded int
	if err := db.QueryRow(`SELECT COUNT(*) FROM settings WHERE id = 1`).Scan(&seeded); err != nil {
		return err
	}
	if seeded > 0 {
		return nil
	}
	_, err := db.Exec(
		`INSERT INTO settings (id, unitate_nume, next_nr, cota_tva) VALUES (1, ?, 1, ?)`,
		defaultUnitate, defaultCotaTVA,
	)
	return err
}

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
