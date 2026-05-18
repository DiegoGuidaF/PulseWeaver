import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
    deleteNetworkPolicyMutation,
    getNetworkPolicyQueryKey,
    listNetworkPoliciesQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";
import type { Options, DeleteNetworkPolicyData } from "@/lib/api";

export function useDeleteNetworkPolicy(options?: { onSuccess?: () => void }) {
    const queryClient = useQueryClient();

    return useMutation({
        ...deleteNetworkPolicyMutation(),
        onSuccess: (_data, variables: Options<DeleteNetworkPolicyData>) => {
            queryClient.removeQueries({
                queryKey: getNetworkPolicyQueryKey({ path: { id: variables.path!.id } }),
            });
            queryClient.invalidateQueries({ queryKey: listNetworkPoliciesQueryKey() });
            options?.onSuccess?.();
        },
    });
}
