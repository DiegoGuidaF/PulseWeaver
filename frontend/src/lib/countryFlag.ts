/**
 * Converts an ISO 3166-1 alpha-2 country code to its flag emoji.
 * Uses Unicode regional indicator symbols (U+1F1E6–U+1F1FF).
 * Returns an empty string for invalid or empty input.
 */
export function countryFlagEmoji(code: string): string {
    if (!code || code.length !== 2) return "";
    const upper = code.toUpperCase();
    return [...upper]
        .map((c) => String.fromCodePoint(0x1f1e6 + c.charCodeAt(0) - 65))
        .join("");
}
