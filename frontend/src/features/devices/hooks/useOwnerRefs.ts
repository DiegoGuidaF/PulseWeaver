import { useQuery } from "@tanstack/react-query";
import { listOwnerRefsOptions } from "@/lib/api/@tanstack/react-query.gen";

/**
 * Flat `{id, display_name}` owner list for pickers and the owner-jump select.
 *
 * The set of owners only moves when a user is created, deleted or renamed, and
 * those paths invalidate this key — so without a staleTime the refetch on every
 * navigation and every window focus would never return anything new.
 */
export function useOwnerRefs() {
  return useQuery({ ...listOwnerRefsOptions(), staleTime: 60_000 });
}
