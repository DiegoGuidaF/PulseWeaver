/**
 * Date display utilities.
 *
 * formatDateTime: medium date + short time, respecting the user's system locale
 * (12h or 24h is determined automatically by the browser locale — no configuration needed).
 */
const dateTimeFormatter = new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
});

export function formatDateTime(d: Date): string {
    return dateTimeFormatter.format(d);
}

export function isPast(d: Date): boolean {
    return d < new Date();
}
