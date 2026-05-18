import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
    updateNetworkPolicyMutation,
    getNetworkPolicyQueryKey,
    listNetworkPoliciesQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";
import type { Options, UpdateNetworkPolicyData } from "@/lib/api";

export function useUpdateNetworkPolicy() {
    const queryClient = useQueryClient();

    return useMutation({
        ...updateNetworkPolicyMutation(),
        onSuccess: (_data, variables: Options<UpdateNetworkPolicyData>) => {
            // PATCH returns 204 — invalidate to refetch detail and list
            queryClient.invalidateQueries({
                queryKey: getNetworkPolicyQueryKey({ path: { id: variables.path!.id } }),
            });
            queryClient.invalidateQueries({ queryKey: listNetworkPoliciesQueryKey() });
        },
    });
}
