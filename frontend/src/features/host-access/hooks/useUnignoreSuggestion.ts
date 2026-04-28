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
    onMutate: async () => {
      await queryClient.cancelQueries({ queryKey: listHostSuggestionsQueryKey() });
    },
    onSuccess: (_data, variables) => {
      // Remove from ignored immediately; the background refetch will restore it
      // to suggestions if applicable.
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
      // Same rationale as useIgnoreSuggestion — don't start a slow refetch immediately.
    },
  });
}
