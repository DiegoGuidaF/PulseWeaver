import { useQuery } from "@tanstack/react-query";
import { listNetworkPoliciesOptions } from "@/lib/api/@tanstack/react-query.gen";

export function useNetworkPolicies() {
    return useQuery(listNetworkPoliciesOptions());
}
