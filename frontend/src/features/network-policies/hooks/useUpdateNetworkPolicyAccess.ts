import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  updateNetworkPolicyAccessMutation,
  getNetworkPolicyQueryKey,
  listNetworkPoliciesQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";
import type { Options, UpdateNetworkPolicyAccessData } from "@/lib/api";

export function useUpdateNetworkPolicyAccess() {
  const queryClient = useQueryClient();

  return useMutation({
    ...updateNetworkPolicyAccessMutation(),
    onSuccess: (_data, variables: Options<UpdateNetworkPolicyAccessData>) => {
      queryClient.invalidateQueries({
        queryKey: getNetworkPolicyQueryKey({ path: { policy_id: variables.path!.policy_id } }),
      });
      queryClient.invalidateQueries({ queryKey: listNetworkPoliciesQueryKey() });
    },
  });
}
