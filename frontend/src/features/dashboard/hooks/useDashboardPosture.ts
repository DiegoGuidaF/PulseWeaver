import { useQuery } from "@tanstack/react-query";
import { getDashboardPostureOptions } from "@/lib/api/@tanstack/react-query.gen";

export function useDashboardPosture() {
    return useQuery({
        ...getDashboardPostureOptions(),
        staleTime: 60_000,
    });
}
