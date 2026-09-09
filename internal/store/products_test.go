package store

import (
	"errors"
	"testing"

	"nota-de-receptie/internal/model"
)

func TestCatalogulPornesteGol(t *testing.T) {
	s := deschide(t)
	produse, err := s.ListProducts()
	if err != nil {
		t.Fatalf("ListProducts: %v", err)
	}
	if len(produse) != 0 {
		t.Errorf("len = %d, vrem 0", len(produse))
	}
}

func TestAddProductPuneLaCoada(t *testing.T) {
	s := deschide(t)
	unu, err := s.AddProduct(model.Product{Denumire: "Pulpa fara os", UM: "Kg.", PretVanzare: 21.9, CotaTVA: 11})
	if err != nil {
		t.Fatalf("AddProduct: %v", err)
	}
	if unu.ID == 0 {
		t.Error("produsul salvat nu a primit id")
	}
	doi, err := s.AddProduct(model.Product{Denumire: "Oua", UM: "Buc.", PretVanzare: 1.2, CotaTVA: 11})
	if err != nil {
		t.Fatalf("AddProduct: %v", err)
	}
	if doi.Ordine <= unu.Ordine {
		t.Errorf("ordine = %d si %d; al doilea produs trebuie sa vina dupa primul", unu.Ordine, doi.Ordine)
	}

	produse, err := s.ListProducts()
	if err != nil {
		t.Fatalf("ListProducts: %v", err)
	}
	if len(produse) != 2 || produse[0].Denumire != "Pulpa fara os" || produse[1].Denumire != "Oua" {
		t.Errorf("catalogul = %+v", produse)
	}
}

func TestAddProductRefuzaDenumireaDuplicata(t *testing.T) {
	s := deschide(t)
	if _, err := s.AddProduct(model.Product{Denumire: "Costita", UM: "Kg.", CotaTVA: 11}); err != nil {
		t.Fatalf("AddProduct: %v", err)
	}
	// Aceeasi denumire cu alte majuscule este tot un duplicat: utilizatorul
	// n-are cum sa deosebeasca doua intrari care arata la fel in lista.
	_, err := s.AddProduct(model.Product{Denumire: "costita", UM: "Kg.", CotaTVA: 11})
	if !errors.Is(err, ErrDenumireDuplicata) {
		t.Errorf("err = %v, vrem ErrDenumireDuplicata", err)
	}
}

func TestSaveProductsPastreazaIdurileCeloraRamase(t *testing.T) {
	s := deschide(t)
	a, _ := s.AddProduct(model.Product{Denumire: "A", UM: "Kg.", PretVanzare: 10, CotaTVA: 11})
	b, _ := s.AddProduct(model.Product{Denumire: "B", UM: "Kg.", PretVanzare: 20, CotaTVA: 11})

	// B se redenumeste si isi schimba pretul, A ramane, se adauga un C nou.
	err := s.SaveProducts([]model.Product{
		{ID: a.ID, Denumire: "A", UM: "Kg.", PretVanzare: 10, CotaTVA: 11, Ordine: 0},
		{ID: b.ID, Denumire: "B redenumit", UM: "Buc.", PretVanzare: 25, CotaTVA: 21, Ordine: 1},
		{ID: 0, Denumire: "C", UM: "Kg.", PretVanzare: 5, CotaTVA: 11, Ordine: 2},
	})
	if err != nil {
		t.Fatalf("SaveProducts: %v", err)
	}

	produse, _ := s.ListProducts()
	if len(produse) != 3 {
		t.Fatalf("len = %d, vrem 3", len(produse))
	}
	if produse[1].ID != b.ID {
		t.Errorf("id-ul lui B = %d, vrem %d pastrat", produse[1].ID, b.ID)
	}
	if produse[1].Denumire != "B redenumit" || produse[1].UM != "Buc." || produse[1].PretVanzare != 25 || produse[1].CotaTVA != 21 {
		t.Errorf("B nu s-a actualizat: %+v", produse[1])
	}
	if produse[2].Denumire != "C" || produse[2].ID == 0 {
		t.Errorf("C nu s-a inserat: %+v", produse[2])
	}
}

func TestSaveProductsStergeCeLipseste(t *testing.T) {
	s := deschide(t)
	a, _ := s.AddProduct(model.Product{Denumire: "A", UM: "Kg.", CotaTVA: 11})
	s.AddProduct(model.Product{Denumire: "B", UM: "Kg.", CotaTVA: 11})

	if err := s.SaveProducts([]model.Product{{ID: a.ID, Denumire: "A", UM: "Kg.", CotaTVA: 11, Ordine: 0}}); err != nil {
		t.Fatalf("SaveProducts: %v", err)
	}
	produse, _ := s.ListProducts()
	if len(produse) != 1 || produse[0].Denumire != "A" {
		t.Errorf("catalogul = %+v, vrem doar A", produse)
	}
}

