package calc

import (
	"math"
	"testing"

	"nota-de-receptie/internal/model"
)

func aproape(t *testing.T, nume string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("%s = %v, vrem %v", nume, got, want)
	}
}

func TestRound2(t *testing.T) {
	cazuri := []struct{ in, want float64 }{
		{0.125, 0.13},
		{-0.125, -0.13},
		{2.344, 2.34},
		{2.345, 2.35},
		{-2.345, -2.35},
		{0, 0},
	}
	for _, c := range cazuri {
		aproape(t, "Round2", Round2(c.in), c.want)
	}
}

func TestValoriRand(t *testing.T) {
	// Exemplul din specificatie: 10 x 20 lei fara TVA, 11%, vandut cu 30 lei.
	v := ValoriRand(model.Rand{
		Cantitate:   10,
		PretFaraTVA: 20,
		CotaTVA:     11,
		PretVanzare: 30,
	})
	aproape(t, "ValoareFaraTVA", v.ValoareFaraTVA, 200)
	aproape(t, "ValoareCuTVA", v.ValoareCuTVA, 222)
	aproape(t, "ValoareVanzare", v.ValoareVanzare, 300)
	aproape(t, "Adaos", v.Adaos, 78)
	aproape(t, "AdaosProcent", v.AdaosProcent, 35.14)
	if !v.AreAdaosProcent {
		t.Error("AreAdaosProcent = false, vrem true")
	}
}

func TestAdaosNegativCandSeVindeSubPretulDeAchizitie(t *testing.T) {
	v := ValoriRand(model.Rand{Cantitate: 1, PretFaraTVA: 100, CotaTVA: 11, PretVanzare: 100})
	aproape(t, "Adaos", v.Adaos, -11)
	aproape(t, "AdaosProcent", v.AdaosProcent, -9.91)
}

func TestFaraAdaosProcentCandValoareaCuTVAEZero(t *testing.T) {
	// Marfa primita gratuit, sau un rand inca necompletat: procentul nu se
	// poate calcula, si nu trebuie sa devina nici infinit, nici zero.
	v := ValoriRand(model.Rand{Cantitate: 5, PretFaraTVA: 0, CotaTVA: 11, PretVanzare: 4})
	aproape(t, "ValoareCuTVA", v.ValoareCuTVA, 0)
	aproape(t, "Adaos", v.Adaos, 20)
	if v.AreAdaosProcent {
		t.Error("AreAdaosProcent = true, vrem false")
	}
	aproape(t, "AdaosProcent", v.AdaosProcent, 0)
}

func TestCoteDiferitePeRanduri(t *testing.T) {
	v := ValoriRand(model.Rand{Cantitate: 2, PretFaraTVA: 50, CotaTVA: 21, PretVanzare: 70})
	aproape(t, "ValoareCuTVA", v.ValoareCuTVA, 121)
	aproape(t, "Adaos", v.Adaos, 19)
}

func TestTotaluri(t *testing.T) {
	randuri := []model.Rand{
		{Cantitate: 10, PretFaraTVA: 20, CotaTVA: 11, PretVanzare: 30},
		{Cantitate: 2, PretFaraTVA: 50, CotaTVA: 21, PretVanzare: 70},
	}
	tot := Totaluri(randuri)
	aproape(t, "ValoareFaraTVA", tot.ValoareFaraTVA, 300)
	aproape(t, "ValoareCuTVA", tot.ValoareCuTVA, 343)
	aproape(t, "ValoareVanzare", tot.ValoareVanzare, 440)
	aproape(t, "Adaos", tot.Adaos, 97)
	// Din totaluri, nu media procentelor de pe randuri (35.14 si 15.70).
	aproape(t, "AdaosProcent", tot.AdaosProcent, 28.28)
}

func TestTotaluriPeListaGoala(t *testing.T) {
	tot := Totaluri(nil)
	aproape(t, "ValoareCuTVA", tot.ValoareCuTVA, 0)
	if tot.AreAdaosProcent {
		t.Error("AreAdaosProcent = true pe lista goala, vrem false")
	}
}

func TestValoareVanzareImpusaInlocuiesteInmultirea(t *testing.T) {
	// Carcasa de 162.2 Kg cumparata cu 12.30 lei/Kg, transata intr-un proces
	// verbal al carui tabel "ce iese" totalizeaza 2501.35 lei. Pretul unitar
	// rotunjit la doua zecimale (15.42) inmultit inapoi ar da 2501.12; nota
	// trebuie sa arate totalul procesului verbal, la ban.
	impusa := 2501.35
	v := ValoriRand(model.Rand{
		Cantitate:            162.2,
		PretFaraTVA:          12.30,
		CotaTVA:              11,
		PretVanzare:          15.42,
		ValoareVanzareImpusa: &impusa,
	})
	aproape(t, "ValoareFaraTVA", v.ValoareFaraTVA, 1995.06)
	aproape(t, "ValoareCuTVA", v.ValoareCuTVA, 2214.52)
	aproape(t, "ValoareVanzare", v.ValoareVanzare, 2501.35)
	aproape(t, "Adaos", v.Adaos, 286.83)
	aproape(t, "AdaosProcent", v.AdaosProcent, 12.95)
}

func TestValoareImpusaZeroNuEAcelasiLucruCuLipsaEi(t *testing.T) {
	// Un proces verbal din care iese numai deseu totalizeaza 0. Zero impus si
	// "nimic impus" trebuie sa dea rezultate diferite, altfel campul nu poate
	// exprima primul caz.
	zero := 0.0
	impus := ValoriRand(model.Rand{
		Cantitate: 10, PretFaraTVA: 5, CotaTVA: 11, PretVanzare: 8,
		ValoareVanzareImpusa: &zero,
	})
	aproape(t, "ValoareVanzare impusa", impus.ValoareVanzare, 0)

	normal := ValoriRand(model.Rand{
		Cantitate: 10, PretFaraTVA: 5, CotaTVA: 11, PretVanzare: 8,
	})
	aproape(t, "ValoareVanzare normala", normal.ValoareVanzare, 80)
}

func TestTotaluriInsumeazaSiRandurileCuValoareImpusa(t *testing.T) {
	impusa := 2501.35
	randuri := []model.Rand{
		{Cantitate: 10, PretFaraTVA: 20, CotaTVA: 11, PretVanzare: 30},
		{Cantitate: 162.2, PretFaraTVA: 12.30, CotaTVA: 11, PretVanzare: 15.42,
			ValoareVanzareImpusa: &impusa},
	}
	tot := Totaluri(randuri)
	aproape(t, "ValoareVanzare", tot.ValoareVanzare, 2801.35)
}
