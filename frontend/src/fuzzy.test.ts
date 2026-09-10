import { describe, expect, it } from 'vitest';
import { cautaProduse, normalizeaza, scorFuzzy } from './fuzzy';

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
