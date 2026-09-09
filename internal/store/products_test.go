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
