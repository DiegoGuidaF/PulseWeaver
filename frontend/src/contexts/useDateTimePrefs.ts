import { createContext, useContext } from 'react';
import type { DateTimePrefs } from '@/lib/userPreferences';

export interface DateTimePrefsContextValue {
  prefs: DateTimePrefs;
  setPrefs: (p: DateTimePrefs) => void;
  formatDateTime: (value: string) => string;
  pickerValueFormat: string;
}

export const DateTimePrefsContext = createContext<DateTimePrefsContextValue | null>(null);

export function useDateTimePrefs(): DateTimePrefsContextValue {
  const ctx = useContext(DateTimePrefsContext);
  if (!ctx) {
    throw new Error('useDateTimePrefs must be used within DateTimePrefsProvider');
  }
  return ctx;
}

export function useDateFormatter(): (value: string) => string {
  return useDateTimePrefs().formatDateTime;
}

export function usePickerValueFormat(): string {
  return useDateTimePrefs().pickerValueFormat;
}
