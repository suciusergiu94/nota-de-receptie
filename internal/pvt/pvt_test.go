package pvt

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"nota-de-receptie/internal/calc"
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

func TestImportulImparteDupaCantitateCandPreturileSuntZero(t *testing.T) {
	path := fixtura(t)
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO documents (id, nr, data, gestiune, created_at, updated_at)
		 VALUES (3, 3, '2026-09-06', 'Magazin Bradet', '2026-09-06T10:00:00Z', '2026-09-06T10:00:00Z')`,
	); err != nil {
		t.Fatalf("document: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO document_intrare_rows (document_id, pozitie, denumire, um,
		        cantitate, pret_fara_tva, pret_cu_tva, cota_tva)
		 VALUES (3, 0, 'A', 'Kg', 100, 0, 0, 11),
		        (3, 1, 'B', 'Kg', 300, 0, 0, 11)`,
	); err != nil {
		t.Fatalf("intrare: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO document_iesire_rows (document_id, pozitie, denumire, um,
		        pret_cu_tva, cantitate, pret_fara_tva, cota_tva)
		 VALUES (3, 0, 'Ceva', 'Kg', 10, 10, 9.01, 11)`,
	); err != nil {
		t.Fatalf("iesire: %v", err)
	}
	db.Close()

	randuri, err := ImportDin(path, 3)
	if err != nil {
		t.Fatalf("ImportDin: %v", err)
	}
	if len(randuri) != 2 {
		t.Fatalf("randuri = %d, vrem 2", len(randuri))
	}
	if randuri[0].ValoareVanzareImpusa == nil || randuri[1].ValoareVanzareImpusa == nil {
		t.Fatal("un rand nu are valoare impusa")
	}
	// Preturile sunt zero pe ambele randuri, deci nu exista nicio proportie
	// de valoare de urmat; impartirea trebuie sa cada pe cantitati: 100 si
	// 300, adica 25% si 75% din totalul de 100.
	aproape(t, "cota[0]", *randuri[0].ValoareVanzareImpusa, 25)
	aproape(t, "cota[1]", *randuri[1].ValoareVanzareImpusa, 75)
}

func TestImportulNumesteRandulDupaDocumentCandDenumireaLipseste(t *testing.T) {
	path := fixtura(t)
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	// nr difera intentionat de id, ca testul sa nu poata trece daca
	// implementarea foloseste din greseala id-ul in loc de nr.
	if _, err := db.Exec(
		`INSERT INTO documents (id, nr, data, gestiune, created_at, updated_at)
		 VALUES (5, 42, '2026-09-07', 'Magazin Bradet', '2026-09-07T10:00:00Z', '2026-09-07T10:00:00Z')`,
	); err != nil {
		t.Fatalf("document: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO document_intrare_rows (document_id, pozitie, denumire, um,
		        cantitate, pret_fara_tva, pret_cu_tva, cota_tva)
		 VALUES (5, 0, '', 'Kg', 10, 5, 5.55, 11)`,
	); err != nil {
		t.Fatalf("intrare: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO document_iesire_rows (document_id, pozitie, denumire, um,
		        pret_cu_tva, cantitate, pret_fara_tva, cota_tva)
		 VALUES (5, 0, 'Ceva', 'Kg', 10, 10, 9.01, 11)`,
	); err != nil {
		t.Fatalf("iesire: %v", err)
	}
	db.Close()

	randuri, err := ImportDin(path, 5)
	if err != nil {
		t.Fatalf("ImportDin: %v", err)
	}
	if len(randuri) != 1 {
		t.Fatalf("randuri = %d, vrem 1", len(randuri))
	}
	if randuri[0].Denumire != "Proces verbal nr. 42" {
		t.Errorf("Denumire = %q, vrem \"Proces verbal nr. 42\"", randuri[0].Denumire)
	}
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

// amprenta is everything about a file that a write would disturb.
type amprentaFisier struct {
	marime    int64
	modificat string
	versiune  int
	continut  string
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
