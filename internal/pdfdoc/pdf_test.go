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

func TestFoldInlocuiestePunctuatiaDinAltePrograme(t *testing.T) {
	// Un nume copiat dintr-un editor de text vine cu liniuta lunga, ghilimele
	// rotunde si spatiu insecabil. Fonturile de baza ale PDF-ului sunt
	// Latin-1 si primesc octetii netradusi, deci orice ramane peste ASCII se
	// tipareste ca octeti bruti.
	cazuri := []struct{ in, vrem string }{
		{"Alfa \u2014 Prod S.R.L.", "Alfa - Prod S.R.L."},
		{"\u201cAlfa\u201d Prod", "\"Alfa\" Prod"},
		{"Alfa\u00a0Prod", "Alfa Prod"},
		{"Alfa\u2026", "Alfa..."},
		{"Caf\u00e9 M\u00fcller & S\u00f8n", "Cafe Muller & Son"},
		{"Str. Mor\u021bii nr. 3", "Str. Mortii nr. 3"},
	}
	for _, c := range cazuri {
		if got := Fold(c.in); got != c.vrem {
			t.Errorf("Fold(%q) = %q, vrem %q", c.in, got, c.vrem)
		}
	}
}

func TestFoldNuLasaNimicPesteASCII(t *testing.T) {
	// Un semn fara echivalent devine un singur "?", iar un sir intreg de
	// astfel de semne tot unul: celula arata ca s-a pierdut ceva, fara sa se
	// umple de semne de intrebare si fara sa ramana goala.
	cazuri := []struct{ in, vrem string }{
		{"\u041a\u043e\u0432\u0430\u043b\u0451\u0432", "?"},
		{"Alfa \u4e2d\u6587 SRL", "Alfa ? SRL"},
		{"\U0001f600", "?"},
		{"linia 1\nlinia 2", "linia 1 linia 2"},
	}
	for _, c := range cazuri {
		if got := Fold(c.in); got != c.vrem {
			t.Errorf("Fold(%q) = %q, vrem %q", c.in, got, c.vrem)
		}
	}

	for _, r := range Fold("a\u2014b\u00e9c\u041a\u0434\U0001f600\u00a0") {
		if r < 0x20 || r > 0x7e {
			t.Errorf("Fold a lasat %q (U+%04X), vrem doar ASCII tiparibil", r, r)
		}
	}
}

func TestAntetulAreUnitateaInainteaTitlului(t *testing.T) {
	// Sectiunea 7 din specificatie fixeaza ordinea paginii: antetul
	// (Unitatea, nr., din data), apoi titlul. Ordinea se citeste din fluxul
	// de continut necomprimat, singurul loc unde chiar apare ce s-a desenat.
	pdf := construieste(doc())
	pdf.SetCompression(false)
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		t.Fatalf("Output: %v", err)
	}
	unitatea := bytes.Index(buf.Bytes(), []byte("UNITATEA"))
	titlu := bytes.Index(buf.Bytes(), []byte("NOTA DE RECEPTIE"))
	if unitatea == -1 || titlu == -1 {
		t.Fatalf("nu am gasit antetul in PDF: unitatea = %d, titlu = %d", unitatea, titlu)
	}
	if unitatea > titlu {
		t.Errorf("titlul e desenat inaintea unitatii (%d > %d)", unitatea, titlu)
	}
}

func TestEtichetelePotrivescInColoane(t *testing.T) {
	// Randul de antet al tabelului e desenat la doua randuri inaltime pentru
	// lizibilitate, nu pentru ca s-ar rupe pe doua linii: CellFormat nu rupe
	// niciodata textul, deci o eticheta mai lata decat coloana ei ar iesi
	// peste vecina, tacut.
	pdf := construieste(doc())
	pdf.SetFont("Arial", "B", 7)
	for i, eticheta := range headerRow() {
		if l := pdf.GetStringWidth(eticheta); l > colWidths[i] {
			t.Errorf("eticheta %q are %.1f mm, coloana are %.1f mm", eticheta, l, colWidths[i])
		}
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

func TestTabelulTipareteValoareaImpusaSiTotalulEi(t *testing.T) {
	// Randul importat dintr-un proces verbal isi poarta propria valoare la pret
	// de vanzare. Inmultirea cantitate x pret ar da 2501.12; pe hartie trebuie
	// sa apara totalul procesului verbal, si el trebuie sa intre ca atare in
	// randul TOTAL.
	impusa := 2501.35
	d := model.Document{
		Nr: 7, Data: "2026-09-09", Unitate: "S.C. Largiana Carn S.R.L.",
		Randuri: []model.Rand{
			{Pozitie: 0, Denumire: "Oua", UM: "Buc.", Cantitate: 30,
				PretFaraTVA: 0.9, CotaTVA: 11, PretVanzare: 1.5},
			{Pozitie: 1, Denumire: "Carcasa", UM: "Kg.", Cantitate: 162.2,
				PretFaraTVA: 12.30, CotaTVA: 11, PretVanzare: 15.42,
				ValoareVanzareImpusa: &impusa},
		},
	}

	celule := randCells(1, d.Randuri[1])
	if celule[8] != "2.501,35" {
		t.Errorf("valoarea la pret de vanzare = %q, vrem \"2.501,35\"", celule[8])
	}
	if celule[7] != "15,42" {
		t.Errorf("pretul de vanzare = %q, vrem \"15,42\"", celule[7])
	}

	total := randTotal(d)
	if total[8] != "2.546,35" {
		t.Errorf("totalul la pret de vanzare = %q, vrem \"2.546,35\"", total[8])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
