import { useState } from "react";
import { useSearchParams } from "react-router-dom";
import { Stack, Title, Text } from "@mantine/core";
import { RequestAuditLogFilters } from "@/features/request-audit-log/components/RequestAuditLogFilters";
import { RequestAuditLogTable } from "@/features/request-audit-log/components/RequestAuditLogTable";
import type { GetRequestAuditLogData } from "@/lib/api";

export function RequestAuditLogPage() {
    const [searchParams] = useSearchParams();
    const [refreshInterval, setRefreshInterval] = useState(0);

    const deviceIdStr = searchParams.get("device_id");
    const outcomeStr = searchParams.get("outcome");
    const ip = searchParams.get("ip") ?? undefined;
    const denyReason = searchParams.get("deny_reason") ?? undefined;
    const fromStr = searchParams.get("from");
    const toStr = searchParams.get("to");

    // The generated TypeScript type uses `Date` for datetime fields, but the
    // Zod request validator (z.iso.datetime) requires ISO string values at
    // runtime. Pass the raw URL strings directly and cast the type.
    const params: GetRequestAuditLogData["query"] = {
        device_id: deviceIdStr ? Number(deviceIdStr) : undefined,
        outcome:
            outcomeStr === "allow" ? true : outcomeStr === "deny" ? false : undefined,
        ip: ip || undefined,
        deny_reason: denyReason || undefined,
        from: (fromStr || undefined) as Date | undefined,
        to: (toStr || undefined) as Date | undefined,
    };

    return (
        <Stack maw={1200} gap="xl">
            <div>
                <Title order={1}>Access Log</Title>
                <Text c="dimmed">Policy decision history for all incoming requests.</Text>
            </div>
            <RequestAuditLogFilters refreshInterval={refreshInterval} onRefreshIntervalChange={setRefreshInterval} />
            <RequestAuditLogTable params={params} refreshInterval={refreshInterval} key={JSON.stringify(params)} />
        </Stack>
    );
}
