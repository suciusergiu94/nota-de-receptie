// Package calc holds the arithmetic of the notă de recepție: the values of one
// row and the totals over a document's rows.
package calc

import (
	"math"

	"nota-de-receptie/internal/model"
)

// Valori is what one row, or a whole document, comes to. The same shape serves
// both: a total is the sum of its rows in every field but the percentage,
// which is worked out from the summed figures.
type Valori struct {
	ValoareFaraTVA float64 `json:"valoareFaraTva"`
	ValoareCuTVA   float64 `json:"valoareCuTva"`
	ValoareVanzare float64 `json:"valoareVanzare"`
	Adaos          float64 `json:"adaos"`
	AdaosProcent   float64 `json:"adaosProcent"`
	// AreAdaosProcent is false when the purchase value is zero — goods
	// received for nothing, or a row not filled in yet. The percentage has no
	// value then, and the form shows a dash rather than a zero that would
	// read as "no markup".
	AreAdaosProcent bool `json:"areAdaosProcent"`
}

// Round2 rounds to two decimals, half away from zero.
func Round2(v float64) float64 {
	r := math.Round(math.Abs(v)*100) / 100
	if v < 0 {
		return -r
	}
	return r
}

// ValoriRand works out one row's values.
//
// PretVanzare carries TVA, like ValoareCuTVA, so the adaos subtracts two
// figures stated the same way. Subtracting a price without TVA from one with
// it would produce a number that is neither a margin nor anything else.
func ValoriRand(r model.Rand) Valori {
	faraTVA := Round2(r.Cantitate * r.PretFaraTVA)
	cuTVA := Round2(faraTVA * (1 + r.CotaTVA/100))
	vanzare := Round2(r.Cantitate * r.PretVanzare)
	return valoriDin(faraTVA, cuTVA, vanzare)
}

// Totaluri sums a document's rows.
//
// Quantity is deliberately absent: U/M is "Buc." on one row and "Kg." on the
// next, so a sum over them would mean nothing. The spec drops the quantity
// total from both the screen and the PDF.
func Totaluri(randuri []model.Rand) Valori {
	var faraTVA, cuTVA, vanzare float64
	for _, r := range randuri {
		v := ValoriRand(r)
		faraTVA += v.ValoareFaraTVA
		cuTVA += v.ValoareCuTVA
		vanzare += v.ValoareVanzare
	}
	return valoriDin(Round2(faraTVA), Round2(cuTVA), Round2(vanzare))
}

// valoriDin derives the adaos and its percentage from three already-rounded
// values. Both ValoriRand and Totaluri end here, so a row and a total can
// never disagree about what an adaos is.
func valoriDin(faraTVA, cuTVA, vanzare float64) Valori {
	v := Valori{
		ValoareFaraTVA: faraTVA,
		ValoareCuTVA:   cuTVA,
		ValoareVanzare: vanzare,
		Adaos:          Round2(vanzare - cuTVA),
	}
	if cuTVA != 0 {
		v.AdaosProcent = Round2(v.Adaos / cuTVA * 100)
		v.AreAdaosProcent = true
	}
	return v
}
