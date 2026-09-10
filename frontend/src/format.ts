/** Parses user input that may use either a dot or a comma decimal separator. */
export function parseNumber(input: string): number {
  const normalised = input.trim().replace(',', '.');
  if (normalised === '') return 0;
  const value = Number(normalised);
  return Number.isFinite(value) ? value : 0;
}

/** Renders a number with a fixed number of decimals (2 by default). */
export function formatNumber(value: number, decimals = 2): string {
  return value.toFixed(decimals);
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
 * Renders a money amount the Romanian way: two decimals, a comma for the
 * decimal separator and a dot grouping the thousands.
 *
 * Zero is written out rather than left blank. On screen an empty cell reads as
 * "not filled in yet", and a genuine zero — goods received for nothing — has to
 * be distinguishable from that. The PDF makes the opposite choice, for the
 * opposite reason: an empty cell on paper is how the form is meant to look.
 */
export function formatLei(value: number): string {
  const [intreg, zecimale] = Math.abs(value).toFixed(2).split('.');
  const grupat = intreg.replace(/\B(?=(\d{3})+$)/g, '.');
  return `${value < 0 ? '-' : ''}${grupat},${zecimale}`;
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