func TestSaveProductsSchimbaNumeleIntreEle(t *testing.T) {
	s := deschide(t)
	a, _ := s.AddProduct(model.Product{Denumire: "A", UM: "Kg.", CotaTVA: 11})
	b, _ := s.AddProduct(model.Product{Denumire: "B", UM: "Kg.", CotaTVA: 11})

	// A si B isi schimba numele intre ele intr-un singur apel. Fiecare UPDATE
	// separat s-ar ciocni cu numele pe care celalalt rand nu l-a eliberat
	// inca, desi starea finala nu are niciun duplicat.
	err := s.SaveProducts([]model.Product{
		{ID: a.ID, Denumire: "B", UM: "Kg.", CotaTVA: 11, Ordine: 0},
		{ID: b.ID, Denumire: "A", UM: "Kg.", CotaTVA: 11, Ordine: 1},
	})
	if err != nil {
		t.Fatalf("SaveProducts: %v", err)
	}

	produse, _ := s.ListProducts()
	if len(produse) != 2 {
		t.Fatalf("len = %d, vrem 2", len(produse))
	}
	if produse[0].ID != a.ID || produse[0].Denumire != "B" {
		t.Errorf("primul rand = %+v, vrem id-ul lui A cu denumirea B", produse[0])
	}
	if produse[1].ID != b.ID || produse[1].Denumire != "A" {
		t.Errorf("al doilea rand = %+v, vrem id-ul lui B cu denumirea A", produse[1])
	}
}

func TestSaveProductsProdusNouIaNumeleEliberat(t *testing.T) {
	s := deschide(t)
	a, _ := s.AddProduct(model.Product{Denumire: "A", UM: "Kg.", CotaTVA: 11})

	// A se redenumeste, iar un produs nou preia exact numele pe care A il
	// elibereaza, in acelasi apel. Trece doar daca vacantarea din pasul 1
	// ruleaza inaintea scrierii valorilor finale din pasul 2.
	err := s.SaveProducts([]model.Product{
		{ID: a.ID, Denumire: "A schimbat", UM: "Kg.", CotaTVA: 11, Ordine: 0},
		{ID: 0, Denumire: "A", UM: "Kg.", CotaTVA: 11, Ordine: 1},
	})
	if err != nil {
		t.Fatalf("SaveProducts: %v", err)
	}

	produse, _ := s.ListProducts()
	if len(produse) != 2 {
		t.Fatalf("len = %d, vrem 2", len(produse))
	}
	if produse[0].ID != a.ID || produse[0].Denumire != "A schimbat" {
		t.Errorf("primul rand = %+v", produse[0])
	}
	if produse[1].ID == 0 || produse[1].ID == a.ID || produse[1].Denumire != "A" {
		t.Errorf("al doilea rand = %+v, vrem un produs nou numit A", produse[1])
	}
}

func TestSaveProductsPastreazaOrdineaDinLista(t *testing.T) {
	s := deschide(t)
	a, _ := s.AddProduct(model.Product{Denumire: "A", UM: "Kg.", CotaTVA: 11})
	b, _ := s.AddProduct(model.Product{Denumire: "B", UM: "Kg.", CotaTVA: 11})

	// B (id mai mare) e trecut inaintea lui A in lista salvata: daca
	// ListProducts ar ordona doar dupa id, ar intoarce tot A, B - gresit.
	err := s.SaveProducts([]model.Product{
		{ID: b.ID, Denumire: "B", UM: "Kg.", CotaTVA: 11, Ordine: 0},
		{ID: a.ID, Denumire: "A", UM: "Kg.", CotaTVA: 11, Ordine: 1},
	})
	if err != nil {
		t.Fatalf("SaveProducts: %v", err)
	}

	produse, err := s.ListProducts()
	if err != nil {
		t.Fatalf("ListProducts: %v", err)
	}
	if len(produse) != 2 || produse[0].ID != b.ID || produse[1].ID != a.ID {
		t.Errorf("catalogul = %+v, vrem B inaintea lui A", produse)
	}
}

func TestSaveProductsRefuzaDuplicatele(t *testing.T) {
	s := deschide(t)
	err := s.SaveProducts([]model.Product{
		{Denumire: "Costita", UM: "Kg.", CotaTVA: 11, Ordine: 0},
		{Denumire: "COSTITA", UM: "Kg.", CotaTVA: 11, Ordine: 1},
	})
	if !errors.Is(err, ErrDenumireDuplicata) {
		t.Errorf("err = %v, vrem ErrDenumireDuplicata", err)
	}
	// Refuzul e total: nimic nu trebuie sa fi ramas scris.
	produse, _ := s.ListProducts()
	if len(produse) != 0 {
		t.Errorf("catalogul = %+v dupa un refuz, vrem gol", produse)
	}
}
