import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  ignoreSuggestionMutation,
  listHostSuggestionsQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";

export function useIgnoreSuggestion() {
  const queryClient = useQueryClient();
  return useMutation({
    ...ignoreSuggestionMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: listHostSuggestionsQueryKey() });
    },
  });
}
