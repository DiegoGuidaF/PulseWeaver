import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  getDateTimePrefs,
  setDateTimePrefs,
  DEFAULT_DATETIME_PREFS,
  DATETIME_PREFS_KEY,
  DATETIME_PREFS_EVENT,
} from '../userPreferences';

describe('userPreferences', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  describe('getDateTimePrefs', () => {
    it('returns browser-detected defaults when localStorage is empty', () => {
      const result = getDateTimePrefs();
      // Browser detection may differ per environment; just verify shape
      expect(['MDY', 'DMY']).toContain(result.dateOrder);
      expect(['12h', '24h']).toContain(result.timeFormat);
    });

    it('returns stored value when set', () => {
      const prefs = { dateOrder: 'DMY' as const, timeFormat: '24h' as const };
      localStorage.setItem(DATETIME_PREFS_KEY, JSON.stringify(prefs));
      expect(getDateTimePrefs()).toEqual(prefs);
    });

    it('returns browser defaults on malformed JSON', () => {
      localStorage.setItem(DATETIME_PREFS_KEY, 'not-json{');
      const result = getDateTimePrefs();
      expect(['MDY', 'DMY']).toContain(result.dateOrder);
      expect(['12h', '24h']).toContain(result.timeFormat);
    });

    it('returns browser defaults when stored shape is invalid', () => {
      localStorage.setItem(DATETIME_PREFS_KEY, JSON.stringify({ dateOrder: 'BOGUS', timeFormat: '99h' }));
      const result = getDateTimePrefs();
      expect(['MDY', 'DMY']).toContain(result.dateOrder);
      expect(['12h', '24h']).toContain(result.timeFormat);
    });
  });

  describe('setDateTimePrefs', () => {
    it('persists to localStorage', () => {
      const prefs = { dateOrder: 'DMY' as const, timeFormat: '24h' as const };
      setDateTimePrefs(prefs);
      expect(getDateTimePrefs()).toEqual(prefs);
    });

    it(`dispatches ${DATETIME_PREFS_EVENT}`, () => {
      const handler = vi.fn();
      window.addEventListener(DATETIME_PREFS_EVENT, handler);
      setDateTimePrefs(DEFAULT_DATETIME_PREFS);
      window.removeEventListener(DATETIME_PREFS_EVENT, handler);
      expect(handler).toHaveBeenCalledOnce();
    });
  });
});
