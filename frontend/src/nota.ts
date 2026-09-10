import type { Document, Product, Rand } from './api';
import { parseDateRO } from './format';

/**
 * The units the form accepts. The paper form's U/M column is free-hand, but
 * this app writes it, and two values are all the business uses — restricting
 * it keeps a document from carrying "kg", "Kg", "kilograme" and "Kg." as four
 * different things.
 */
export const UM_PERMISE = ['Buc.', 'Kg.'] as const;

/** A blank row, ready to be typed into. */
export function randGol(cotaImplicita: number): Rand {
  return {
    id: 0,
    productId: undefined,
    pozitie: 0,
    denumire: '',
    um: 'Kg.',
    cantitate: 0,
    pretFaraTva: 0,
    cotaTva: cotaImplicita,
    pretVanzare: 0,
  } as unknown as Rand;
}

/**
 * A row starting from a catalogue product: name, unit, selling price and rate
 * are copied in, and from this point the row owns them. Editing the product
 * later leaves this row alone.
 *
 * The purchase price is not copied, because the product does not carry one:
 * it is what this particular delivery cost, and it is typed in at reception.
 */
export function randDinProdus(p: Product): Rand {
  return {
    id: 0,
    productId: p.id,
    pozitie: 0,
    denumire: p.denumire,
    um: p.um,
    cantitate: 0,
    pretFaraTva: 0,
    cotaTva: p.cotaTva,
    pretVanzare: p.pretVanzare,
  } as unknown as Rand;
}

/**
 * The first thing wrong with the document, or undefined if nothing is.
 *
 * Both dates are checked as they were typed rather than as they are stored: a
 * half-typed "09/09/20" never becomes an ISO date, so the form only writes a
 * date onto the document once it parses. Checking the stored value would
 * therefore pass the last valid date while the field on screen reads
 * something else entirely, and the PDF would print a date nobody chose.
 */
export function validareDocument(
  doc: Document,
  dataTastata: string,
  dataLivrareTastata: string,
): string | undefined {
  if (!Number.isInteger(doc.nr) || doc.nr < 1) {
    return 'Numărul notei trebuie să fie un număr întreg, cel puțin 1.';
  }
  if (parseDateRO(dataTastata) === undefined) {
    return 'Data notei nu este o dată validă. Se scrie ZZ/LL/AAAA.';
  }
  // The delivery date is optional — not every reception has a document to
  // reference — but anything typed into it has to be a real date.
  if (dataLivrareTastata.trim() !== '' && parseDateRO(dataLivrareTastata) === undefined) {
    return 'Data documentului de livrare nu este o dată validă. Se scrie ZZ/LL/AAAA.';
  }
  if (doc.randuri.length === 0) {
    return 'Nota trebuie să aibă cel puțin un produs.';
  }
  for (const [i, r] of doc.randuri.entries()) {
    if (r.denumire.trim() === '') {
      return `Rândul ${i + 1} nu are denumire.`;
    }
    if (!(UM_PERMISE as readonly string[]).includes(r.um)) {
      return `Rândul ${i + 1}: U/M trebuie să fie „Buc.” sau „Kg.”.`;
    }
  }
  return undefined;
}
