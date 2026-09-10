package pvt

import (
	"math"
	"testing"

	"nota-de-receptie/internal/calc"
)

func aproape(t *testing.T, nume string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("%s = %v, vrem %v", nume, got, want)
	}
}

func suma(v []float64) float64 {
	var s float64
	for _, x := range v {
		s += x
	}
	return calc.Round2(s)
}

func TestCoteCuUnSingurRand(t *testing.T) {
	// Cazul obisnuit: o carcasa, tot totalul pe randul ei.
	got := Cote([]float64{1995.06}, 2501.35)
	if len(got) != 1 {
		t.Fatalf("cote = %d, vrem 1", len(got))
	}
	aproape(t, "cota", got[0], 2501.35)
}

func TestCoteImpartProportional(t *testing.T) {
	got := Cote([]float64{300, 100}, 400)
	aproape(t, "cota[0]", got[0], 300)
	aproape(t, "cota[1]", got[1], 100)
	aproape(t, "suma", suma(got), 400)
}

func TestCoteDauSumaExactaCandImpartireaNuIese(t *testing.T) {
	// 100 impartit in trei parti egale nu se poate scrie in bani. Restul
	// merge la randul cel mai mare, si suma trebuie sa fie exact 100.
	got := Cote([]float64{100, 100, 100}, 100)
	aproape(t, "suma", suma(got), 100)
	aproape(t, "cota[1]", got[1], 33.33)
	aproape(t, "cota[2]", got[2], 33.33)
	aproape(t, "cota[0]", got[0], 33.34)
}

func TestRestulMergeLaRandulCelMaiMare(t *testing.T) {
	got := Cote([]float64{1, 1000}, 100)
	aproape(t, "suma", suma(got), 100)
	// Randul mic primeste cota lui rotunjita; restul cade pe cel mare.
	aproape(t, "cota[0]", got[0], 0.1)
	aproape(t, "cota[1]", got[1], 99.9)
}

func TestCoteCuValoriZero(t *testing.T) {
	// Intrari trecute cu pretul zero: nu exista proportie de urmat, si tot
	// totalul merge pe primul rand in loc sa se piarda.
	got := Cote([]float64{0, 0}, 50)
	aproape(t, "cota[0]", got[0], 50)
	aproape(t, "cota[1]", got[1], 0)
	aproape(t, "suma", suma(got), 50)
}

func TestCoteFaraValori(t *testing.T) {
	if got := Cote(nil, 100); len(got) != 0 {
		t.Errorf("cote = %v, vrem gol", got)
	}
}

func TestCoteCuTotalZero(t *testing.T) {
	got := Cote([]float64{10, 20}, 0)
	aproape(t, "suma", suma(got), 0)
}
