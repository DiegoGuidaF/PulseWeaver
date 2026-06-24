import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import { CursorPagination } from "./CursorPagination";
import { renderWithProviders, setupUser } from "@/test/utils";

describe("CursorPagination", () => {
    it("renders result count and disables unavailable navigation", () => {
        renderWithProviders(
            <CursorPagination total={1} nextCursor={null} onCursorChange={vi.fn()} />,
        );

        expect(screen.getByText("1 result")).toBeInTheDocument();
        expect(screen.getByText("Page 1 of 1")).toBeInTheDocument();
        expect(screen.getByRole("button", { name: /previous page/i })).toBeDisabled();
        expect(screen.getByRole("button", { name: /next page/i })).toBeDisabled();
    });

    it("enables next navigation and calls back with the next cursor", async () => {
        const user = setupUser();
        const onCursorChange = vi.fn();
        renderWithProviders(
            <CursorPagination total={50} pageSize={25} nextCursor="cursor-2" onCursorChange={onCursorChange} />,
        );

        await user.click(screen.getByRole("button", { name: /next page/i }));

        expect(onCursorChange).toHaveBeenCalledWith("cursor-2");
        expect(screen.getByText("Page 2 of 2")).toBeInTheDocument();
        expect(screen.getByRole("button", { name: /previous page/i })).toBeEnabled();
    });

    it("calls back with the cached previous cursor", async () => {
        const user = setupUser();
        const onCursorChange = vi.fn();

        function Harness() {
            const [nextCursor, setNextCursor] = useState("cursor-2");
            return (
                <CursorPagination
                    total={75}
                    pageSize={25}
                    nextCursor={nextCursor}
                    onCursorChange={(cursor) => {
                        onCursorChange(cursor);
                        setNextCursor(cursor === "cursor-2" ? "cursor-3" : "cursor-2");
                    }}
                />
            );
        }

        renderWithProviders(<Harness />);

        await user.click(screen.getByRole("button", { name: /next page/i }));
        await user.click(screen.getByRole("button", { name: /next page/i }));
        await user.click(screen.getByRole("button", { name: /previous page/i }));

        expect(onCursorChange).toHaveBeenLastCalledWith("cursor-2");
        expect(screen.getByText("Page 2 of 3")).toBeInTheDocument();
    });
});
