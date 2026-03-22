export const DATEPICKER_TIME_PRESETS = [
    { key: "last_5m",   label: "5m",   ms: 5 * 60 * 1000 },
    { key: "last_15m",  label: "15m",  ms: 15 * 60 * 1000 },
    { key: "last_30m",  label: "30m",  ms: 30 * 60 * 1000 },
    { key: "last_1h",   label: "1h",   ms: 60 * 60 * 1000 },
    { key: "last_6h",   label: "6h",   ms: 6 * 60 * 60 * 1000 },
    { key: "last_24h",  label: "24h",  ms: 24 * 60 * 60 * 1000 },
    { key: "last_3d",   label: "3d",   ms: 3 * 24 * 60 * 60 * 1000 },
    { key: "last_1w",   label: "1w",   ms: 7 * 24 * 60 * 60 * 1000 },
    { key: "last_1mo",  label: "1mo",  ms: 30 * 24 * 60 * 60 * 1000 },
] as const;

export type PresetKey = (typeof DATEPICKER_TIME_PRESETS)[number]["key"];

export const PRESET_MS: Record<string, number> = Object.fromEntries(
    DATEPICKER_TIME_PRESETS.map(({ key, ms }) => [key, ms]),
);
