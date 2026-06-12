import { describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import { renderWithProviders, setupUser } from "@/test/utils";
import { ErrorState } from "./ErrorState";

describe("ErrorState", () => {
    it("renders the default title and the message derived from an error", () => {
        renderWithProviders(<ErrorState error={new Error("boom")} />);

        expect(screen.getByText("Failed to load")).toBeInTheDocument();
        expect(screen.getByText("boom")).toBeInTheDocument();
    });

    it("uses an explicit message over the derived one and a custom title", () => {
        renderWithProviders(
            <ErrorState error={new Error("boom")} message="Could not fetch policies" title="Network error" />,
        );

        expect(screen.getByText("Network error")).toBeInTheDocument();
        expect(screen.getByText("Could not fetch policies")).toBeInTheDocument();
        expect(screen.queryByText("boom")).not.toBeInTheDocument();
    });

    it("renders a retry action and invokes onRetry when clicked", async () => {
        const user = setupUser();
        const onRetry = vi.fn();
        renderWithProviders(<ErrorState error={new Error("boom")} onRetry={onRetry} />);

        await user.click(screen.getByRole("button", { name: /try again/i }));
        expect(onRetry).toHaveBeenCalledOnce();
    });

    it("renders no retry button when onRetry is omitted", () => {
        renderWithProviders(<ErrorState error={new Error("boom")} />);

        expect(screen.queryByRole("button", { name: /try again/i })).not.toBeInTheDocument();
    });
});
