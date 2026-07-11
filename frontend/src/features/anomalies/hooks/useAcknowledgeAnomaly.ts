import { useMutation, useQueryClient } from "@tanstack/react-query";
import { acknowledgeAnomalyMutation } from "@/lib/api/@tanstack/react-query.gen";
import { toApiError } from "@/lib/api-client";

/**
 * Acknowledging is idempotent server-side, but a row can already be gone by
 * the time this request lands (acknowledged elsewhere, or reconciled by a
 * racing background refetch). A 404 in that case means the desired end state
 * already holds, so it's folded into the success path instead of surfacing
 * as a mutation error — components never show an error toast for it.
 */
export function useAcknowledgeAnomaly() {
    const queryClient = useQueryClient();
    const { mutationFn, ...mutationOptions } = acknowledgeAnomalyMutation();

    return useMutation({
        ...mutationOptions,
        mutationFn: async (variables, context) => {
            try {
                return await mutationFn!(variables, context);
            } catch (err) {
                if (toApiError(err).status === 404) return undefined;
                throw err;
            }
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: [{ _id: "listAnomalies" }] });
        },
    });
}
