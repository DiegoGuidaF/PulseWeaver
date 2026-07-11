import { useAnomalies } from "./useAnomalies";
import { OPEN_ANOMALIES_QUERY } from "../constants";

/**
 * The shared open-anomalies query used by the nav badge, the dashboard
 * section, and the Anomalies page's default view. All three call this same
 * hook so they read one TanStack cache entry instead of three near-identical
 * queries; each surface narrows the result client-side as needed (e.g. the
 * dashboard drops `info` severity, the badge just counts).
 */
export function useOpenAnomalies() {
    return useAnomalies(OPEN_ANOMALIES_QUERY);
}
