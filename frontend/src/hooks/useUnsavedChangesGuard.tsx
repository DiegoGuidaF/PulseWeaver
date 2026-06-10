import { useEffect } from "react";
import { useBlocker } from "react-router-dom";
import { Text } from "@mantine/core";
import { modals } from "@mantine/modals";

/**
 * Guards against accidental data loss while there are unsaved changes:
 * - prompts the browser's native dialog when closing or refreshing the tab;
 * - blocks in-app navigation (nav links, back button, programmatic `navigate`)
 *   and asks the user to confirm before the draft is discarded.
 *
 * In-app blocking relies on a data router (`createBrowserRouter`, see `App.tsx`),
 * which `useBlocker` requires.
 */
export function useUnsavedChangesGuard(isDirty: boolean): void {
  useEffect(() => {
    if (!isDirty) return;

    function handleBeforeUnload(e: BeforeUnloadEvent) {
      e.preventDefault();
    }

    window.addEventListener("beforeunload", handleBeforeUnload);
    return () => window.removeEventListener("beforeunload", handleBeforeUnload);
  }, [isDirty]);

  const blocker = useBlocker(
    ({ currentLocation, nextLocation }) =>
      isDirty && currentLocation.pathname !== nextLocation.pathname,
  );

  useEffect(() => {
    if (blocker.state !== "blocked") return;

    modals.openConfirmModal({
      title: "Discard unsaved changes?",
      children: (
        <Text size="sm">
          You have changes that haven't been saved. Leaving this page will discard them.
        </Text>
      ),
      labels: { confirm: "Discard changes", cancel: "Keep editing" },
      confirmProps: { color: "red" },
      // Force an explicit choice so the blocker never lingers in `blocked`.
      withCloseButton: false,
      closeOnClickOutside: false,
      closeOnEscape: false,
      onConfirm: () => blocker.proceed(),
      onCancel: () => blocker.reset(),
    });
  }, [blocker]);
}
