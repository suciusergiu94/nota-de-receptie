package pdfdoc

import (
	"bytes"
	"testing"

	"nota-de-receptie/internal/model"
)

func doc() model.Document {
	return model.Document{
		Nr:                  7,
		Data:                "2026-09-09",
		Unitate:             "S.C. Largiana Carn S.R.L.",
		DocumentLivrare:     "Factură",
		DocumentLivrareNr:   "1234",
		DocumentLivrareData: "2026-09-08",
		Furnizor:            "Alfa SRL",
		Randuri: []model.Rand{
			{Pozitie: 0, Denumire: "Pulpă fără os", UM: "Kg.", Cantitate: 10,
				PretFaraTVA: 20, CotaTVA: 11, PretVanzare: 30},
			{Pozitie: 1, Denumire: "Ouă", UM: "Buc.", Cantitate: 30,
				PretFaraTVA: 0.9, CotaTVA: 11, PretVanzare: 1.5},
		},
	}
}

func TestFold(t *testing.T) {
	got := Fold("Pulpă fără os, șuncă țărănească, ÎNTOCMIT")
	want := "Pulpa fara os, sunca taraneasca, INTOCMIT"
	if got != want {
		t.Errorf("Fold = %q, vrem %q", got, want)
	}
}

func TestRenderIntoarceUnPDF(t *testing.T) {
	data, err := Render(doc())
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !bytes.HasPrefix(data, []byte("%PDF")) {
		t.Errorf("iesirea nu incepe cu %%PDF: %q", data[:min(8, len(data))])
	}
	if len(data) < 1000 {
		t.Errorf("PDF-ul are doar %d octeti; pare gol", len(data))
	}
}

func TestRenderPeDocumentFaraRanduri(t *testing.T) {
	d := doc()
	d.Randuri = nil
	data, err := Render(d)
	if err != nil {
		t.Fatalf("Render pe document gol: %v", err)
	}
	if !bytes.HasPrefix(data, []byte("%PDF")) {
		t.Error("iesirea nu e un PDF")
	}
}

func TestRenderPeMulteRanduriTreceInPaginaUrmatoare(t *testing.T) {
	d := doc()
	d.Randuri = nil
	for i := 0; i < 120; i++ {
		d.Randuri = append(d.Randuri, model.Rand{
			Pozitie: i, Denumire: "Produs", UM: "Kg.", Cantitate: 1,
			PretFaraTVA: 1, CotaTVA: 11, PretVanzare: 2,
		})
	}
	// Numarul de pagini se citeste de la fpdf, nu se ghiceste din octetii
	// fisierului: reprezentarea lor interna e treaba bibliotecii si se poate
	// schimba fara ca nimic sa fie in neregula.
	if n := construieste(d).PageNo(); n < 2 {
		t.Errorf("am construit %d pagini, vrem cel putin 2", n)
	}
	if _, err := Render(d); err != nil {
		t.Fatalf("Render: %v", err)
	}
}

func TestLatimileColoanelorUmpluPagina(t *testing.T) {
	var suma float64
	for _, w := range colWidths {
		suma += w
	}
	if suma != 277 {
		t.Errorf("suma latimilor = %v, vrem 277 (A4 landscape minus marginile)", suma)
	}
	if len(colWidths) != 9 {
		t.Errorf("len(colWidths) = %d, vrem 9 coloane", len(colWidths))
	}

	// A reordering of the same nine widths would still sum to 277 and still
	// have 9 entries, so pin the exact slice too: the column order is part of
	// the form's contract, not just its total width.
	vrem := []float64{13, 70, 16, 26, 28, 30, 30, 28, 36}
	if len(colWidths) == len(vrem) {
		for i := range vrem {
			if colWidths[i] != vrem[i] {
				t.Errorf("colWidths[%d] = %v, vrem %v", i, colWidths[i], vrem[i])
			}
		}
	}

	vremEtichete := []string{
		"Nr. crt.", "DENUMIREA", "U/M", "Cantitatea", "Pret fara T.V.A.",
		"Valoare fara T.V.A.", "Valoare cu T.V.A.", "Pret de vanzare",
		"Valoare la pret de vanzare",
	}
	got := headerRow()
	if len(got) != len(vremEtichete) {
		t.Fatalf("len(headerRow()) = %d, vrem %d", len(got), len(vremEtichete))
	}
	for i := range vremEtichete {
		if got[i] != vremEtichete[i] {
			t.Errorf("headerRow()[%d] = %q, vrem %q", i, got[i], vremEtichete[i])
		}
	}

	// A column added to one and not the other should fail loudly here, not
	// panic later when render zips colWidths against headerRow by index.
	if len(headerRow()) != len(colWidths) {
		t.Errorf("len(headerRow()) = %d != len(colWidths) = %d", len(headerRow()), len(colWidths))
	}
}

func TestNum(t *testing.T) {
	cazuri := []struct {
		in   float64
		vrem string
	}{
		{222, "222,00"},
		{1234.5, "1.234,50"},
		{-11, "-11,00"},
		{0, ""},
	}
	for _, c := range cazuri {
		if got := num(c.in); got != c.vrem {
			t.Errorf("num(%v) = %q, vrem %q", c.in, got, c.vrem)
		}
	}
}

func TestRandulTotalNuAreCantitate(t *testing.T) {
	// U/M variaza de la rand la rand, deci o suma peste cantitati ar fi falsa.
	got := randTotal(doc())
	if got[3] != "" {
		t.Errorf("celula de cantitate din TOTAL = %q, vrem goala", got[3])
	}
	if got[5] == "" || got[6] == "" || got[8] == "" {
		t.Errorf("TOTAL nu are valorile: %q", got)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
