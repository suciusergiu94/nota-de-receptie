import { describe, expect, it } from 'vitest';
import { round2, totaluri, valoriRand } from './calc';

describe('round2', () => {
  it('rotunjeste la doua zecimale, jumatate departandu-se de zero', () => {
    expect(round2(0.125)).toBe(0.13);
    expect(round2(-0.125)).toBe(-0.13);
    expect(round2(2.344)).toBe(2.34);
    expect(round2(2.345)).toBe(2.35);
    expect(round2(-2.345)).toBe(-2.35);
    expect(round2(0)).toBe(0);
  });
});

describe('valoriRand', () => {
  it('calculeaza exemplul din specificatie', () => {
    const v = valoriRand({ cantitate: 10, pretFaraTva: 20, cotaTva: 11, pretVanzare: 30 });
    expect(v.valoareFaraTva).toBe(200);
    expect(v.valoareCuTva).toBe(222);
    expect(v.valoareVanzare).toBe(300);
    expect(v.adaos).toBe(78);
    expect(v.adaosProcent).toBe(35.14);
  });

  it('da adaos negativ cand se vinde sub pretul de achizitie', () => {
    const v = valoriRand({ cantitate: 1, pretFaraTva: 100, cotaTva: 11, pretVanzare: 100 });
    expect(v.adaos).toBe(-11);
    expect(v.adaosProcent).toBe(-9.91);
  });

  it('nu da procent cand valoarea cu TVA e zero', () => {
    const v = valoriRand({ cantitate: 5, pretFaraTva: 0, cotaTva: 11, pretVanzare: 4 });
    expect(v.valoareCuTva).toBe(0);
    expect(v.adaos).toBe(20);
    expect(v.adaosProcent).toBeUndefined();
  });

  it('aplica cota fiecarui rand in parte', () => {
    const v = valoriRand({ cantitate: 2, pretFaraTva: 50, cotaTva: 21, pretVanzare: 70 });
    expect(v.valoareCuTva).toBe(121);
    expect(v.adaos).toBe(19);
  });

  it('foloseste valoarea de vanzare impusa in locul inmultirii', () => {
    const v = valoriRand({
      cantitate: 162.2,
      pretFaraTva: 12.3,
      cotaTva: 11,
      pretVanzare: 15.42,
      valoareVanzareImpusa: 2501.35,
    });
    expect(v.valoareFaraTva).toBe(1995.06);
    expect(v.valoareCuTva).toBe(2214.52);
    expect(v.valoareVanzare).toBe(2501.35);
    expect(v.adaos).toBe(286.83);
    expect(v.adaosProcent).toBe(12.95);
  });

  it('trateaza null si undefined ca "nimic impus"', () => {
    const cuNull = valoriRand({
      cantitate: 10, pretFaraTva: 5, cotaTva: 11, pretVanzare: 8,
      valoareVanzareImpusa: null,
    });
    expect(cuNull.valoareVanzare).toBe(80);

    const fara = valoriRand({ cantitate: 10, pretFaraTva: 5, cotaTva: 11, pretVanzare: 8 });
    expect(fara.valoareVanzare).toBe(80);
  });

  it('deosebeste zero impus de lipsa valorii impuse', () => {
    const v = valoriRand({
      cantitate: 10, pretFaraTva: 5, cotaTva: 11, pretVanzare: 8,
      valoareVanzareImpusa: 0,
    });
    expect(v.valoareVanzare).toBe(0);
  });
});

describe('totaluri', () => {
  it('insumeaza randurile si calculeaza procentul din totaluri', () => {
    const t = totaluri([
      { cantitate: 10, pretFaraTva: 20, cotaTva: 11, pretVanzare: 30 },
      { cantitate: 2, pretFaraTva: 50, cotaTva: 21, pretVanzare: 70 },
    ]);
    expect(t.valoareFaraTva).toBe(300);
    expect(t.valoareCuTva).toBe(343);
    expect(t.valoareVanzare).toBe(440);
    expect(t.adaos).toBe(97);
    // Din totaluri, nu media procentelor de pe randuri (35,14 si 15,70).
    expect(t.adaosProcent).toBe(28.28);
  });

  it('da zero si niciun procent pe lista goala', () => {
    const t = totaluri([]);
    expect(t.valoareCuTva).toBe(0);
    expect(t.adaosProcent).toBeUndefined();
  });

  it('insumeaza si randurile cu valoare impusa', () => {
    const tot = totaluri([
      { cantitate: 10, pretFaraTva: 20, cotaTva: 11, pretVanzare: 30 },
      {
        cantitate: 162.2, pretFaraTva: 12.3, cotaTva: 11, pretVanzare: 15.42,
        valoareVanzareImpusa: 2501.35,
      },
    ]);
    expect(tot.valoareVanzare).toBe(2801.35);
  });
});
