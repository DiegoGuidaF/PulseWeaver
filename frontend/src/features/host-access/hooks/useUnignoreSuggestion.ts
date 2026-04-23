import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  unignoreSuggestionMutation,
  listHostSuggestionsQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";

export function useUnignoreSuggestion() {
  const queryClient = useQueryClient();
  return useMutation({
    ...unignoreSuggestionMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: listHostSuggestionsQueryKey() });
    },
  });
}
