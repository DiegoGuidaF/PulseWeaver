import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  createKnownHostsMutation,
  listKnownHostsQueryKey,
  listHostSuggestionsQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";
import type { ListHostSuggestionsResponse, Options, CreateKnownHostsData } from "@/lib/api";

export function useCreateKnownHosts() {
  const queryClient = useQueryClient();
  return useMutation({
    ...createKnownHostsMutation(),
    onSuccess: (_data, variables: Options<CreateKnownHostsData>) => {
      const promoted = new Set(variables.body?.fqdns ?? []);

      // Immediately remove promoted FQDNs from the cached suggestions list so
      // the row disappears without waiting for the slow listHostSuggestions refetch.
      queryClient.setQueryData<ListHostSuggestionsResponse>(
        listHostSuggestionsQueryKey(),
        (old) =>
          old
            ? { ...old, suggestions: old.suggestions.filter((s) => !promoted.has(s.fqdn)) }
            : old,
      );

      queryClient.invalidateQueries({ queryKey: listKnownHostsQueryKey() });
      queryClient.invalidateQueries({ queryKey: listHostSuggestionsQueryKey() });
    },
  });
}
