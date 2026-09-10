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
	"nota-de-receptie/internal/model"
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
	ErrIndisponibil  = errors.New("aplicatia Proces verbal de transare nu este instalata pe acest calculator, sau nu a fost pornita niciodata")
	ErrVersiuneVeche = errors.New("baza aplicatiei Proces verbal de transare are o forma mai veche, dinaintea primei versiuni publicate; deschide o data acea aplicatie si incearca din nou")
	ErrForma         = errors.New("baza aplicatiei Proces verbal de transare are alta forma decat cea asteptata; probabil a fost actualizata la o versiune mai noua")
	ErrJurnal        = errors.New("baza aplicatiei Proces verbal de transare a ramas cu un jurnal neincheiat dupa o inchidere fortata; deschide o data acea aplicatie si incearca din nou")
	ErrFaraIntrare   = errors.New("procesul verbal ales nu are niciun rand in tabelul \"ce intra\"")
)

// Sumar is one proces verbal as it appears in the picker.
type Sumar struct {
	ID       int64  `json:"id"`
	Nr       int    `json:"nr"`
	Data     string `json:"data"` // ISO YYYY-MM-DD
	Gestiune string `json:"gestiune"`
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
