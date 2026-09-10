import { describe, expect, it } from 'vitest';
import { formatDateRO, formatLei, formatNumber, formatProcent, parseDateRO, parseNumber } from './format';

describe('parseNumber', () => {
  it('accepts a dot decimal separator', () => {
    expect(parseNumber('162.20')).toBe(162.2);
  });

  it('accepts a comma decimal separator', () => {
    expect(parseNumber('162,20')).toBe(162.2);
  });

  it('citeste formatul romanesc cu punct la mii si virgula la zecimale', () => {
    expect(parseNumber('1.234,50')).toBe(1234.5);
    expect(parseNumber('1.234.567,89')).toBe(1234567.89);
    expect(parseNumber('-1.234,50')).toBe(-1234.5);
  });

  it('citeste un grup de mii fara zecimale', () => {
    expect(parseNumber('1.234')).toBe(1234);
  });

  it('treats blank input as zero', () => {
    expect(parseNumber('')).toBe(0);
    expect(parseNumber('   ')).toBe(0);
  });

  it('nu transforma in zero un text care nu e numar', () => {
    // Zero e o valoare pe care o receptie chiar o poate avea; daca "abc" ar
    // da 0, s-ar salva ca si cum utilizatorul l-ar fi scris el.
    expect(parseNumber('abc')).toBeUndefined();
    expect(parseNumber('1.234.50')).toBeUndefined();
    expect(parseNumber('-')).toBeUndefined();
  });

  it('trims surrounding whitespace', () => {
    expect(parseNumber(' 15 ')).toBe(15);
  });
});

describe('formatNumber', () => {
  it('scrie zecimalele cu virgula si miile cu punct', () => {
    expect(formatNumber(328.5)).toBe('328,50');
    expect(formatNumber(0)).toBe('0,00');
    expect(formatNumber(1234.5)).toBe('1.234,50');
    expect(formatNumber(-1234.5)).toBe('-1.234,50');
  });

  it('honours an explicit decimal count', () => {
    expect(formatNumber(21.9, 3)).toBe('21,900');
    expect(formatNumber(1234.5, 0)).toBe('1.235');
  });
});

describe('parseNumber si formatNumber impreuna', () => {
  it('citesc inapoi exact ce scriu in celule', () => {
    for (const valoare of [0, 1, 20, 162.2, 1234.5, 1234567.89, -1234.5, 0.05]) {
      expect(parseNumber(formatNumber(valoare))).toBe(valoare);
    }
  });

  it('nu deriveaza dupa mai multe treceri', () => {
    // O valoare tastata, formatata, recitita si reformatata trebuie sa arate
    // la fel: altfel un pret ar aluneca la fiecare redesenare a tabelului.
    const tastat = '1.234,50';
    const odata = formatNumber(parseNumber(tastat)!);
    const inca = formatNumber(parseNumber(odata)!);
    expect(odata).toBe(tastat);
    expect(inca).toBe(tastat);
  });

  it('scriu la fel ca formatLei, ca sa nu existe doua notatii pe acelasi rand', () => {
    expect(formatNumber(1234.5)).toBe(formatLei(1234.5));
  });
});

describe('formatDateRO', () => {
  it('renders an ISO date as dd/mm/yyyy', () => {
    expect(formatDateRO('2026-09-03')).toBe('03/09/2026');
  });

  it('passes through anything that is not an ISO date', () => {
    expect(formatDateRO('')).toBe('');
    expect(formatDateRO('nope')).toBe('nope');
  });
});

describe('parseDateRO', () => {
  it('reads a dd/mm/yyyy date back as ISO', () => {
    expect(parseDateRO('03/09/2026')).toBe('2026-09-03');
  });

  it('accepts a dot or a dash separator and missing leading zeros', () => {
    expect(parseDateRO('03.09.2026')).toBe('2026-09-03');
    expect(parseDateRO('03-09-2026')).toBe('2026-09-03');
    expect(parseDateRO('3/9/2026')).toBe('2026-09-03');
    expect(parseDateRO('  3/9/2026  ')).toBe('2026-09-03');
  });

  it('accepts 29 February in a leap year', () => {
    expect(parseDateRO('29/02/2024')).toBe('2024-02-29');
  });

  it('rejects dates that do not exist', () => {
    expect(parseDateRO('31/02/2026')).toBeUndefined();
    expect(parseDateRO('29/02/2026')).toBeUndefined();
    expect(parseDateRO('01/13/2026')).toBeUndefined();
    expect(parseDateRO('00/09/2026')).toBeUndefined();
  });

  it('rejects blank and half-typed input', () => {
    expect(parseDateRO('')).toBeUndefined();
    expect(parseDateRO('03/09')).toBeUndefined();
    expect(parseDateRO('2026-09-03')).toBeUndefined();
    expect(parseDateRO('nope')).toBeUndefined();
  });

  it('round-trips with formatDateRO', () => {
    expect(formatDateRO(parseDateRO('03/09/2026')!)).toBe('03/09/2026');
  });
});

describe('formatLei', () => {
  it('scrie doua zecimale cu virgula', () => {
    expect(formatLei(78)).toBe('78,00');
    expect(formatLei(1234.5)).toBe('1.234,50');
    expect(formatLei(-11)).toBe('-11,00');
  });

  it('scrie zero ca zero, nu ca gol', () => {
    expect(formatLei(0)).toBe('0,00');
  });
});

describe('formatProcent', () => {
  it('adauga semnul procentului', () => {
    expect(formatProcent(35.14)).toBe('35,14 %');
  });

  it('scrie liniuta cand procentul nu se poate calcula', () => {
    expect(formatProcent(undefined)).toBe('—');
  });
});
