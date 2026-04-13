import { notifications } from "@mantine/notifications";

/**
 * Returns a `copy(text, options?)` function that writes to the clipboard and
 * shows a Mantine notification on success or failure.
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
    return { copy };
}
