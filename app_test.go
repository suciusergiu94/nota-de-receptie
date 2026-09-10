package main

import (
	"path/filepath"
	"testing"

	"nota-de-receptie/internal/model"
	"nota-de-receptie/internal/store"
)

// appDeTest builds an App on a throwaway database. ExportPDF is left out of
// these tests: it opens a native save dialog, which needs a running Wails
// context.
func appDeTest(t *testing.T) *App {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return &App{store: s}
}

func TestNewDocumentDraft(t *testing.T) {
	a := appDeTest(t)
	draft, err := a.NewDocumentDraft()
	if err != nil {
		t.Fatalf("NewDocumentDraft: %v", err)
	}
	if draft.ID != 0 {
		t.Errorf("ID = %d, vrem 0 pe o schita nesalvata", draft.ID)
	}
	if draft.Nr != 1 {
		t.Errorf("Nr = %d, vrem 1 pe prima instalare", draft.Nr)
	}
	if draft.Unitate != "S.C. Largiana Carn S.R.L." {
		t.Errorf("Unitate = %q; schita nu a preluat-o din setari", draft.Unitate)
	}
	if len(draft.Data) != 10 {
		t.Errorf("Data = %q, vrem o data ISO", draft.Data)
	}
	if draft.Randuri == nil {
		t.Error("Randuri = nil; vrem o lista goala, ca frontendul sa nu vada null")
	}
	if len(draft.Randuri) != 0 {
		t.Errorf("len(Randuri) = %d, vrem 0: nota de receptie porneste goala", len(draft.Randuri))
	}
}

func TestSchitaUrmeazaContorul(t *testing.T) {
	a := appDeTest(t)
	primul, _ := a.NewDocumentDraft()
	primul.Randuri = []model.Rand{
		{Denumire: "X", UM: "Kg.", Cantitate: 1, PretFaraTVA: 10, CotaTVA: 11, PretVanzare: 15},
	}
	if _, err := a.SaveDocument(primul); err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}

	alDoilea, err := a.NewDocumentDraft()
	if err != nil {
		t.Fatalf("NewDocumentDraft: %v", err)
	}
	if alDoilea.Nr != 2 {
		t.Errorf("Nr = %d pe a doua schita, vrem 2", alDoilea.Nr)
	}
}

func TestAddProductDinFormular(t *testing.T) {
	a := appDeTest(t)
	p, err := a.AddProduct(model.Product{Denumire: "Costita", UM: "Kg.", PretVanzare: 29, CotaTVA: 11})
	if err != nil {
		t.Fatalf("AddProduct: %v", err)
	}
	if p.ID == 0 {
		t.Error("produsul nu a primit id")
	}
	produse, _ := a.ListProducts()
	if len(produse) != 1 {
		t.Errorf("len = %d, vrem 1", len(produse))
	}
}

func TestListeleIntorcSliceGolNuNil(t *testing.T) {
	// Un nil devine null in JSON, iar frontendul ar cadea pe .map().
	a := appDeTest(t)
	documente, err := a.ListDocuments()
	if err != nil {
		t.Fatalf("ListDocuments: %v", err)
	}
	if documente == nil {
		t.Error("ListDocuments = nil, vrem []")
	}
	produse, _ := a.ListProducts()
	if produse == nil {
		t.Error("ListProducts = nil, vrem []")
	}
	furnizori, _ := a.ListFurnizori()
	if furnizori == nil {
		t.Error("ListFurnizori = nil, vrem []")
	}
}

func TestSaveSiSterge(t *testing.T) {
	a := appDeTest(t)
	draft, _ := a.NewDocumentDraft()
	draft.Furnizor = "Alfa SRL"
	draft.Randuri = []model.Rand{
		{Denumire: "X", UM: "Kg.", Cantitate: 2, PretFaraTVA: 10, CotaTVA: 11, PretVanzare: 15},
	}
	salvat, err := a.SaveDocument(draft)
	if err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}

	got, err := a.GetDocument(salvat.ID)
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}
	if got.Furnizor != "Alfa SRL" {
		t.Errorf("Furnizor = %q", got.Furnizor)
	}

	if err := a.DeleteDocument(salvat.ID); err != nil {
		t.Fatalf("DeleteDocument: %v", err)
	}
	lista, _ := a.ListDocuments()
	if len(lista) != 0 {
		t.Errorf("len = %d dupa stergere, vrem 0", len(lista))
	}
}
