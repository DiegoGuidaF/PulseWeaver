import { useMemo, useState } from "react";
import {
    Alert,
    Badge,
    Card,
    Checkbox,
    Grid,
    Group,
    Progress,
    SegmentedControl,
    Stack,
    Switch,
    Text,
    TextInput,
} from "@mantine/core";
import { IconAlertTriangle, IconSearch } from "@tabler/icons-react";
import type { NetworkPolicyDetail } from "@/lib/api";
import { StagedChangesBar } from "@/features/host-access/components/StagedChangesBar";
import type { NetworkPolicyHostAccessDraft, NetworkPolicyHostAccessAction } from "../drafts/networkPolicyHostAccessDraft";

const MAX_PREVIEW = 4;

type HostFilter = "all" | "granted" | "not-granted" | "via-group";

interface Props {
    detail: NetworkPolicyDetail;
    draft: NetworkPolicyHostAccessDraft;
    dispatch: React.Dispatch<NetworkPolicyHostAccessAction>;
    isDirty: boolean;
    isSaving: boolean;
    onSave: () => void;
    onDiscard: () => void;
}

export function NetworkPolicyHostAccessPanel({
    detail,
    draft,
    dispatch,
    isDirty,
    isSaving,
    onSave,
    onDiscard,
}: Props) {
    const [hostSearch, setHostSearch] = useState("");
    const [hostFilter, setHostFilter] = useState<HostFilter>("all");

    // Compute host IDs covered by currently assigned groups
    const viaGroupHostIds = useMemo(
        () =>
            new Set(
                detail.host_groups
                    .filter((g) => draft.assignedGroupIds.has(g.id))
                    .flatMap((g) => g.hosts.map((h) => h.id)),
            ),
        [detail.host_groups, draft.assignedGroupIds],
    );

    const effectiveHostIds = useMemo(
        () => new Set([...draft.assignedHostIds, ...viaGroupHostIds]),
        [draft.assignedHostIds, viaGroupHostIds],
    );

    const directCount = [...draft.assignedHostIds].filter((id) => !viaGroupHostIds.has(id)).length;

    const filteredHosts = useMemo(() => {
        const q = hostSearch.trim().toLowerCase();
        return detail.individual_hosts.filter((h) => {
            if (q && !h.fqdn.toLowerCase().includes(q)) return false;
            const isDirect = draft.assignedHostIds.has(h.id);
            const isVia = viaGroupHostIds.has(h.id);
            switch (hostFilter) {
                case "granted": return isDirect || isVia;
                case "not-granted": return !isDirect && !isVia;
                case "via-group": return isVia;
                default: return true;
            }
        });
    }, [detail.individual_hosts, hostSearch, hostFilter, draft.assignedHostIds, viaGroupHostIds]);

    const unconfigured = !draft.allowAllHosts && draft.assignedGroupIds.size === 0 && draft.assignedHostIds.size === 0;

    return (
        <Stack gap="lg">
            {/* Allow all hosts toggle */}
            <Stack gap={4}>
                <Switch
                    label="Allow all hosts"
                    checked={draft.allowAllHosts}
                    onChange={(e) => dispatch({ type: "setAllowAll", value: e.currentTarget.checked })}
                />
                <Text size="xs" c="dimmed">
                    When enabled, all hosts are accessible through this policy.
                </Text>
            </Stack>

            {/* Unconfigured state banner */}
            {unconfigured && (
                <Alert icon={<IconAlertTriangle size={16} />} color="yellow">
                    <Text size="sm" fw={500}>This policy has no host access configured.</Text>
                    <Text size="sm">
                        Traffic from <Text span ff="monospace">{detail.cidr}</Text> is currently denied
                        for all hosts. Add host groups or individual hosts below, or enable "Allow all".
                    </Text>
                </Alert>
            )}

            {/* Effective access summary */}
            {!draft.allowAllHosts && effectiveHostIds.size > 0 && (
                <Group gap="xs" wrap="nowrap">
                    <Text size="sm" c="dimmed">Effective access:</Text>
                    <Text size="sm">
                        {effectiveHostIds.size} of {detail.total_host_count} hosts
                    </Text>
                    {viaGroupHostIds.size > 0 && (
                        <Text size="sm" c="dimmed">· {viaGroupHostIds.size} via groups</Text>
                    )}
                    {directCount > 0 && (
                        <Text size="sm" c="dimmed">· {directCount} direct</Text>
                    )}
                    <Progress
                        value={detail.total_host_count > 0 ? (effectiveHostIds.size / detail.total_host_count) * 100 : 0}
                        size="xs"
                        style={{ flex: 1, maxWidth: 80 }}
                        color="indigo"
                    />
                </Group>
            )}

            {/* Two-column layout */}
            <Grid>
                {/* Left: host groups */}
                <Grid.Col span={{ base: 12, md: 5 }}>
                    <Stack gap="xs">
                        <Text size="sm" fw={600} c="dimmed" tt="uppercase" style={{ letterSpacing: "0.05em" }}>
                            Host groups
                        </Text>
                        {detail.host_groups.length === 0 ? (
                            <Text size="sm" c="dimmed">No host groups configured.</Text>
                        ) : (
                            detail.host_groups.map((group) => {
                                const isAssigned = draft.assignedGroupIds.has(group.id);
                                const preview = group.hosts.slice(0, MAX_PREVIEW);
                                const overflow = group.hosts.length - preview.length;
                                return (
                                    <Card
                                        key={group.id}
                                        withBorder
                                        p="sm"
                                        style={{
                                            cursor: "pointer",
                                            opacity: draft.allowAllHosts ? 0.5 : 1,
                                            borderColor: isAssigned
                                                ? "var(--mantine-color-indigo-5)"
                                                : undefined,
                                        }}
                                        onClick={() => {
                                            if (!draft.allowAllHosts) {
                                                dispatch({ type: "toggleGroup", id: group.id, assigned: !isAssigned });
                                            }
                                        }}
                                    >
                                        <Group gap="sm" wrap="nowrap">
                                            <Checkbox
                                                checked={isAssigned}
                                                onChange={() => {}}
                                                tabIndex={-1}
                                                aria-hidden
                                                disabled={draft.allowAllHosts}
                                                onClick={(e) => e.stopPropagation()}
                                            />
                                            <Stack gap={2} style={{ flex: 1, minWidth: 0 }}>
                                                <Group gap="xs">
                                                    {group.color && (
                                                        <div
                                                            style={{
                                                                width: 10,
                                                                height: 10,
                                                                borderRadius: "50%",
                                                                background: group.color,
                                                                flexShrink: 0,
                                                            }}
                                                        />
                                                    )}
                                                    <Text size="sm" fw={500} tt="uppercase">
                                                        {group.name}
                                                    </Text>
                                                    <Text size="xs" c="dimmed">{group.hosts.length} hosts</Text>
                                                </Group>
                                                {preview.map((h) => (
                                                    <Text key={h.id} size="xs" c="dimmed" truncate>
                                                        {h.fqdn}
                                                    </Text>
                                                ))}
                                                {overflow > 0 && (
                                                    <Text size="xs" c="dimmed">+{overflow} more</Text>
                                                )}
                                            </Stack>
                                        </Group>
                                    </Card>
                                );
                            })
                        )}
                    </Stack>
                </Grid.Col>

                {/* Right: individual hosts */}
                <Grid.Col span={{ base: 12, md: 7 }}>
                    <Stack gap="xs">
                        <Text size="sm" fw={600} c="dimmed" tt="uppercase" style={{ letterSpacing: "0.05em" }}>
                            Individual hosts
                        </Text>
                        <TextInput
                            placeholder="Search hosts..."
                            leftSection={<IconSearch size={14} />}
                            value={hostSearch}
                            onChange={(e) => setHostSearch(e.currentTarget.value)}
                            disabled={draft.allowAllHosts}
                            size="sm"
                        />
                        <SegmentedControl
                            size="xs"
                            value={hostFilter}
                            onChange={(v) => setHostFilter(v as HostFilter)}
                            disabled={draft.allowAllHosts}
                            data={[
                                { label: "All", value: "all" },
                                { label: "Granted", value: "granted" },
                                { label: "Not granted", value: "not-granted" },
                                { label: "Via group", value: "via-group" },
                            ]}
                        />
                        <Stack gap={4} style={{ opacity: draft.allowAllHosts ? 0.5 : 1 }}>
                            {filteredHosts.length === 0 ? (
                                <Text size="sm" c="dimmed" ta="center" py="md">
                                    No hosts match.
                                </Text>
                            ) : (
                                filteredHosts.map((host) => {
                                    const isDirect = draft.assignedHostIds.has(host.id);
                                    const isVia = viaGroupHostIds.has(host.id);
                                    return (
                                        <Group key={host.id} gap="sm" wrap="nowrap">
                                            <Checkbox
                                                checked={isDirect || isVia}
                                                indeterminate={isVia && !isDirect}
                                                disabled={draft.allowAllHosts || isVia}
                                                onChange={() => {
                                                    if (!draft.allowAllHosts && !isVia) {
                                                        dispatch({ type: "toggleHost", id: host.id, assigned: !isDirect });
                                                    }
                                                }}
                                            />
                                            <Text size="sm" style={{ flex: 1, minWidth: 0 }} truncate>
                                                {host.fqdn}
                                            </Text>
                                            {isDirect && !isVia && (
                                                <Badge size="xs" variant="outline" color="indigo">
                                                    DIRECT
                                                </Badge>
                                            )}
                                            {isVia && (
                                                <Badge size="xs" variant="light" color="gray">
                                                    VIA GROUP
                                                </Badge>
                                            )}
                                        </Group>
                                    );
                                })
                            )}
                        </Stack>
                    </Stack>
                </Grid.Col>
            </Grid>

            <StagedChangesBar
                visible={isDirty}
                summary="You have unsaved changes."
                saving={isSaving}
                onSave={onSave}
                onDiscard={onDiscard}
            />
        </Stack>
    );
}
