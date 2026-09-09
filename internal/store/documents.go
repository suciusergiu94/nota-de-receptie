package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"nota-de-receptie/internal/model"
)

// ListDocuments returns every document, newest first.
func (s *Store) ListDocuments() ([]model.DocumentSummary, error) {
	rows, err := s.db.Query(
		`SELECT id, nr, data, furnizor FROM documents ORDER BY data DESC, id DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("citire documente: %w", err)
	}
	defer rows.Close()

	out := []model.DocumentSummary{}
	for rows.Next() {
		var d model.DocumentSummary
		if err := rows.Scan(&d.ID, &d.Nr, &d.Data, &d.Furnizor); err != nil {
			return nil, fmt.Errorf("citire document: %w", err)
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// GetDocument reads one document with its rows.
func (s *Store) GetDocument(id int64) (model.Document, error) {
	var d model.Document
	err := s.db.QueryRow(
		`SELECT id, nr, data, unitate, document_livrare, document_livrare_nr,
		        document_livrare_data, furnizor, created_at, updated_at
		 FROM documents WHERE id = ?`, id,
	).Scan(
		&d.ID, &d.Nr, &d.Data, &d.Unitate, &d.DocumentLivrare, &d.DocumentLivrareNr,
		&d.DocumentLivrareData, &d.Furnizor, &d.CreatedAt, &d.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Document{}, ErrNotFound
	}
	if err != nil {
		return model.Document{}, fmt.Errorf("citire document: %w", err)
	}

	if d.Randuri, err = s.randuri(id); err != nil {
		return model.Document{}, err
	}
	return d, nil
}

func (s *Store) randuri(documentID int64) ([]model.Rand, error) {
	rows, err := s.db.Query(
		`SELECT id, product_id, pozitie, denumire, um, cantitate, pret_fara_tva,
		        cota_tva, pret_vanzare
		 FROM document_rows WHERE document_id = ? ORDER BY pozitie, id`, documentID,
	)
	if err != nil {
		return nil, fmt.Errorf("citire randuri: %w", err)
	}
	defer rows.Close()

	out := []model.Rand{}
	for rows.Next() {
		var r model.Rand
		var productID sql.NullInt64
		if err := rows.Scan(&r.ID, &productID, &r.Pozitie, &r.Denumire, &r.UM,
			&r.Cantitate, &r.PretFaraTVA, &r.CotaTVA, &r.PretVanzare); err != nil {
			return nil, fmt.Errorf("citire rand: %w", err)
		}
		if productID.Valid {
			id := productID.Int64
			r.ProductID = &id
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// SaveDocument creates or updates a document and returns it as stored.
//
// Everything happens in one transaction: the document, its rows, the supplier
// suggestion and the number counter. There is no interleaving in which a save
// reports failure for a document that was in fact written, and none in which a
// document exists while the counter still points at its number.
func (s *Store) SaveDocument(doc model.Document) (model.Document, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return model.Document{}, fmt.Errorf("salvare document: %w", err)
	}
	defer tx.Rollback()

	acum := time.Now().UTC().Format(time.RFC3339)
	esteNou := doc.ID == 0

	if esteNou {
		doc.CreatedAt = acum
		doc.UpdatedAt = acum
		res, err := tx.Exec(
			`INSERT INTO documents (nr, data, unitate, document_livrare, document_livrare_nr,
			        document_livrare_data, furnizor, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			doc.Nr, doc.Data, doc.Unitate, doc.DocumentLivrare, doc.DocumentLivrareNr,
			doc.DocumentLivrareData, doc.Furnizor, doc.CreatedAt, doc.UpdatedAt,
		)
		if err != nil {
			return model.Document{}, fmt.Errorf("salvare document: %w", err)
		}
		if doc.ID, err = res.LastInsertId(); err != nil {
			return model.Document{}, fmt.Errorf("salvare document: %w", err)
		}
	} else {
		doc.UpdatedAt = acum
		if _, err := tx.Exec(
			`UPDATE documents SET nr = ?, data = ?, unitate = ?, document_livrare = ?,
			        document_livrare_nr = ?, document_livrare_data = ?, furnizor = ?,
			        updated_at = ?
			 WHERE id = ?`,
			doc.Nr, doc.Data, doc.Unitate, doc.DocumentLivrare, doc.DocumentLivrareNr,
			doc.DocumentLivrareData, doc.Furnizor, doc.UpdatedAt, doc.ID,
		); err != nil {
			return model.Document{}, fmt.Errorf("salvare document: %w", err)
		}
	}

	// The rows are replaced wholesale rather than diffed: they have no identity
	// of their own beyond their position, the form lets any of them be deleted
	// or reordered, and a document never holds enough of them for the rewrite
	// to cost anything.
	if _, err := tx.Exec(`DELETE FROM document_rows WHERE document_id = ?`, doc.ID); err != nil {
		return model.Document{}, fmt.Errorf("salvare randuri: %w", err)
	}
	for i := range doc.Randuri {
		r := &doc.Randuri[i]
		r.Pozitie = i
		var productID any
		if r.ProductID != nil {
			productID = *r.ProductID
		}
		res, err := tx.Exec(
			`INSERT INTO document_rows (document_id, product_id, pozitie, denumire, um,
			        cantitate, pret_fara_tva, cota_tva, pret_vanzare)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			doc.ID, productID, r.Pozitie, r.Denumire, r.UM, r.Cantitate,
			r.PretFaraTVA, r.CotaTVA, r.PretVanzare,
		)
		if err != nil {
			return model.Document{}, fmt.Errorf("salvare rand %d: %w", i+1, err)
		}
		if r.ID, err = res.LastInsertId(); err != nil {
			return model.Document{}, fmt.Errorf("salvare rand %d: %w", i+1, err)
		}
	}

	if err := retineFurnizor(tx, doc.Furnizor); err != nil {
		return model.Document{}, err
	}

	// The counter only ever moves forward, and only for a document that is
	// being created. Re-editing an old document must not drag the counter back
	// onto numbers already used, and saving one deliberately numbered below the
	// counter must not either.
	if esteNou {
		if _, err := tx.Exec(
			`UPDATE settings SET next_nr = ? WHERE id = 1 AND next_nr <= ?`,
			doc.Nr+1, doc.Nr,
		); err != nil {
			return model.Document{}, fmt.Errorf("actualizare numarator: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return model.Document{}, fmt.Errorf("salvare document: %w", err)
	}
	return doc, nil
}

// DeleteDocument removes a document and, through the foreign key, its rows.
func (s *Store) DeleteDocument(id int64) error {
	res, err := s.db.Exec(`DELETE FROM documents WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("stergere document: %w", err)
	}
	afectate, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("stergere document: %w", err)
	}
	if afectate == 0 {
		return ErrNotFound
	}
	return nil
}
