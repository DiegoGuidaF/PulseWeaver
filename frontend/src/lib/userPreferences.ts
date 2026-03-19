export const DATETIME_PREFS_KEY = 'pulseweaver_datetime_prefs';
export const DATETIME_PREFS_EVENT = 'pulseweaver:datetime-prefs-updated';

export type DateOrder = 'MDY' | 'DMY';
export type TimeFormat = '12h' | '24h';

export interface DateTimePrefs {
  dateOrder: DateOrder;
  timeFormat: TimeFormat;
}

export const DEFAULT_DATETIME_PREFS: DateTimePrefs = { dateOrder: 'MDY', timeFormat: '12h' };

function detectBrowserPrefs(): DateTimePrefs {
  try {
    const hour12 = new Intl.DateTimeFormat(undefined, { hour: 'numeric' }).resolvedOptions().hour12;
    const timeFormat: TimeFormat = hour12 ? '12h' : '24h';
    const parts = new Intl.DateTimeFormat().formatToParts(new Date(2000, 0, 15));
    const monthIdx = parts.findIndex((p) => p.type === 'month');
    const dayIdx = parts.findIndex((p) => p.type === 'day');
    const dateOrder: DateOrder =
      monthIdx !== -1 && dayIdx !== -1 && monthIdx < dayIdx ? 'MDY' : 'DMY';
    return { dateOrder, timeFormat };
  } catch {
    return DEFAULT_DATETIME_PREFS;
  }
}

export function getDateTimePrefs(): DateTimePrefs {
  try {
    const raw = localStorage.getItem(DATETIME_PREFS_KEY);
    if (!raw) return detectBrowserPrefs();
    const parsed = JSON.parse(raw) as { dateOrder?: unknown; timeFormat?: unknown };
    const dateOrder =
      parsed.dateOrder === 'MDY' || parsed.dateOrder === 'DMY' ? parsed.dateOrder : null;
    const timeFormat =
      parsed.timeFormat === '12h' || parsed.timeFormat === '24h' ? parsed.timeFormat : null;
    if (dateOrder && timeFormat) return { dateOrder, timeFormat };
    return detectBrowserPrefs();
  } catch {
    return detectBrowserPrefs();
  }
}

export function setDateTimePrefs(p: DateTimePrefs): void {
  localStorage.setItem(DATETIME_PREFS_KEY, JSON.stringify(p));
  window.dispatchEvent(new Event(DATETIME_PREFS_EVENT));
}
