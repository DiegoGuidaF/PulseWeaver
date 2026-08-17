import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";
import { renderWithProviders } from "@/test/utils";
import { FilterableCell } from "./FilterableCell";

describe("FilterableCell", () => {
    it("lets a value shrink by default", () => {
        renderWithProviders(
            <FilterableCell filterLabel="Filter by this value">
                <span>Disabled</span>
            </FilterableCell>,
        );

        expect(screen.getByText("Disabled").parentElement).not.toHaveStyle({ minWidth: "max-content" });
    });

    it("holds a value at its natural width when noShrink is set", () => {
        // The column measures from its cells, so without this a badge in a
        // minority of rows is sized by whatever the majority renders and clipped.
        renderWithProviders(
            <FilterableCell filterLabel="Filter by this value" noShrink>
                <span>Disabled</span>
            </FilterableCell>,
        );

        expect(screen.getByText("Disabled").parentElement).toHaveStyle({ minWidth: "max-content" });
    });
});
