package pdfdoc

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-pdf/fpdf"

	"nota-de-receptie/internal/calc"
	"nota-de-receptie/internal/model"
)

// colWidths are the product table's column widths in mm, summing to the 277mm
// printable width of a landscape A4 with 10mm margins. Landscape is what makes
// nine columns readable; portrait would squeeze the denumire to nothing.
var colWidths = []float64{13, 70, 16, 26, 28, 30, 30, 28, 36}

// livrareWidths are the delivery table's four columns, over the same 277mm.
var livrareWidths = []float64{80, 45, 52, 100}

const (
	marginLeft = 10.0
	rowHeight  = 5.5
)

// Render draws doc onto landscape A4 pages and returns the PDF bytes.
func Render(doc model.Document) ([]byte, error) {
	var buf bytes.Buffer
	if err := construieste(doc).Output(&buf); err != nil {
		return nil, fmt.Errorf("generare PDF: %w", err)
	}
	return buf.Bytes(), nil
}

// construieste draws the whole document and hands back the fpdf document
// itself. Render only serialises it; keeping the two apart lets a test ask how
// many pages a document came to, which reading the serialised bytes cannot
// answer without depending on how fpdf lays them out.
func construieste(doc model.Document) *fpdf.Fpdf {
	pdf := fpdf.New("L", "mm", "A4", "")
	pdf.SetMargins(marginLeft, 10, marginLeft)
	pdf.SetAutoPageBreak(true, 12)
	pdf.AddPage()

	drawHeader(pdf, doc)
	drawLivrare(pdf, doc)
	pdf.Ln(6)
	drawProduse(pdf, doc)
	drawFooter(pdf)
	return pdf
}

// drawHeader draws the unit line first and the title under it, which is the
// order the paper form has: whose reception this is, then what the sheet is.
func drawHeader(pdf *fpdf.Fpdf, doc model.Document) {
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(150, 6, "UNITATEA: "+Fold(doc.Unitate), "", 0, "L", false, 0, "")
	pdf.CellFormat(60, 6, fmt.Sprintf("nr. %d", doc.Nr), "", 0, "L", false, 0, "")
	pdf.CellFormat(67, 6, "din "+formatDate(doc.Data), "", 1, "L", false, 0, "")

	pdf.Ln(2)
	pdf.SetFont("Arial", "B", 15)
	pdf.CellFormat(277, 8, "NOTA DE RECEPTIE", "", 1, "C", false, 0, "")
	pdf.Ln(3)
}

// drawLivrare draws the delivery table: header row, then the single row of
// values. Cod fiscal and Achitat cu are gone from the form, so four columns
// are all there is.
func drawLivrare(pdf *fpdf.Fpdf, doc model.Document) {
	header := []string{"DOCUMENT LIVRARE", "NR.", "DATA", "FURNIZORUL"}
	valori := []string{
		Fold(doc.DocumentLivrare),
		Fold(doc.DocumentLivrareNr),
		formatDate(doc.DocumentLivrareData),
		Fold(doc.Furnizor),
	}

	pdf.SetFont("Arial", "B", 9)
	pdf.SetX(marginLeft)
	for i, cell := range header {
		pdf.CellFormat(livrareWidths[i], rowHeight+1, cell, "1", 0, "C", false, 0, "")
	}
	pdf.Ln(-1)

	pdf.SetFont("Arial", "", 9)
	pdf.SetX(marginLeft)
	for i, cell := range valori {
		pdf.CellFormat(livrareWidths[i], rowHeight+1, cell, "1", 0, "C", false, 0, "")
	}
	pdf.Ln(-1)
}

func headerRow() []string {
	return []string{
		"Nr. crt.", "DENUMIREA", "U/M", "Cantitatea", "Pret fara T.V.A.",
		"Valoare fara T.V.A.", "Valoare cu T.V.A.", "Pret de vanzare",
		"Valoare la pret de vanzare",
	}
}

// randCells renders one product row. The adaos is absent by design: it is a
// working figure for the screen, and the form it is printed on has no column
// for it.
func randCells(index int, r model.Rand) []string {
	v := calc.ValoriRand(r)
	return []string{
		strconv.Itoa(index + 1),
		Fold(r.Denumire),
		Fold(r.UM),
		num(r.Cantitate),
		num(r.PretFaraTVA),
		num(v.ValoareFaraTVA),
		num(v.ValoareCuTVA),
		num(r.PretVanzare),
		num(v.ValoareVanzare),
	}
}

// randTotal renders the TOTAL row. The quantity cell stays empty: U/M is
// "Buc." on one row and "Kg." on the next, so a sum over them would be a
// figure nobody could defend on a signed document.
func randTotal(doc model.Document) []string {
	t := calc.Totaluri(doc.Randuri)
	return []string{
		"", "TOTAL", "", "", "",
		num(t.ValoareFaraTVA), num(t.ValoareCuTVA), "", num(t.ValoareVanzare),
	}
}

