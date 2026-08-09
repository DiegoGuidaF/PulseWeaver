import { notifications } from "@mantine/notifications";

/**
 * Returns clipboard writers that show a Mantine notification on success or
 * failure: `copy(text, options?)` for text, `copyImage(makePng, options?)` for
 * an image.
 */
export function useClipboard() {
    async function copy(
        text: string,
        {
            successMessage = "Copied to clipboard",
            errorMessage = "Failed to copy",
        }: { successMessage?: string; errorMessage?: string } = {},
    ) {
        if (!("clipboard" in navigator) || !navigator.clipboard?.writeText) {
            notifications.show({ message: "Copy to clipboard is not supported in this browser.", color: "red" });
            return;
        }
        try {
            await navigator.clipboard.writeText(text);
            notifications.show({ message: successMessage, color: "green" });
        } catch {
            notifications.show({ message: errorMessage, color: "red" });
        }
    }
    async function copyImage(
        makePng: () => Promise<Blob>,
        {
            successMessage = "Copied to clipboard",
            errorMessage = "Failed to copy image",
        }: { successMessage?: string; errorMessage?: string } = {},
    ) {
        if (typeof ClipboardItem === "undefined" || !navigator.clipboard?.write) {
            notifications.show({ message: "Copying images is not supported in this browser.", color: "red" });
            return;
        }
        try {
            // The blob is handed over as a promise rather than awaited first:
            // Safari only honours a write issued synchronously inside the user
            // gesture that triggered it.
            await navigator.clipboard.write([new ClipboardItem({ "image/png": makePng() })]);
            notifications.show({ message: successMessage, color: "green" });
        } catch {
            notifications.show({ message: errorMessage, color: "red" });
        }
    }

    return { copy, copyImage };
}
