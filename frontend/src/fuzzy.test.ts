import { describe, expect, it } from 'vitest';
import { cautaDupaNumar, cautaProduse, normalizeaza, scorFuzzy } from './fuzzy';

const produse = [
  { denumire: 'Pulpă fără os' },
  { denumire: 'Pulpă cu os' },
  { denumire: 'Ceafă fără os' },
  { denumire: 'Cotlet cu os' },
  { denumire: 'Ouă' },
];

describe('normalizeaza', () => {
  it('scoate diacriticele si majusculele', () => {
    expect(normalizeaza('Pulpă Fără Os')).toBe('pulpa fara os');
    expect(normalizeaza('ȘUNCĂ ȚĂRĂNEASCĂ')).toBe('sunca taraneasca');
  });
});

describe('scorFuzzy', () => {
  it('nu potriveste ce nu e subsecventa', () => {
    expect(scorFuzzy('pulpa fara os', 'zzz')).toBeUndefined();
  });

  it('potriveste literele in ordine, chiar daca sar peste altele', () => {
    expect(scorFuzzy('pulpa fara os', 'pfo')).toBeDefined();
  });

  it('da scor mai bun potrivirii de la inceput decat celei din mijloc', () => {
    const laInceput = scorFuzzy('pulpa fara os', 'pulpa')!;
    const inMijloc = scorFuzzy('ceafa pulpa', 'pulpa')!;
    expect(laInceput).toBeLessThan(inMijloc);
  });

  it('da scor mai bun potrivirii compacte decat celei imprastiate', () => {
    const compact = scorFuzzy('pulpa fara os', 'pul')!;
    const imprastiat = scorFuzzy('pulpa fara os', 'pas')!;
    expect(compact).toBeLessThan(imprastiat);
  });
});

describe('cautaProduse', () => {
  it('intoarce catalogul in ordinea lui cand cautarea e goala', () => {
    expect(cautaProduse(produse, '').map((p) => p.denumire)).toEqual(
      produse.map((p) => p.denumire),
    );
  });

  it('gaseste fara diacritice ce e scris cu diacritice', () => {
    const gasite = cautaProduse(produse, 'pulpa fara');
    expect(gasite[0].denumire).toBe('Pulpă fără os');
  });

  it('gaseste dupa initiale', () => {
    const gasite = cautaProduse(produse, 'cfo');
    expect(gasite[0].denumire).toBe('Ceafă fără os');
  });

  it('nu intoarce nimic pentru o cautare fara potriviri', () => {
    expect(cautaProduse(produse, 'zzzz')).toEqual([]);
  });

  it('respecta limita', () => {
    expect(cautaProduse(produse, 'o', 2)).toHaveLength(2);
  });
});

describe('cautaDupaNumar', () => {
  // Numerele nu vin in ordine crescatoare cu data: lista soseste cu cel mai
  // recent proces verbal primul, oricare i-ar fi numarul.
  const procese = [
    { nr: 25 },
    { nr: 15 },
    { nr: 7 },
    { nr: 5 },
    { nr: 51 },
    { nr: 3 },
    { nr: 2 },
  ];

  it('intoarce primele cateva in ordinea listei cand cautarea e goala', () => {
    expect(cautaDupaNumar(procese, '').map((p) => p.nr)).toEqual([25, 15, 7, 5, 51]);
  });

  it('gaseste orice numar care contine cifrele tastate', () => {
    expect(cautaDupaNumar(procese, '5').map((p) => p.nr)).toEqual([51, 25, 15, 5]);
  });

  it('cere cifrele una langa alta, nu imprastiate prin numar', () => {
    // 51 contine "5" si "1", dar nu "15": altfel o cautare de doua cifre ar
    // scoate jumatate din lista.
    expect(cautaDupaNumar(procese, '15').map((p) => p.nr)).toEqual([15]);
  });

  it('nu taie potrivirile la limita: cine cauta vrea sa vada tot', () => {
    expect(cautaDupaNumar(procese, '', 2)).toHaveLength(2);
    expect(cautaDupaNumar(procese, '5')).toHaveLength(4);
  });

  it('ignora ce nu e cifra, ca sa mearga si „nr. 15”', () => {
    expect(cautaDupaNumar(procese, 'nr. 15').map((p) => p.nr)).toEqual([15]);
  });

  it('nu intoarce nimic pentru un numar care nu exista', () => {
    expect(cautaDupaNumar(procese, '999')).toEqual([]);
  });
});
