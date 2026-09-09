package store

import (
	"database/sql"
	"fmt"
	"strings"
)

// ListFurnizori returns the remembered supplier names, alphabetically.
func (s *Store) ListFurnizori() ([]string, error) {
	rows, err := s.db.Query(`SELECT nume FROM furnizori ORDER BY nume COLLATE NOCASE`)
	if err != nil {
		return nil, fmt.Errorf("citire furnizori: %w", err)
	}
	defer rows.Close()

	out := []string{}
	for rows.Next() {
		var nume string
		if err := rows.Scan(&nume); err != nil {
			return nil, fmt.Errorf("citire furnizor: %w", err)
		}
		out = append(out, nume)
	}
	return out, rows.Err()
}

// DeleteFurnizor forgets one suggestion. Documents keep the supplier as plain
// text, so nothing that has been filed changes: this removes an entry from a
// list of hints, not a supplier from history.
//
// A name that is not there is not an error — the user asked for it to be gone,
// and it is.
func (s *Store) DeleteFurnizor(nume string) error {
	if _, err := s.db.Exec(
		`DELETE FROM furnizori WHERE nume = ? COLLATE NOCASE`, strings.TrimSpace(nume),
	); err != nil {
		return fmt.Errorf("stergere furnizor: %w", err)
	}
	return nil
}

// retineFurnizor records a supplier name as a suggestion for later documents.
// It runs inside SaveDocument's transaction, so a document and the suggestion
// it produced are committed together.
//
// An empty name is skipped rather than stored: a document may legitimately go
// without a supplier, and a blank suggestion would be an unpickable entry in
// the list. A name already there, in any casing, is left as it was — the first
// spelling wins, so the list does not grow a second entry that reads the same.
func retineFurnizor(tx *sql.Tx, nume string) error {
	nume = strings.TrimSpace(nume)
	if nume == "" {
		return nil
	}
	if _, err := tx.Exec(
		`INSERT INTO furnizori (nume) VALUES (?)
		 ON CONFLICT DO NOTHING`, nume,
	); err != nil {
		return fmt.Errorf("memorare furnizor: %w", err)
	}
	return nil
}
