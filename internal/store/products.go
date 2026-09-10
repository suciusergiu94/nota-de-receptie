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
// is deleted. Order is taken entirely from each product's position in the
// slice (stored as ordine = its index); the Ordine field on the input is
// ignored, since the slice's order is what the caller actually means.
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

	// Pass 1: delete what the list no longer carries. This runs first because
	// a deleted product keeps its name until it is gone, and that name is one
	// the list may well be handing to another row: deleting "A" and adding a
	// new "A", or deleting "B" and renaming "A" to "B", are both a single
	// save. Deleting last made either of those collide with a name nothing on
	// screen still holds, and since the deletion and the rename travel in the
	// same list, pressing Salvează again could never break the deadlock.
	//
	// Only entries that already have an id can name a row to keep; a row the
	// user has just added has no id yet and cannot save anything from deletion.
	pastrate := make([]any, 0, len(produse))
	for _, p := range produse {
		if p.ID != 0 {
			pastrate = append(pastrate, p.ID)
		}
	}
	if err := stergeProduseleLipsa(tx, pastrate); err != nil {
		return err
	}

	// Pass 2: vacate every surviving row's current name into a value that
	// cannot collide with anything (SQLite checks UNIQUE per statement, not
	// at commit). Without this, swapping two products' names in one save
	// would have the first UPDATE collide with a name the other row hasn't
	// given up yet, even though the end state has no duplicate at all.
	//
	// A leading NUL byte was tried first (as a value no real product name
	// could contain) but modernc.org/sqlite's NOCASE comparison treats it as
	// a C-string terminator: every "\x00tmp-N" value compares equal to every
	// other one, which promptly collides across rows instead of avoiding
	// collisions. A control character that is not a string terminator (SOH)
	// keeps the per-id suffix significant to the comparison.
	for _, p := range produse {
		if p.ID == 0 {
			continue
		}
		locTemp := fmt.Sprintf("\x01tmp-%d", p.ID)
		if _, err := tx.Exec(`UPDATE products SET denumire = ? WHERE id = ?`, locTemp, p.ID); err != nil {
			return fmt.Errorf("salvare produse: %w", err)
		}
	}

	// Pass 3: write the final values. A genuine duplicate (two entries in
	// the input asking for the same name) still collides here and is
	// reported as ErrDenumireDuplicata; a name freed by pass 1 or pass 2 no
	// longer blocks a different row from taking it.
	for i, p := range produse {
		denumire := strings.TrimSpace(p.Denumire)
		if p.ID == 0 {
			if _, err := tx.Exec(
				`INSERT INTO products (denumire, um, pret_vanzare, cota_tva, ordine)
				 VALUES (?, ?, ?, ?, ?)`,
				denumire, p.UM, p.PretVanzare, p.CotaTVA, i,
			); err != nil {
				if esteConflictDeDenumire(err) {
					return ErrDenumireDuplicata
				}
				return fmt.Errorf("salvare produs %q: %w", denumire, err)
			}
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
	}

	// Everything above is one transaction: a refusal in the write pass rolls
	// the deletions back with it, so a save that fails changes nothing.
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
