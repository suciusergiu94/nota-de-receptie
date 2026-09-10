/**
 * The arithmetic of the notă de recepție, mirroring internal/calc on the Go
 * side. Both exist because the form recalculates on every keystroke, which
 * cannot go through Wails, while the PDF and the store must not depend on the
 * browser. calc.test.ts uses the same worked examples as calc_test.go, so the
 * two cannot drift apart without a test failing.
 */

/** One row's inputs. */
export interface RandCalculabil {
  cantitate: number;
  pretFaraTva: number;
  cotaTva: number;
  pretVanzare: number;
  /**
   * The row's valoare la preț de vânzare when it came from a proces verbal de
   * transare, instead of the cantitate × preț this file derives for every
   * other row. Null and undefined both mean "nothing imposed": the Go side
   * omits the field when it is nil, but a row built here may carry it null.
   */
  valoareVanzareImpusa?: number | null;
}

/** What a row, or a whole document, comes to. */
export interface Valori {
  valoareFaraTva: number;
  valoareCuTva: number;
  valoareVanzare: number;
  adaos: number;
  /**
   * Undefined when the purchase value is zero: the percentage has no value
   * then, and a zero would read as "no markup", which is a different claim.
   */
  adaosProcent: number | undefined;
}

/** Rounds to two decimals, half away from zero. */
export function round2(v: number): number {
  const r = Math.round(Math.abs(v) * 100) / 100;
  return v < 0 ? -r : r;
}

/** Works out one row's values. */
export function valoriRand(r: RandCalculabil): Valori {
  const faraTva = round2(r.cantitate * r.pretFaraTva);
  const cuTva = round2(faraTva * (1 + r.cotaTva / 100));
  const vanzare =
    r.valoareVanzareImpusa === undefined || r.valoareVanzareImpusa === null
      ? round2(r.cantitate * r.pretVanzare)
      : round2(r.valoareVanzareImpusa);
  return valoriDin(faraTva, cuTva, vanzare);
}

/**
 * Sums a document's rows. Quantity is deliberately absent: U/M is "Buc." on
 * one row and "Kg." on the next, so a sum over them would mean nothing.
 */
export function totaluri(randuri: RandCalculabil[]): Valori {
  let faraTva = 0;
  let cuTva = 0;
  let vanzare = 0;
  for (const r of randuri) {
    const v = valoriRand(r);
    faraTva += v.valoareFaraTva;
    cuTva += v.valoareCuTva;
    vanzare += v.valoareVanzare;
  }
  return valoriDin(round2(faraTva), round2(cuTva), round2(vanzare));
}

/**
 * Derives the adaos and its percentage from three already-rounded values.
 * Both valoriRand and totaluri end here, so a row and a total can never
 * disagree about what an adaos is.
 */
function valoriDin(valoareFaraTva: number, valoareCuTva: number, valoareVanzare: number): Valori {
  const adaos = round2(valoareVanzare - valoareCuTva);
  return {
    valoareFaraTva,
    valoareCuTva,
    valoareVanzare,
    adaos,
    adaosProcent: valoareCuTva === 0 ? undefined : round2((adaos / valoareCuTva) * 100),
  };
}
