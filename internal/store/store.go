// Package store is the SQLite persistence layer.
package store

import (
	"database/sql"
	"errors"
	"fmt"

	_ "modernc.org/sqlite"

	"nota-de-receptie/internal/model"
)

// ErrNotFound is returned when an id does not exist.
var ErrNotFound = errors.New("documentul nu a fost gasit")

// Store owns the database handle.
type Store struct {
	db *sql.DB
}

// Open opens (creating if needed) the SQLite file at path, applies the schema
// and writes the first-run settings row.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("deschidere bazei de date: %w", err)
	}
	// SQLite tolerates a single writer; one connection keeps writes serialized.
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("conectare la baza de date: %w", err)
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrare baza de date: %w", err)
	}
	return &Store{db: db}, nil
}

// Close releases the database handle.
func (s *Store) Close() error {
	return s.db.Close()
}

// GetSettings reads the single settings row.
func (s *Store) GetSettings() (model.Settings, error) {
	var out model.Settings
	err := s.db.QueryRow(
		`SELECT unitate_nume, next_nr, cota_tva FROM settings WHERE id = 1`,
	).Scan(&out.UnitateNume, &out.NextNr, &out.CotaTVA)
	if err != nil {
		return model.Settings{}, fmt.Errorf("citire setari: %w", err)
	}
	return out, nil
}

// SaveSettings overwrites the settings row.
func (s *Store) SaveSettings(in model.Settings) error {
	_, err := s.db.Exec(
		`INSERT INTO settings (id, unitate_nume, next_nr, cota_tva)
		 VALUES (1, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET unitate_nume = excluded.unitate_nume,
		        next_nr = excluded.next_nr, cota_tva = excluded.cota_tva`,
		in.UnitateNume, in.NextNr, in.CotaTVA,
	)
	if err != nil {
		return fmt.Errorf("salvare setari: %w", err)
	}
	return nil
}
