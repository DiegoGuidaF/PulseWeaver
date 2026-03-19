import { describe, expect, it } from 'vitest';
import { formatDateTimeWith, isPast } from '../dates';

// A fixed RFC3339 timestamp: 2025-07-04T14:30:00Z = 14:30 UTC
const FIXED_ISO = '2025-07-04T14:30:00.000Z';

describe('formatDateTimeWith', () => {
  it('formats a known RFC3339 string for MDY 12h', () => {
    const result = formatDateTimeWith(FIXED_ISO, 'MDY', '12h');
    expect(result).toMatch(/AM|PM/i);
  });

  it('omits AM/PM when timeFormat is "24h"', () => {
    const result = formatDateTimeWith(FIXED_ISO, 'MDY', '24h');
    expect(result).not.toMatch(/AM|PM/i);
  });

  it('includes AM/PM when timeFormat is "12h" for DMY order', () => {
    const result = formatDateTimeWith(FIXED_ISO, 'DMY', '12h');
    expect(result).toMatch(/AM|PM/i);
  });

  it('uses day-first order for DMY', () => {
    // en-GB medium date style: "4 Jul 2025" — day comes before month
    const result = formatDateTimeWith('2025-07-04T00:00:00.000Z', 'DMY', '24h');
    const dayPos = result.indexOf('4');
    const julPos = result.indexOf('Jul');
    expect(dayPos).toBeLessThan(julPos);
  });

  it('uses month-first order for MDY', () => {
    // en-US medium date style: "Jul 4, 2025" — month comes before day
    const result = formatDateTimeWith('2025-07-04T00:00:00.000Z', 'MDY', '24h');
    const julPos = result.indexOf('Jul');
    const dayPos = result.indexOf('4');
    expect(julPos).toBeLessThan(dayPos);
  });
});

describe('isPast', () => {
  it('returns true for a past date string', () => {
    expect(isPast('2000-01-01T00:00:00.000Z')).toBe(true);
  });

  it('returns false for a future date string', () => {
    expect(isPast('2099-12-31T23:59:59.000Z')).toBe(false);
  });

  it('works with a Date object', () => {
    expect(isPast(new Date('2000-01-01T00:00:00.000Z'))).toBe(true);
    expect(isPast(new Date('2099-12-31T23:59:59.000Z'))).toBe(false);
  });
});
