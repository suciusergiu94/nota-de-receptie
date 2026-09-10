/**
 * Reads a figure the way it is written on this form: a comma for the decimals
 * and a dot grouping the thousands, which is what formatNumber puts into the
 * editable cells and what formatLei prints beside them. A plain "1234.5" is
 * read too, because that is how a keyboard with no comma habit types it.
 *
 * A blank field is zero: a quantity nobody filled in is nothing received, and
 * refusing to read it would stop the user from clearing a cell.
 *
 * Anything else — "abc", "1.234.50", a stray minus — comes back undefined
 * rather than as zero. Zero is a figure a reception can legitimately carry, so
 * a silent zero here would be indistinguishable from a deliberate one and
 * would be filed without a word. The callers keep the last readable value on
 * the row instead, and Salvează refuses to file the document while a cell
 * still reads like that, so nothing is saved behind the user's back.
 */
export function parseNumber(input: string): number | undefined {
  const text = input.trim();
  if (text === '') return 0;
  const value = Number(separatoriNormalizati(text));
  return Number.isFinite(value) ? value : undefined;
}

/** Rewrites Romanian separators as the dot decimal point Number() reads. */
function separatoriNormalizati(text: string): string {
  const virgula = text.lastIndexOf(',');
  if (virgula !== -1) {
    // A comma is unambiguous: it is the decimal separator, so every dot in
    // front of it groups thousands and comes out.
    return `${text.slice(0, virgula).replace(/\./g, '')}.${text.slice(virgula + 1)}`;
  }
  // Dots alone are ambiguous. "1.234" is a grouped thousand — it is what this
  // app writes for 1234 — while "1234.5" and "20.00" are decimals typed by
  // hand. Only a full grouping (a leading group of one to three digits, then
  // groups of exactly three) is read as thousands; everything else keeps the
  // dot as the decimal point.
  if (/^-?\d{1,3}(\.\d{3})+$/.test(text)) return text.replace(/\./g, '');
  return text;
}

/**
 * Renders a number the way the whole app writes figures: a comma for the
 * decimals and a dot grouping the thousands.
 *
 * The editable cells are filled with this, and parseNumber reads it back
 * unchanged, so a figure that is typed, saved and shown again does not drift.
 * The two notations used to differ — a price field read "20.00" while the
 * value beside it read "1.234,50" — and copying the app's own thousands
 * format into a price field then produced a figure it could not read.
 */
export function formatNumber(value: number, decimals = 2): string {
  const [intreg, zecimale] = Math.abs(value).toFixed(decimals).split('.');
  const grupat = intreg.replace(/\B(?=(\d{3})+$)/g, '.');
  const semn = value < 0 ? '-' : '';
  return zecimale === undefined ? `${semn}${grupat}` : `${semn}${grupat},${zecimale}`;
}

/**
 * Renders a number for a cell the user is the one to fill in: blank when there
 * is nothing filled in yet.
 *
 * A fresh row starts every figure at zero, and formatNumber would put "0,00"
 * into each of its cells — a figure nobody typed, which has to be selected and
 * deleted before the real one can go in. parseNumber reads a blank cell back
 * as zero, so the note carries the same figure either way; only the screen
 * changes.
 *
 * Cells that come filled in from data keep formatNumber, zero included: the
 * T.V.A. rate from Setări, the selling price of a product picked from the
 * catalogue, every figure on a row imported from a proces verbal. There a
 * zero is an answer the app is giving, not an empty field.
 */
export function formatInput(value: number): string {
  return value === 0 ? '' : formatNumber(value);
}

/** Renders an ISO date (YYYY-MM-DD) the way the paper form writes it. */
export function formatDateRO(iso: string): string {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(iso);
  if (!match) return iso;
  const [, year, month, day] = match;
  return `${day}/${month}/${year}`;
}

/**
 * Reads a date the way it is typed on the form (ZZ/LL/AAAA) back into the ISO
 * YYYY-MM-DD the document stores. A dot or a dash is accepted in place of the
 * slash and the day and month may be typed without a leading zero, because
 * that is how people write dates by hand; anything that is not a real calendar
 * date (31/02, month 13, a half-typed value) returns undefined.
 */
export function parseDateRO(input: string): string | undefined {
  const match = /^(\d{1,2})[./-](\d{1,2})[./-](\d{4})$/.exec(input.trim());
  if (!match) return undefined;
  const [, day, month, year] = match;
  const iso = `${year}-${month.padStart(2, '0')}-${day.padStart(2, '0')}`;
  // Round-trips through Date to reject days that the month does not have:
  // new Date('2026-02-31') rolls over to March, so the ISO text comes back
  // different from what went in.
  const parsed = new Date(`${iso}T00:00:00Z`);
  if (Number.isNaN(parsed.getTime()) || parsed.toISOString().slice(0, 10) !== iso) {
    return undefined;
  }
  return iso;
}

/**
 * Renders a money amount: formatNumber with the two decimals money always
 * carries, named for what it is used for.
 *
 * Zero is written out rather than left blank. On screen an empty cell reads as
 * "not filled in yet", and a genuine zero — goods received for nothing — has to
 * be distinguishable from that. The PDF makes the opposite choice, for the
 * opposite reason: an empty cell on paper is how the form is meant to look.
 */
export function formatLei(value: number): string {
  return formatNumber(value, 2);
}

/**
 * Renders a markup percentage, or a dash when there is none to render — the
 * purchase value was zero, so the percentage has no value. A zero there would
 * read as "no markup", which is a different and wrong statement.
 */
export function formatProcent(value: number | undefined): string {
  if (value === undefined) return '—';
  return `${value.toFixed(2).replace('.', ',')} %`;
}
