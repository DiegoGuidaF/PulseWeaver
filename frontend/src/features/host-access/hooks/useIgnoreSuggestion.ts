import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  ignoreSuggestionMutation,
  listHostSuggestionsQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";
import type { HostSuggestionsPage } from "@/lib/api";

export function useIgnoreSuggestion() {
  const queryClient = useQueryClient();
  return useMutation({
    ...ignoreSuggestionMutation(),
    onSuccess: (data, variables) => {
      // Patch the cache in place rather than invalidating: the suggestions query is
      // expensive and should only run on page load or after a hosts/groups save.
      queryClient.setQueryData<HostSuggestionsPage>(
        listHostSuggestionsQueryKey(),
        (old) => {
          if (!old) return old;
          return {
            suggestions: old.suggestions.filter((s) => s.fqdn !== variables.body.fqdn),
            ignored: [data, ...old.ignored],
          };
        },
      );
    },
  });
}
