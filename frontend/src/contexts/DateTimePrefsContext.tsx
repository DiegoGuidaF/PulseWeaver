import {
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react';
import {
  getDateTimePrefs,
  setDateTimePrefs,
  DATETIME_PREFS_EVENT,
  type DateTimePrefs,
} from '@/lib/userPreferences';
import { dateTimeFormat, formatDateTimeWith } from '@/lib/dates';
import { DateTimePrefsContext } from './useDateTimePrefs';

export function DateTimePrefsProvider({ children }: { children: ReactNode }) {
  const [prefs, setPrefsState] = useState<DateTimePrefs>(() => getDateTimePrefs());

  useEffect(() => {
    function onUpdate() {
      setPrefsState(getDateTimePrefs());
    }
    window.addEventListener(DATETIME_PREFS_EVENT, onUpdate);
    window.addEventListener('storage', onUpdate);
    return () => {
      window.removeEventListener(DATETIME_PREFS_EVENT, onUpdate);
      window.removeEventListener('storage', onUpdate);
    };
  }, []);

  function setPrefs(p: DateTimePrefs) {
    setDateTimePrefs(p);
    setPrefsState(p);
  }

  const formatDateTime = useMemo(
    () => (value: string) => formatDateTimeWith(value, prefs.dateOrder, prefs.timeFormat),
    [prefs.dateOrder, prefs.timeFormat],
  );

  const pickerValueFormat = useMemo(
    () => dateTimeFormat(prefs.dateOrder, prefs.timeFormat),
    [prefs.dateOrder, prefs.timeFormat],
  );

  return (
    <DateTimePrefsContext.Provider value={{ prefs, setPrefs, formatDateTime, pickerValueFormat }}>
      {children}
    </DateTimePrefsContext.Provider>
  );
}
