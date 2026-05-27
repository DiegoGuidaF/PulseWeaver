import { useEffect, useRef } from "react";

/**
 * Adds aria-label to the filter trigger buttons rendered by mantine-datatable.
 * The library renders an unlabelled icon button per filterable column and
 * provides no API to customise its attributes, so the label is set imperatively
 * after mount.
 *
 * Returns a ref that must be attached to the div wrapping <DataTable>.
 */
export function useFilterButtonLabels(
  labelMap: Record<string, string>,
): React.RefObject<HTMLDivElement | null> {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    el.querySelectorAll<HTMLTableCellElement>("th[data-accessor]").forEach((th) => {
      const label = labelMap[th.getAttribute("data-accessor") ?? ""];
      if (!label) return;
      th.querySelectorAll<HTMLButtonElement>("button[aria-haspopup]").forEach((btn) => {
        if (!btn.getAttribute("aria-label")) btn.setAttribute("aria-label", label);
      });
    });
  // labelMap is a literal object constant at each call site — no need to re-run
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);
  return ref;
}
