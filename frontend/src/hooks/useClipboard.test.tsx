import { afterEach, describe, expect, it, vi } from "vitest";
import { notifications } from "@mantine/notifications";
import { screen } from "@testing-library/react";
import { useClipboard } from "./useClipboard";
import { renderWithProviders, setupUser } from "@/test/utils";

function CopyButton({
    text = "copy me",
    successMessage,
    errorMessage,
}: {
    text?: string;
    successMessage?: string;
    errorMessage?: string;
}) {
    const { copy } = useClipboard();
    return (
        <button onClick={() => copy(text, { successMessage, errorMessage })}>
            Copy
        </button>
    );
}

function setClipboard(writeText: ((text: string) => Promise<void>) | undefined) {
    Object.defineProperty(navigator, "clipboard", {
        value: writeText ? { writeText } : undefined,
        configurable: true,
    });
}

describe("useClipboard", () => {
    afterEach(() => {
        vi.restoreAllMocks();
    });

    it("writes text and shows a success notification", async () => {
        const user = setupUser();
        const writeText = vi.fn().mockResolvedValue(undefined);
        const show = vi.spyOn(notifications, "show");
        setClipboard(writeText);

        renderWithProviders(<CopyButton successMessage="Copied API key" />);

        await user.click(screen.getByRole("button", { name: /copy/i }));

        expect(writeText).toHaveBeenCalledWith("copy me");
        expect(show).toHaveBeenCalledWith({ message: "Copied API key", color: "green" });
    });

    it("shows a failure notification when clipboard write fails", async () => {
        const user = setupUser();
        const show = vi.spyOn(notifications, "show");
        setClipboard(vi.fn().mockRejectedValue(new Error("denied")));

        renderWithProviders(<CopyButton errorMessage="Could not copy token" />);

        await user.click(screen.getByRole("button", { name: /copy/i }));

        expect(show).toHaveBeenCalledWith({ message: "Could not copy token", color: "red" });
    });

    it("shows an unsupported-browser notification when the clipboard API is unavailable", async () => {
        const user = setupUser();
        const show = vi.spyOn(notifications, "show");
        setClipboard(undefined);

        renderWithProviders(<CopyButton />);

        await user.click(screen.getByRole("button", { name: /copy/i }));

        expect(show).toHaveBeenCalledWith({
            message: "Copy to clipboard is not supported in this browser.",
            color: "red",
        });
    });
});
