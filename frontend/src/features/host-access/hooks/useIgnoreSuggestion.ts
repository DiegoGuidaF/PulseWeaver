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
    onMutate: async () => {
      // Cancel any in-flight suggestions fetch so its connection slot is free for
      // the POST. The AbortSignal is wired through the generated SDK, so this
      // actually aborts the HTTP request — not just marks it cancelled in React Query.
      await queryClient.cancelQueries({ queryKey: listHostSuggestionsQueryKey() });
    },
    onSuccess: (data, variables) => {
      // Remove from suggestions and prepend to ignored immediately — don't wait for the
      // slow suggestions refetch to reflect the change.
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
      // Do not invalidate here — the setQueryData above is the source of truth.
      // Invalidating immediately would start a new slow GET that blocks subsequent
      // ignore POSTs on HTTP/1.1. Natural refetch triggers (window focus, tab
      // re-mount) handle eventual consistency.
    },
  });
}
