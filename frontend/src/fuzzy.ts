/**
 * Searching the product catalogue from the document form.
 *
 * The match is a subsequence match, not a substring one: the catalogue is
 * typed in Romanian with diacritics but searched on a keyboard where they are
 * awkward, and people type initials ("cfo" for "Ceafă fără os") as often as
 * they type prefixes. Both have to find the row.
 */

const DIACRITICE: Record<string, string> = {
  ă: 'a', â: 'a', î: 'i', ș: 's', ş: 's', ț: 't', ţ: 't',
};

/** Lowercases and strips Romanian diacritics, so search ignores both. */
export function normalizeaza(s: string): string {
  return s
    .toLowerCase()
    .replace(/[ăâîșşțţ]/g, (ch) => DIACRITICE[ch] ?? ch)
    .replace(/\s+/g, ' ')
    .trim();
}

/**
 * How well `cautare` matches `text`, as a penalty: smaller is better, and
 * undefined means it does not match at all.
 *
 * The penalty is where the match starts plus how far it had to jump between
 * letters. That ranks a prefix above a match buried in the middle, and a
 * compact match above one scattered across the name — which is the order
 * someone scanning the list expects.
 *
 * Both arguments are normalised here rather than by the caller, so a caller
 * cannot forget to.
 */
export function scorFuzzy(text: string, cautare: string): number | undefined {
  const t = normalizeaza(text);
  const c = normalizeaza(cautare);
  if (c === '') return 0;

  let inceput = -1;
  let ultima = -1;
  let salturi = 0;
  let i = 0;

  for (const litera of c) {
    const gasit = t.indexOf(litera, ultima + 1);
    if (gasit === -1) return undefined;
    if (i === 0) {
      inceput = gasit;
    } else {
      salturi += gasit - ultima - 1;
    }
    ultima = gasit;
    i += 1;
  }
  return inceput * 2 + salturi;
}

/**
 * The products matching `cautare`, best first.
 *
 * An empty search returns the catalogue in its own order — the list the user
 * arranged in Setări — rather than an alphabetised one, so the dropdown opens
 * on something familiar.
 */
export function cautaProduse<T extends { denumire: string }>(
  produse: T[],
  cautare: string,
  limita = 8,
): T[] {
  if (normalizeaza(cautare) === '') return produse.slice(0, limita);

  return produse
    .map((produs) => ({ produs, scor: scorFuzzy(produs.denumire, cautare) }))
    .filter((p): p is { produs: T; scor: number } => p.scor !== undefined)
    .sort((a, b) => a.scor - b.scor || a.produs.denumire.localeCompare(b.produs.denumire, 'ro'))
    .slice(0, limita)
    .map((p) => p.produs);
}

/**
 * The proces verbal documents whose number contains the digits typed, largest
 * number first.
 *
 * Numbers are matched as a substring rather than as the subsequence
 * `scorFuzzy` uses for names: "5" has to bring up 5, 15 and 25, but "15" has
 * no business bringing up 51 — with a couple of hundred documents a scattered
 * digit match returns most of the list, which is no answer at all.
 *
 * `limita` caps only the empty search, where the list opens on the newest few
 * in the order it arrived. A search shows every match: someone who typed a
 * number is looking for one document, and a hidden match reads as "it isn't
 * there".
 */
export function cautaDupaNumar<T extends { nr: number }>(
  procese: T[],
  cautare: string,
  limita = 5,
): T[] {
  const cifre = cautare.replace(/\D/g, '');
  if (cifre === '') return procese.slice(0, limita);

  return procese.filter((p) => String(p.nr).includes(cifre)).sort((a, b) => b.nr - a.nr);
}
