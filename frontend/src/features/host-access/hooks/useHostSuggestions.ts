import { useQuery } from "@tanstack/react-query";
import { listHostSuggestionsOptions } from "@/lib/api/@tanstack/react-query.gen";

export function useHostSuggestions() {
  return useQuery(listHostSuggestionsOptions());
}
