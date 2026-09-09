package store

import (
	"errors"
	"testing"

	"nota-de-receptie/internal/model"
)

// docExemplu is a two-row reception, ready to save.
func docExemplu(produsID *int64) model.Document {
	return model.Document{
		Nr:                  1,
		Data:                "2026-09-09",
		Unitate:             "S.C. Largiana Carn S.R.L.",
		DocumentLivrare:     "Factura",
		DocumentLivrareNr:   "1234",
		DocumentLivrareData: "2026-09-08",
		Furnizor:            "Alfa SRL",
		Randuri: []model.Rand{
			{ProductID: produsID, Pozitie: 0, Denumire: "Pulpa fara os", UM: "Kg.",
				Cantitate: 10, PretFaraTVA: 20, CotaTVA: 11, PretVanzare: 30},
			{Pozitie: 1, Denumire: "Oua", UM: "Buc.",
				Cantitate: 30, PretFaraTVA: 0.9, CotaTVA: 11, PretVanzare: 1.5},
		},
	}
}

func TestSaveSiGetDocument(t *testing.T) {
	s := deschide(t)
	salvat, err := s.SaveDocument(docExemplu(nil))
	if err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}
	if salvat.ID == 0 {
		t.Fatal("documentul salvat nu a primit id")
	}
	if salvat.CreatedAt == "" || salvat.UpdatedAt == "" {
		t.Error("createdAt/updatedAt nu au fost completate")
	}

	got, err := s.GetDocument(salvat.ID)
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}
	if got.Nr != 1 || got.Furnizor != "Alfa SRL" || got.DocumentLivrare != "Factura" {
		t.Errorf("antetul = %+v", got)
	}
	if len(got.Randuri) != 2 {
		t.Fatalf("len(Randuri) = %d, vrem 2", len(got.Randuri))
	}
	if got.Randuri[0].Denumire != "Pulpa fara os" || got.Randuri[0].UM != "Kg." {
		t.Errorf("randul 0 = %+v", got.Randuri[0])
	}
	if got.Randuri[1].UM != "Buc." || got.Randuri[1].Cantitate != 30 {
		t.Errorf("randul 1 = %+v", got.Randuri[1])
	}
}

func TestGetDocumentInexistent(t *testing.T) {
	s := deschide(t)
	_, err := s.GetDocument(999)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, vrem ErrNotFound", err)
	}
}

func TestSaveDocumentUrcaContorul(t *testing.T) {
	s := deschide(t)
	if _, err := s.SaveDocument(docExemplu(nil)); err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}
	setari, _ := s.GetSettings()
	if setari.NextNr != 2 {
		t.Errorf("NextNr = %d dupa documentul 1, vrem 2", setari.NextNr)
	}

	// Un numar sarit urca contorul peste el, nu cu unu.
	d := docExemplu(nil)
	d.Nr = 10
	if _, err := s.SaveDocument(d); err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}
	setari, _ = s.GetSettings()
	if setari.NextNr != 11 {
		t.Errorf("NextNr = %d dupa documentul 10, vrem 11", setari.NextNr)
	}

	// Un numar mai mic nu coboara contorul.
	d2 := docExemplu(nil)
	d2.Nr = 3
	if _, err := s.SaveDocument(d2); err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}
	setari, _ = s.GetSettings()
	if setari.NextNr != 11 {
		t.Errorf("NextNr = %d dupa documentul 3, vrem tot 11", setari.NextNr)
	}
}

func TestSaveDocumentExistentNuUrcaContorul(t *testing.T) {
	s := deschide(t)
	salvat, _ := s.SaveDocument(docExemplu(nil))
	setari, _ := s.GetSettings()
	inainte := setari.NextNr

	salvat.Furnizor = "Beta SA"
	if _, err := s.SaveDocument(salvat); err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}
	setari, _ = s.GetSettings()
	if setari.NextNr != inainte {
		t.Errorf("NextNr = %d dupa o reeditare, vrem tot %d", setari.NextNr, inainte)
	}
}