func drawProduse(pdf *fpdf.Fpdf, doc model.Document) {
	ensureHeaderFits(pdf)
	drawTableHeader(pdf)

	pdf.SetFont("Arial", "", 8)
	for i, r := range doc.Randuri {
		ensureRowFits(pdf)
		pdf.SetFont("Arial", "", 8)
		pdf.SetX(marginLeft)
		for j, cell := range randCells(i, r) {
			pdf.CellFormat(colWidths[j], rowHeight, cell, "1", 0, align(j), false, 0, "")
		}
		pdf.Ln(-1)
	}

	ensureRowFits(pdf)
	pdf.SetFont("Arial", "B", 8)
	pdf.SetX(marginLeft)
	for j, cell := range randTotal(doc) {
		pdf.CellFormat(colWidths[j], rowHeight, cell, "1", 0, align(j), false, 0, "")
	}
	pdf.Ln(-1)
}

// align keeps the denumire left and every figure right, which is how a column
// of numbers is read.
func align(col int) string {
	switch col {
	case 1:
		return "L"
	case 0, 2:
		return "C"
	default:
		return "R"
	}
}

// drawTableHeader draws one instance of the column header row. It is drawn at
// twice a body row's height on purpose: the labels are set two points smaller
// than the figures, and the extra air is what keeps a nine-column band of them
// readable. Nothing wraps — CellFormat cannot — and every label fits its
// column, the widest ("Valoare la pret de vanzare") by some 5mm.
func drawTableHeader(pdf *fpdf.Fpdf) {
	pdf.SetFont("Arial", "B", 7)
	pdf.SetX(marginLeft)
	for i, cell := range headerRow() {
		pdf.CellFormat(colWidths[i], rowHeight*2, cell, "1", 0, "C", false, 0, "")
	}
	pdf.Ln(-1)
}

// ensureRowFits forces a page break, and redraws the column header on the new
// page, if a single row would not fit above the bottom margin. It is a no-op
// while the current page still has room, so a one-page document is unaffected.
func ensureRowFits(pdf *fpdf.Fpdf) {
	if !incape(pdf, rowHeight) {
		pdf.AddPage()
		drawTableHeader(pdf)
	}
}

// ensureHeaderFits breaks the page if the header row itself would not fit. It
// does not draw the header — the caller does that immediately after — so it
// never duplicates it the way ensureRowFits' redraw would.
func ensureHeaderFits(pdf *fpdf.Fpdf) {
	if !incape(pdf, rowHeight*2) {
		pdf.AddPage()
	}
}

func incape(pdf *fpdf.Fpdf, inaltime float64) bool {
	_, pageHeight := pdf.GetPageSize()
	_, _, _, bottom := pdf.GetMargins()
	return pdf.GetY()+inaltime <= pageHeight-bottom
}

// drawFooter draws the three signature labels. They carry no names: the form
// is signed by hand, so the app has no field for them.
func drawFooter(pdf *fpdf.Fpdf) {
	pdf.Ln(10)
	pdf.SetFont("Arial", "B", 10)
	pdf.SetX(marginLeft)
	pdf.CellFormat(92, 6, "COMISIA DE RECEPTIE,", "", 0, "C", false, 0, "")
	pdf.CellFormat(92, 6, "INTOCMIT,", "", 0, "C", false, 0, "")
	pdf.CellFormat(93, 6, "GESTIONAR,", "", 1, "C", false, 0, "")
}

// num renders a money or quantity value the Romanian way: comma for the
// decimal separator, dot grouping the thousands. Zero renders blank, the way
// the paper form leaves unused cells empty — the opposite of the on-screen
// choice, and deliberately so.
func num(v float64) string {
	r := calc.Round2(v)
	if r == 0 {
		return ""
	}
	neg := r < 0
	if neg {
		r = -r
	}
	s := strconv.FormatFloat(r, 'f', 2, 64)
	dot := strings.IndexByte(s, '.')
	intPart, decPart := groupThousands(s[:dot]), s[dot+1:]
	if neg {
		return "-" + intPart + "," + decPart
	}
	return intPart + "," + decPart
}

// groupThousands inserts a "." every three digits from the right. It works on
// the integer part alone, called before the decimal part is glued back on, so
// grouping can never reach into the decimals.
func groupThousands(digits string) string {
	n := len(digits)
	if n <= 3 {
		return digits
	}
	lead := n % 3
	if lead == 0 {
		lead = 3
	}
	var b strings.Builder
	b.WriteString(digits[:lead])
	for i := lead; i < n; i += 3 {
		b.WriteByte('.')
		b.WriteString(digits[i : i+3])
	}
	return b.String()
}

// formatDate turns an ISO date into the dd/mm/yyyy the form is written in. An
// empty or malformed value comes back as it went in — the delivery date is
// optional, and printing a placeholder date would be worse than printing none.
func formatDate(iso string) string {
	if len(iso) != 10 {
		return iso
	}
	return iso[8:10] + "/" + iso[5:7] + "/" + iso[0:4]
}
