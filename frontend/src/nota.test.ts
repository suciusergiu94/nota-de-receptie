import { describe, expect, it } from 'vitest';
import type { Document, Product } from './api';
import { UM_PERMISE, randDinProdus, randGol, validareDocument } from './nota';

function docValid(): Document {
  return {
    id: 0,
    nr: 1,
    data: '2026-09-09',
    unitate: 'S.C. Largiana Carn S.R.L.',
    documentLivrare: 'Factură',
    documentLivrareNr: '1234',
    documentLivrareData: '2026-09-08',
    furnizor: 'Alfa SRL',
    createdAt: '',
    updatedAt: '',
    randuri: [
      {
        id: 0, productId: undefined, pozitie: 0, denumire: 'Pulpă fără os',
        um: 'Kg.', cantitate: 10, pretFaraTva: 20, cotaTva: 11, pretVanzare: 30,
      },
    ],
  } as unknown as Document;
}

describe('randGol', () => {
  it('porneste de la cota implicita si de la Kg.', () => {
    const r = randGol(11);
    expect(r.cotaTva).toBe(11);
    expect(r.um).toBe('Kg.');
    expect(r.denumire).toBe('');
    expect(r.cantitate).toBe(0);
    expect(r.productId).toBeUndefined();
  });
});

describe('randDinProdus', () => {
  it('copiaza denumirea, unitatea, pretul de vanzare si cota', () => {
    const p = {
      id: 7, denumire: 'Ouă', um: 'Buc.', pretVanzare: 1.5, cotaTva: 11, ordine: 0,
    } as unknown as Product;
    const r = randDinProdus(p);
    expect(r.productId).toBe(7);
    expect(r.denumire).toBe('Ouă');
    expect(r.um).toBe('Buc.');
    expect(r.pretVanzare).toBe(1.5);
    expect(r.cotaTva).toBe(11);
    // Pretul de achizitie nu vine de la produs: variaza de la livrare la livrare.
    expect(r.pretFaraTva).toBe(0);
    expect(r.cantitate).toBe(0);
  });
});

describe('validareDocument', () => {
  it('accepta un document complet', () => {
    expect(validareDocument(docValid(), '09/09/2026')).toBeUndefined();
  });

  it('refuza un numar mai mic de 1', () => {
    const d = docValid();
    d.nr = 0;
    expect(validareDocument(d, '09/09/2026')).toMatch(/număr/i);
  });

  it('refuza o data care nu exista', () => {
    expect(validareDocument(docValid(), '31/02/2026')).toMatch(/dat/i);
  });

  it('refuza un document fara randuri', () => {
    const d = docValid();
    d.randuri = [];
    expect(validareDocument(d, '09/09/2026')).toMatch(/produs/i);
  });

  it('refuza un rand fara denumire si spune care', () => {
    const d = docValid();
    d.randuri[0].denumire = '   ';
    const problema = validareDocument(d, '09/09/2026');
    expect(problema).toMatch(/rândul 1/i);
  });

  it('refuza o unitate de masura pe care formularul n-o cunoaste', () => {
    const d = docValid();
    d.randuri[0].um = 'litri';
    expect(validareDocument(d, '09/09/2026')).toMatch(/U\/M/i);
  });
});

describe('UM_PERMISE', () => {
  it('are exact cele doua unitati de pe formular', () => {
    expect([...UM_PERMISE]).toEqual(['Buc.', 'Kg.']);
  });
});
