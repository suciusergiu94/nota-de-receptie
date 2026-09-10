package pvt

import "nota-de-receptie/internal/calc"

// Cote splits total across len(valori) shares, in proportion to valori, so
// that the shares add up to exactly Round2(total).
//
// Shares rounded independently miss the total by a ban or two, and here the
// total is the whole point: it is what the proces verbal came to, and the notă
// has to carry that figure and not a near miss. So every share but one is the
// rounded quotient, and the remaining one takes whatever is left. That one is
// the largest, where a couple of bani weigh proportionally least — on a 2500
// lei carcass they vanish; on a 30 lei row they would show.
//
// With all values zero there is no proportion to follow, and the whole total
// goes on the first share rather than being lost.
func Cote(valori []float64, total float64) []float64 {
	out := make([]float64, len(valori))
	if len(valori) == 0 {
		return out
	}
	total = calc.Round2(total)

	var sumaValori float64
	for _, v := range valori {
		sumaValori += v
	}
	if sumaValori == 0 {
		out[0] = total
		return out
	}

	mare := indexulCelMaiMare(valori)
	rest := total
	for i, v := range valori {
		if i == mare {
			continue
		}
		out[i] = calc.Round2(total * v / sumaValori)
		rest = calc.Round2(rest - out[i])
	}
	out[mare] = rest
	return out
}

// indexulCelMaiMare is the position of the largest value, the first of them if
// several tie.
func indexulCelMaiMare(valori []float64) int {
	mare := 0
	for i, v := range valori {
		if v > valori[mare] {
			mare = i
		}
	}
	return mare
}
