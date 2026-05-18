import { useMutation, useQueryClient } from "@tanstack/react-query";
import { createNetworkPolicyMutation, listNetworkPoliciesQueryKey } from "@/lib/api/@tanstack/react-query.gen";
import type { NetworkPolicy } from "@/lib/api";

export function useCreateNetworkPolicy(options?: {
    onSuccess?: (data: NetworkPolicy) => void;
}) {
    const queryClient = useQueryClient();

    return useMutation({
        ...createNetworkPolicyMutation(),
        onSuccess: (data) => {
            queryClient.invalidateQueries({ queryKey: listNetworkPoliciesQueryKey() });
            options?.onSuccess?.(data);
        },
    });
}
