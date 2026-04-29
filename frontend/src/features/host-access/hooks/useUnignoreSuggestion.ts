import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  unignoreSuggestionMutation,
  listHostSuggestionsQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";
import type { HostSuggestionsPage } from "@/lib/api";

export function useUnignoreSuggestion() {
  const queryClient = useQueryClient();
  return useMutation({
    ...unignoreSuggestionMutation(),
    onSuccess: (_data, variables) => {
      // Patch the cache in place rather than invalidating: the suggestions query is
      // expensive and should only run on page load or after a hosts/groups save.
      // The unignored fqdn isn't added back to `suggestions` here — it will reappear
      // (if still applicable) on the next natural refetch.
      queryClient.setQueryData<HostSuggestionsPage>(
        listHostSuggestionsQueryKey(),
        (old) => {
          if (!old) return old;
          return {
            ...old,
            ignored: old.ignored.filter((s) => s.fqdn !== variables.path.fqdn),
          };
        },
      );
    },
  });
}
