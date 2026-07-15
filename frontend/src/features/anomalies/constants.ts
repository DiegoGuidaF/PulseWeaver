import type { ComponentType } from "react";
import {
    IconChartDots,
    IconClockPause,
    IconFingerprint,
    IconKeyOff,
    IconMapPin,
    IconPlaneTilt,
    IconRadar2,
    IconRepeat,
    IconTrendingUp,
    IconWorldOff,
} from "@tabler/icons-react";
import { AnomalyKind, AnomalySeverity, AnomalyStatus } from "@/lib/api";
import type { ListAnomaliesData } from "@/lib/api";

export type AnomalyKindFamily = "rules" | "volume" | "novelty";

interface AnomalyKindMeta {
    label: string;
    icon: ComponentType<{ size?: number; stroke?: number }>;
    family: AnomalyKindFamily;
}

export const ANOMALY_KIND_META: Record<AnomalyKind, AnomalyKindMeta> = {
    [AnomalyKind.EXPIRED_ACCESS]: { label: "Expired access", icon: IconClockPause, family: "rules" },
    [AnomalyKind.INVALID_TOKEN]: { label: "Invalid token", icon: IconKeyOff, family: "rules" },
    [AnomalyKind.DENY_SPIKE]: { label: "Deny spike", icon: IconTrendingUp, family: "volume" },
    [AnomalyKind.ENTITY_DRIFT]: { label: "Entity drift", icon: IconChartDots, family: "volume" },
    [AnomalyKind.GEO_DENIED]: { label: "Unexpected geography", icon: IconWorldOff, family: "volume" },
    [AnomalyKind.HOST_PROBING]: { label: "Host probing", icon: IconRadar2, family: "volume" },
    [AnomalyKind.ADDRESS_CHURN]: { label: "Address churn", icon: IconRepeat, family: "volume" },
    [AnomalyKind.NEW_USER_AGENT]: { label: "New fingerprint", icon: IconFingerprint, family: "novelty" },
    [AnomalyKind.NEW_COUNTRY]: { label: "New country", icon: IconMapPin, family: "novelty" },
    [AnomalyKind.IMPOSSIBLE_TRAVEL]: { label: "Impossible travel", icon: IconPlaneTilt, family: "novelty" },
};

export const ANOMALY_KIND_FAMILY_LABELS: Record<AnomalyKindFamily, string> = {
    rules: "Rules",
    volume: "Volume",
    novelty: "Novelty",
};

export const ANOMALY_KIND_OPTIONS = Object.values(AnomalyKind).map((kind) => ({
    value: kind,
    label: ANOMALY_KIND_META[kind].label,
}));

/** Kind options for the page's multi-select filter, grouped by detector family (Mantine's `{ group, items }[]` shape). */
export const ANOMALY_KIND_GROUPED_OPTIONS = (["rules", "volume", "novelty"] as const satisfies readonly AnomalyKindFamily[]).map(
    (family) => ({
        group: ANOMALY_KIND_FAMILY_LABELS[family],
        items: Object.values(AnomalyKind)
            .filter((kind) => ANOMALY_KIND_META[kind].family === family)
            .map((kind) => ({ value: kind, label: ANOMALY_KIND_META[kind].label })),
    }),
);

interface AnomalySeverityMeta {
    color: string;
    label: string;
}

// Severity follows the UI style guide's Attention-band registers (critical =
// red, warning = the "noted" yellow) rather than the brand's amber — amber is
// reserved for liveness/heartbeat signals and would collide with severity if
// reused here.
export const ANOMALY_SEVERITY_META: Record<AnomalySeverity, AnomalySeverityMeta> = {
    [AnomalySeverity.CRITICAL]: { color: "red", label: "Critical" },
    [AnomalySeverity.WARNING]: { color: "yellow", label: "Warning" },
    [AnomalySeverity.INFO]: { color: "gray", label: "Info" },
};

export const ANOMALY_SEVERITY_OPTIONS = Object.values(AnomalySeverity).map((severity) => ({
    value: severity,
    label: ANOMALY_SEVERITY_META[severity].label,
}));

/** Client-only: extends the server status enum with an "all" option for the filter UI. */
export type AnomalyStatusFilter = AnomalyStatus | "all";

export const ANOMALY_STATUS_FILTER_OPTIONS: { value: AnomalyStatusFilter; label: string }[] = [
    { value: AnomalyStatus.OPEN, label: "Open" },
    { value: AnomalyStatus.ACKNOWLEDGED, label: "Acknowledged" },
    { value: "all", label: "All" },
];

/** Severities shown on the dashboard section — `info` is page-only by default. */
export const DASHBOARD_ANOMALY_SEVERITIES: ReadonlySet<AnomalySeverity> = new Set([
    AnomalySeverity.WARNING,
    AnomalySeverity.CRITICAL,
]);

/** Rows shown in the dashboard "Unusual activity" section before it links out to the full page. */
export const DASHBOARD_ANOMALY_ROW_LIMIT = 5;

/**
 * The exact query params shared by the nav badge, the dashboard section, and
 * the Anomalies page's default view. One constant object (rather than three
 * call sites re-typing `{ status: "open", limit: 500 }`) is what keeps their
 * TanStack cache key identical, so all three read the same cache entry.
 */
export const OPEN_ANOMALIES_QUERY: NonNullable<ListAnomaliesData["query"]> = {
    status: AnomalyStatus.OPEN,
    limit: 500,
};

/** Result set is retention-bounded, so a single page fetch never needs pagination. */
export const ANOMALIES_PAGE_LIMIT = 500;
