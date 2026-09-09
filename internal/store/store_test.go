package store

import (
	"path/filepath"
	"testing"

	"nota-de-receptie/internal/model"
)

// deschide creates a store on a fresh file under the test's temp dir.
func deschide(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestOpenSeamanaSetarileImplicite(t *testing.T) {
	s := deschide(t)
	got, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if got.UnitateNume != "S.C. Largiana Carn S.R.L." {
		t.Errorf("UnitateNume = %q", got.UnitateNume)
	}
	if got.NextNr != 1 {
		t.Errorf("NextNr = %d, vrem 1", got.NextNr)
	}
	if got.CotaTVA != 11 {
		t.Errorf("CotaTVA = %v, vrem 11", got.CotaTVA)
	}
}

func TestCatalogulSiFurnizoriiPornescGoale(t *testing.T) {
	s := deschide(t)
	var produse, furnizori int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM products`).Scan(&produse); err != nil {
		t.Fatalf("numarare produse: %v", err)
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM furnizori`).Scan(&furnizori); err != nil {
		t.Fatalf("numarare furnizori: %v", err)
	}
	if produse != 0 || furnizori != 0 {
		t.Errorf("produse = %d, furnizori = %d; vrem 0 si 0", produse, furnizori)
	}
}

func TestSaveSettingsSuprascrie(t *testing.T) {
	s := deschide(t)
	vrem := model.Settings{UnitateNume: "Alta firma", NextNr: 42, CotaTVA: 21}
	if err := s.SaveSettings(vrem); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	got, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if got != vrem {
		t.Errorf("setari = %+v, vrem %+v", got, vrem)
	}
}

func TestADouaDeschidereNuReseamana(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := s.SaveSettings(model.Settings{UnitateNume: "X", NextNr: 7, CotaTVA: 5}); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	s.Close()

	s2, err := Open(path)
	if err != nil {
		t.Fatalf("a doua Open: %v", err)
	}
	defer s2.Close()
	got, err := s2.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if got.NextNr != 7 {
		t.Errorf("NextNr = %d dupa redeschidere, vrem 7", got.NextNr)
	}
}
