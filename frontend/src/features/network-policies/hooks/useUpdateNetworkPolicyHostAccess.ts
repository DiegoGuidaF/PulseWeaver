import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
    updateNetworkPolicyHostAccessMutation,
    getNetworkPolicyQueryKey,
    listNetworkPoliciesQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";
import type { Options, UpdateNetworkPolicyHostAccessData } from "@/lib/api";

export function useUpdateNetworkPolicyHostAccess() {
    const queryClient = useQueryClient();

    return useMutation({
        ...updateNetworkPolicyHostAccessMutation(),
        onSuccess: (_data, variables: Options<UpdateNetworkPolicyHostAccessData>) => {
            // PUT returns 204 — invalidate to refetch updated host counts and assignment state
            queryClient.invalidateQueries({
                queryKey: getNetworkPolicyQueryKey({ path: { id: variables.path!.id } }),
            });
            queryClient.invalidateQueries({ queryKey: listNetworkPoliciesQueryKey() });
        },
    });
}
