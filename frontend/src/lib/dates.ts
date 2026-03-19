import dayjs from 'dayjs';
import type { DateOrder, TimeFormat } from './userPreferences';

export function dateTimeFormat(dateOrder: DateOrder, timeFormat: TimeFormat): string {
  const datePart = dateOrder === 'MDY' ? 'MMM DD, YYYY' : 'DD MMM YYYY';
  const timePart = timeFormat === '12h' ? 'hh:mm A' : 'HH:mm';
  return `${datePart} ${timePart}`;
}

export function formatDateTimeWith(value: string, dateOrder: DateOrder, timeFormat: TimeFormat): string {
  return dayjs(value).format(dateTimeFormat(dateOrder, timeFormat));
}

export function isPast(value: string | Date): boolean {
  return (typeof value === 'string' ? new Date(value) : value) < new Date();
}