func TestSaveDocumentInlocuiesteRandurile(t *testing.T) {
	s := deschide(t)
	salvat, _ := s.SaveDocument(docExemplu(nil))

	salvat.Randuri = []model.Rand{
		{Pozitie: 0, Denumire: "Singurul rand", UM: "Kg.", Cantitate: 1, PretFaraTVA: 1, CotaTVA: 11, PretVanzare: 2},
	}
	if _, err := s.SaveDocument(salvat); err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}
	got, _ := s.GetDocument(salvat.ID)
	if len(got.Randuri) != 1 || got.Randuri[0].Denumire != "Singurul rand" {
		t.Errorf("randuri = %+v", got.Randuri)
	}
}

func TestSaveDocumentMemoreazaFurnizorul(t *testing.T) {
	s := deschide(t)
	if _, err := s.SaveDocument(docExemplu(nil)); err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}
	f, _ := s.ListFurnizori()
	if len(f) != 1 || f[0] != "Alfa SRL" {
		t.Errorf("furnizori = %v, vrem [Alfa SRL]", f)
	}
}

func TestEditareaProdusuluiNuSchimbaDocumentulSalvat(t *testing.T) {
	// Cerinta centrala a aplicatiei: randul e o fotografie, nu o legatura.
	s := deschide(t)
	p, err := s.AddProduct(model.Product{Denumire: "Pulpa fara os", UM: "Kg.", PretVanzare: 30, CotaTVA: 11})
	if err != nil {
		t.Fatalf("AddProduct: %v", err)
	}
	salvat, err := s.SaveDocument(docExemplu(&p.ID))
	if err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}

	if err := s.SaveProducts([]model.Product{
		{ID: p.ID, Denumire: "Pulpa fara os (alt nume)", UM: "Buc.", PretVanzare: 99, CotaTVA: 21, Ordine: 0},
	}); err != nil {
		t.Fatalf("SaveProducts: %v", err)
	}

	got, _ := s.GetDocument(salvat.ID)
	r := got.Randuri[0]
	if r.Denumire != "Pulpa fara os" || r.UM != "Kg." || r.PretVanzare != 30 || r.CotaTVA != 11 {
		t.Errorf("randul s-a schimbat odata cu produsul: %+v", r)
	}
}

func TestStergereaProdusuluiLasaRandulIntreg(t *testing.T) {
	s := deschide(t)
	p, _ := s.AddProduct(model.Product{Denumire: "Pulpa fara os", UM: "Kg.", PretVanzare: 30, CotaTVA: 11})
	salvat, _ := s.SaveDocument(docExemplu(&p.ID))

	if err := s.SaveProducts(nil); err != nil {
		t.Fatalf("SaveProducts: %v", err)
	}

	got, _ := s.GetDocument(salvat.ID)
	r := got.Randuri[0]
	if r.ProductID != nil {
		t.Errorf("ProductID = %v, vrem nil dupa stergerea produsului", *r.ProductID)
	}
	if r.Denumire != "Pulpa fara os" || r.PretVanzare != 30 {
		t.Errorf("randul si-a pierdut datele: %+v", r)
	}
}

func TestListDocumentsCelMaiNouPrimul(t *testing.T) {
	s := deschide(t)
	vechi := docExemplu(nil)
	vechi.Data = "2026-01-01"
	vechi.Nr = 1
	s.SaveDocument(vechi)

	nou := docExemplu(nil)
	nou.Data = "2026-09-09"
	nou.Nr = 2
	s.SaveDocument(nou)

	lista, err := s.ListDocuments()
	if err != nil {
		t.Fatalf("ListDocuments: %v", err)
	}
	if len(lista) != 2 {
		t.Fatalf("len = %d, vrem 2", len(lista))
	}
	if lista[0].Nr != 2 {
		t.Errorf("primul din lista = NR %d, vrem NR 2", lista[0].Nr)
	}
	if lista[0].Furnizor != "Alfa SRL" {
		t.Errorf("rezumatul nu poarta furnizorul: %+v", lista[0])
	}
}

func TestDeleteDocumentSterageSiRandurile(t *testing.T) {
	s := deschide(t)
	salvat, _ := s.SaveDocument(docExemplu(nil))

	if err := s.DeleteDocument(salvat.ID); err != nil {
		t.Fatalf("DeleteDocument: %v", err)
	}
	if _, err := s.GetDocument(salvat.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, vrem ErrNotFound", err)
	}
	var randuri int
	s.db.QueryRow(`SELECT COUNT(*) FROM document_rows`).Scan(&randuri)
	if randuri != 0 {
		t.Errorf("au ramas %d randuri orfane", randuri)
	}
}
