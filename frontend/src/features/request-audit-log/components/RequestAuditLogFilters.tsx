import { useEffect } from "react";
import { useSearchParams } from "react-router-dom";
import { Group, Select, SegmentedControl, TextInput } from "@mantine/core";
import { DatePickerInput } from "@mantine/dates";
import { IconSearch } from "@tabler/icons-react";
import { useDevices } from "@/features/devices/hooks/useDevices";
import { useRequestAuditLogDenyReasons } from "../hooks/useRequestAuditLogDenyReasons";
import { DENY_REASON_LABELS } from "../constants";

export function RequestAuditLogFilters() {
    const [searchParams, setSearchParams] = useSearchParams();
    const { data: devices } = useDevices();
    const { data: denyReasons } = useRequestAuditLogDenyReasons();

    // Set default date range on first load if not in URL
    useEffect(() => {
        const hasFrom = searchParams.has("from");
        const hasTo = searchParams.has("to");
        if (!hasFrom || !hasTo) {
            const now = new Date();
            const dayAgo = new Date(now.getTime() - 24 * 60 * 60 * 1000);
            setSearchParams(
                (prev) => {
                    if (!hasFrom) prev.set("from", dayAgo.toISOString());
                    if (!hasTo) prev.set("to", now.toISOString());
                    return prev;
                },
                { replace: true },
            );
        }
    }, []); // eslint-disable-line react-hooks/exhaustive-deps

    const deviceId = searchParams.get("device_id") ?? null;
    const outcome = searchParams.get("outcome") ?? "all";
    const ip = searchParams.get("ip") ?? "";
    const denyReason = searchParams.get("deny_reason") ?? null;
    const fromStr = searchParams.get("from");
    const toStr = searchParams.get("to");
    const fromDate = fromStr ? new Date(fromStr) : null;
    const toDate = toStr ? new Date(toStr) : null;

    const deviceOptions = (devices ?? []).map((d) => ({
        value: String(d.id),
        label: d.name,
    }));

    const denyReasonOptions = (denyReasons ?? []).map((r) => ({
        value: r,
        label: DENY_REASON_LABELS[r] ?? r,
    }));

    function setParam(key: string, value: string | null) {
        setSearchParams((prev) => {
            if (value === null || value === "") {
                prev.delete(key);
            } else {
                prev.set(key, value);
            }
            return prev;
        });
    }

    function handleDateRange([from, to]: [Date | null, Date | null]) {
        setSearchParams((prev) => {
            if (from) {
                prev.set("from", from.toISOString());
            } else {
                prev.delete("from");
            }
            if (to) {
                prev.set("to", to.toISOString());
            } else {
                prev.delete("to");
            }
            return prev;
        });
    }

    const showDenyReason = outcome !== "allow";

    return (
        <Group wrap="wrap" gap="sm" align="flex-end">
            <Select
                placeholder="All devices"
                data={deviceOptions}
                value={deviceId}
                onChange={(val) => setParam("device_id", val)}
                clearable
                w={200}
            />
            <SegmentedControl
                data={[
                    { label: "All", value: "all" },
                    { label: "Allow", value: "allow" },
                    { label: "Deny", value: "deny" },
                ]}
                value={outcome}
                onChange={(val) => {
                    if (val === "allow") {
                        // Clear deny_reason when switching to allow — it's irrelevant
                        setSearchParams((prev) => {
                            prev.set("outcome", val);
                            prev.delete("deny_reason");
                            return prev;
                        });
                    } else if (val === "all") {
                        setParam("outcome", null);
                    } else {
                        setParam("outcome", val);
                    }
                }}
            />
            {showDenyReason && (
                <Select
                    label="Deny reason"
                    placeholder="Any"
                    data={denyReasonOptions}
                    value={denyReason}
                    onChange={(val) => setParam("deny_reason", val)}
                    clearable
                    w={200}
                />
            )}
            <DatePickerInput
                type="range"
                placeholder="Date range"
                value={[fromDate, toDate]}
                onChange={handleDateRange}
                clearable
                w={240}
            />
            <TextInput
                placeholder="Filter by IP"
                leftSection={<IconSearch size={16} />}
                value={ip}
                onChange={(e) => setParam("ip", e.currentTarget.value)}
                w={200}
            />
        </Group>
    );
}
