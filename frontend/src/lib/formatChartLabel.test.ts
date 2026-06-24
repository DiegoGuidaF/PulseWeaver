import { describe, expect, it } from "vitest";
import { formatChartLabel, presetToMs } from "./formatChartLabel";
import { PRESET_MS } from "@/lib/timePresets";

describe("formatChartLabel", () => {
    it("formats intraday ranges as time", () => {
        expect(formatChartLabel("2024-01-02T03:04:00", PRESET_MS.last_24h)).toBe("03:04");
    });

    it("formats week-or-shorter ranges with month, day, and time", () => {
        expect(formatChartLabel("2024-01-02T03:04:00", PRESET_MS.last_1w)).toBe("Jan 2 03:04");
    });

    it("formats longer ranges with month and day only", () => {
        expect(formatChartLabel("2024-01-02T03:04:00", PRESET_MS.last_1mo)).toBe("Jan 2");
    });

    it("resolves known presets and falls back to one day", () => {
        expect(presetToMs("last_1w")).toBe(PRESET_MS.last_1w);
        expect(presetToMs("not-a-preset")).toBe(PRESET_MS.last_24h);
    });
});
