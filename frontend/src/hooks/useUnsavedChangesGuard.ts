import { useEffect } from "react";

/**
 * Guards against accidental data loss by prompting the user when they try to
 * close or refresh the browser tab while there are unsaved changes.
 *
 * For in-app navigation blocking (e.g. clicking nav links or switching tabs),
 * use controlled state at the component level instead — `useBlocker` requires
 * a data router (`createBrowserRouter`) which this app does not use yet.
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
}
