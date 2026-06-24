import { beforeEach, describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import { AppErrorBoundary } from "./ErrorBoundary";
import { renderWithProviders, setupUser } from "@/test/utils";

function Boom({ shouldThrow }: { shouldThrow: boolean }) {
    if (shouldThrow) {
        throw new Error("render exploded");
    }
    return <div>Recovered content</div>;
}

function ToggleBoom() {
    if (throwOnRender) {
        throw new Error("render exploded");
    }
    return <div>Recovered content</div>;
}

let throwOnRender = false;

describe("AppErrorBoundary", () => {
    beforeEach(() => {
        vi.spyOn(console, "error").mockImplementation(() => {});
        throwOnRender = false;
    });

    it("renders fallback content for render errors", () => {
        renderWithProviders(
            <AppErrorBoundary>
                <Boom shouldThrow />
            </AppErrorBoundary>,
        );

        expect(screen.getByText("Something went wrong")).toBeInTheDocument();
        expect(screen.getByText("render exploded")).toBeInTheDocument();
        expect(screen.getByRole("button", { name: /try again/i })).toBeInTheDocument();
    });

    it("resets through the fallback action when children stop throwing", async () => {
        const user = setupUser();
        throwOnRender = true;
        renderWithProviders(
            <AppErrorBoundary>
                <ToggleBoom />
            </AppErrorBoundary>,
        );

        expect(screen.getByText("Something went wrong")).toBeInTheDocument();
        throwOnRender = false;
        await user.click(screen.getByRole("button", { name: /try again/i }));

        expect(screen.getByText("Recovered content")).toBeInTheDocument();
    });
});
