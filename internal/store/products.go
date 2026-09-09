package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"nota-de-receptie/internal/model"
)

// ErrDenumireDuplicata is returned when two products would share a name.
// Names are compared without regard to case: two catalogue entries that read
// the same are the same as far as the person picking one is concerned.
var ErrDenumireDuplicata = errors.New("exista deja un produs cu aceasta denumire")

// ListProducts returns the whole catalogue in display order.
func (s *Store) ListProducts() ([]model.Product, error) {
	rows, err := s.db.Query(
		`SELECT id, denumire, um, pret_vanzare, cota_tva, ordine
		 FROM products ORDER BY ordine, id`,
	)
	if err != nil {
		return nil, fmt.Errorf("citire produse: %w", err)
	}
	defer rows.Close()

	out := []model.Product{}
	for rows.Next() {
		var p model.Product
		if err := rows.Scan(&p.ID, &p.Denumire, &p.UM, &p.PretVanzare, &p.CotaTVA, &p.Ordine); err != nil {
			return nil, fmt.Errorf("citire produs: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// AddProduct inserts one product at the end of the catalogue and returns it
// with the id and ordine it was given. It exists so a name typed into the
// document form can be filed without leaving the form.
func (s *Store) AddProduct(p model.Product) (model.Product, error) {
	var maxOrdine sql.NullInt64
	if err := s.db.QueryRow(`SELECT MAX(ordine) FROM products`).Scan(&maxOrdine); err != nil {
		return model.Product{}, fmt.Errorf("citire ordine produse: %w", err)
	}
	p.Ordine = int(maxOrdine.Int64) + 1
	if !maxOrdine.Valid {
		p.Ordine = 0
	}

	res, err := s.db.Exec(
		`INSERT INTO products (denumire, um, pret_vanzare, cota_tva, ordine)
		 VALUES (?, ?, ?, ?, ?)`,
		strings.TrimSpace(p.Denumire), p.UM, p.PretVanzare, p.CotaTVA, p.Ordine,
	)
	if err != nil {
		if esteConflictDeDenumire(err) {
			return model.Product{}, ErrDenumireDuplicata
		}
		return model.Product{}, fmt.Errorf("salvare produs: %w", err)
	}
	if p.ID, err = res.LastInsertId(); err != nil {
		return model.Product{}, fmt.Errorf("salvare produs: %w", err)
	}
	p.Denumire = strings.TrimSpace(p.Denumire)
	return p, nil
}

// SaveProducts brings the catalogue in line with the given list: rows with an
// id are updated, rows without one are inserted, and anything not in the list
// is deleted.
//
// It deliberately does not delete everything and reinsert. document_rows keeps
// a product_id with ON DELETE SET NULL, and wiping the table would sever every
// saved document's link to the catalogue — the documents would survive (they
// carry their own snapshot) but they would all look as though their products
// had been deleted. Updating in place keeps the ids, so only products the user
// actually removed lose their link.
func (s *Store) SaveProducts(produse []model.Product) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("salvare produse: %w", err)
	}
	defer tx.Rollback()

	pastrate := make([]any, 0, len(produse))
	for i, p := range produse {
		denumire := strings.TrimSpace(p.Denumire)
		if p.ID == 0 {
			res, err := tx.Exec(
				`INSERT INTO products (denumire, um, pret_vanzare, cota_tva, ordine)
				 VALUES (?, ?, ?, ?, ?)`,
				denumire, p.UM, p.PretVanzare, p.CotaTVA, i,
			)
			if err != nil {
				if esteConflictDeDenumire(err) {
					return ErrDenumireDuplicata
				}
				return fmt.Errorf("salvare produs %q: %w", denumire, err)
			}
			id, err := res.LastInsertId()
			if err != nil {
				return fmt.Errorf("salvare produs %q: %w", denumire, err)
			}
			pastrate = append(pastrate, id)
			continue
		}
		if _, err := tx.Exec(
			`UPDATE products SET denumire = ?, um = ?, pret_vanzare = ?, cota_tva = ?, ordine = ?
			 WHERE id = ?`,
			denumire, p.UM, p.PretVanzare, p.CotaTVA, i, p.ID,
		); err != nil {
			if esteConflictDeDenumire(err) {
				return ErrDenumireDuplicata
			}
			return fmt.Errorf("salvare produs %q: %w", denumire, err)
		}
		pastrate = append(pastrate, p.ID)
	}

	if err := stergeProduseleLipsa(tx, pastrate); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("salvare produse: %w", err)
	}
	return nil
}

// stergeProduseleLipsa removes every product whose id is not in pastrate.
func stergeProduseleLipsa(tx *sql.Tx, pastrate []any) error {
	if len(pastrate) == 0 {
		if _, err := tx.Exec(`DELETE FROM products`); err != nil {
			return fmt.Errorf("stergere produse: %w", err)
		}
		return nil
	}
	semne := strings.TrimSuffix(strings.Repeat("?,", len(pastrate)), ",")
	if _, err := tx.Exec(`DELETE FROM products WHERE id NOT IN (`+semne+`)`, pastrate...); err != nil {
		return fmt.Errorf("stergere produse: %w", err)
	}
	return nil
}

// esteConflictDeDenumire reports whether err is the unique index on
// products.denumire refusing a duplicate. modernc.org/sqlite reports it as a
// message rather than a typed error, so the message is what there is to match
// on; the index name makes the match specific to this one constraint.
func esteConflictDeDenumire(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed: products.denumire")
}
