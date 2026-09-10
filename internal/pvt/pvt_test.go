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
