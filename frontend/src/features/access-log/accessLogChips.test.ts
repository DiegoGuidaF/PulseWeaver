import { describe, expect, it, vi } from "vitest";
import { SortOrder } from "@/lib/api";
import { buildAccessLogChips } from "./accessLogChips";
import type { ColumnFilterState, FilterColumnKey } from "./filterConfig";
import type { AccessLogFilters } from "./hooks/useAccessLogFilters";

/** An unfiltered `AccessLogFilters`, narrowed per test by the overrides. */
function makeFilters(overrides: Partial<AccessLogFilters> = {}): AccessLogFilters {
    const columns: Partial<Record<FilterColumnKey, ColumnFilterState>> = {};
    return {
        queryParams: {},
        filterKey: "",
        presetStr: null,
        fromStr: null,
        toStr: null,
        outcomeStr: null,
        sort: "created_at",
        order: SortOrder.DESC,
        hasCustomTo: false,
        hasActiveFilters: false,
        getColumnFilter: (key) => columns[key] ?? { op: "in", values: [] },
        setColumnFilter: vi.fn(),
        setPreset: vi.fn(),
        setOutcome: vi.fn(),
        setSort: vi.fn(),
        setSearchParams: vi.fn(),
        clearAll: vi.fn(),
        ...overrides,
    };
}

/** Stands in for the user's date preference; identity keeps assertions readable. */
const formatDateTime = (value: string) => value;

function build(overrides: Partial<AccessLogFilters>, valueLabels = {}) {
    return buildAccessLogChips({ filters: makeFilters(overrides), formatDateTime, valueLabels });
}

describe("buildAccessLogChips", () => {
    it("returns no chips when nothing is filtered", () => {
        expect(build({})).toEqual([]);
    });

    it("does not chip a preset — it is a view setting, not a filter", () => {
        expect(build({ presetStr: "last_24h" })).toEqual([]);
    });

    it.each([
        { name: "both bounds", fromStr: "A", toStr: "B", expected: "A → B" },
        { name: "open end", fromStr: "A", toStr: null, expected: "A → now" },
        { name: "open start", fromStr: null, toStr: "B", expected: "— → B" },
    ])("renders a Time chip with $name", ({ fromStr, toStr, expected }) => {
        const [chip] = build({ fromStr, toStr });
        expect(chip).toMatchObject({ label: "Time", value: expected });
    });

    it("clears both bounds when the Time chip is removed", () => {
        const setSearchParams = vi.fn();
        const [chip] = build({ fromStr: "A", toStr: "B", setSearchParams });

        chip.onRemove();

        const updater = setSearchParams.mock.calls[0][0] as (p: URLSearchParams) => URLSearchParams;
        const next = updater(new URLSearchParams({ from: "A", to: "B", preset: "last_24h" }));
        expect(next.has("from")).toBe(false);
        expect(next.has("to")).toBe(false);
        expect(next.get("preset")).toBe("last_24h");
    });

    it.each([
        ["allow", "Allow"],
        ["deny", "Deny"],
    ])("renders the %s outcome as %s", (outcomeStr, expected) => {
        expect(build({ outcomeStr })).toEqual([
            expect.objectContaining({ label: "Outcome", value: expected }),
        ]);
    });

    it("phrases a column filter with its operator and values", () => {
        const chips = build({
            getColumnFilter: (key) =>
                key === "client_ip" ? { op: "not_in", values: ["10.0.0.1", "10.0.0.2"] } : { op: "in", values: [] },
        });

        expect(chips).toEqual([
            expect.objectContaining({ label: "IP", value: "is none of 10.0.0.1, 10.0.0.2" }),
        ]);
    });

    it("phrases a null operator without values", () => {
        const chips = build({
            getColumnFilter: (key) =>
                key === "country_code" ? { op: "is_null", values: [] } : { op: "in", values: [] },
        });

        expect(chips).toEqual([expect.objectContaining({ label: "Country", value: "is unknown" })]);
    });

    it("resolves ids to names through valueLabels, falling back to the raw id", () => {
        const chips = build(
            {
                getColumnFilter: (key) =>
                    key === "device_id" ? { op: "in", values: ["1", "7"] } : { op: "in", values: [] },
            },
            { device_id: (v: string) => (v === "1" ? "Laptop" : v) },
        );

        expect(chips[0].value).toBe("is any of Laptop, 7");
    });

    it("orders chips as time, outcome, then columns in column order", () => {
        const chips = build({
            fromStr: "A",
            outcomeStr: "deny",
            getColumnFilter: (key) =>
                key === "client_ip" || key === "user_id"
                    ? { op: "in", values: ["x"] }
                    : { op: "in", values: [] },
        });

        expect(chips.map((c) => c.label)).toEqual(["Time", "Outcome", "IP", "User"]);
    });
});
