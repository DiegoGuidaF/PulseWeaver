import { useQuery } from "@tanstack/react-query";
import { getNetworkPolicyOptions } from "@/lib/api/@tanstack/react-query.gen";

export function useNetworkPolicy(id: number) {
    return useQuery({
        ...getNetworkPolicyOptions({ path: { id } }),
        enabled: !isNaN(id),
    });
}
