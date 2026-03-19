import { beforeEach, describe, expect, it } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import type { ReactNode } from 'react';
import { DateTimePrefsProvider } from '../DateTimePrefsContext';
import { useDateFormatter, useDateTimePrefs } from '../useDateTimePrefs';

const FIXED_ISO = '2025-07-04T14:30:00.000Z';

function wrapper({ children }: { children: ReactNode }) {
  return <DateTimePrefsProvider>{children}</DateTimePrefsProvider>;
}

describe('DateTimePrefsContext', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it('useDateFormatter returns a function that formats strings', () => {
    const { result } = renderHook(() => useDateFormatter(), { wrapper });
    const formatted = result.current(FIXED_ISO);
    expect(typeof formatted).toBe('string');
    expect(formatted.length).toBeGreaterThan(0);
  });

  it('changing prefs via setPrefs updates the formatter returned by the hook', () => {
    const { result } = renderHook(() => useDateTimePrefs(), { wrapper });

    const formattedBefore = result.current.formatDateTime(FIXED_ISO);

    act(() => {
      result.current.setPrefs({ dateOrder: 'DMY', timeFormat: '24h' });
    });

    const formattedAfter = result.current.formatDateTime(FIXED_ISO);
    // DMY 24h format should not include AM/PM and output differs from MDY 12h
    expect(formattedAfter).not.toMatch(/AM|PM/i);
    expect(formattedAfter).not.toBe(formattedBefore);
  });

  it('preferences have a valid shape when localStorage is empty', () => {
    const { result } = renderHook(() => useDateTimePrefs(), { wrapper });
    expect(['MDY', 'DMY']).toContain(result.current.prefs.dateOrder);
    expect(['12h', '24h']).toContain(result.current.prefs.timeFormat);
  });

  it('useDateTimePrefs throws outside DateTimePrefsProvider', () => {
    // renderHook without wrapper — hook must throw
    expect(() => {
      renderHook(() => useDateTimePrefs());
    }).toThrow('useDateTimePrefs must be used within DateTimePrefsProvider');
  });
});
