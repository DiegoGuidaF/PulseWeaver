import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Badge, Button, Center, Group, Stack, Text, ThemeIcon, Tooltip } from "@mantine/core";
import { IconNetwork } from "@tabler/icons-react";
import { DataTable, type DataTableSortStatus } from "mantine-datatable";
import { notifications } from "@mantine/notifications";
import type { NetworkPolicyListItem } from "@/lib/api";
import { useDateFormatter } from "@/contexts/useDateTimePrefs";
import { toErrorMessage } from "@/lib/api-client";
import { formatEffectiveAccess } from "@/features/subjects/constants";
import { GroupFilterBar } from "@/features/subjects/components/GroupFilterBar";
import { useDeleteNetworkPolicy } from "../hooks/useDeleteNetworkPolicy";
import { DeleteNetworkPolicyModal } from "./DeleteNetworkPolicyModal";

interface Props {
  policies: NetworkPolicyListItem[];
  onNewPolicy: () => void;
}

function collectGroups(policies: NetworkPolicyListItem[]) {
  const seen = new Map<number, { id: number; name: string }>();
  for (const p of policies) {
    for (const g of p.groups) {
      if (!seen.has(g.id)) seen.set(g.id, g);
    }
  }
  return [...seen.values()].sort((a, b) => a.name.localeCompare(b.name));
}

export function NetworkPoliciesTable({ policies, onNewPolicy }: Props) {
  const navigate = useNavigate();
  const formatDateTime = useDateFormatter();
  const [deleteTarget, setDeleteTarget] = useState<NetworkPolicyListItem | null>(null);
  const [groupFilter, setGroupFilter] = useState<Set<number>>(new Set());
  const [sortStatus, setSortStatus] = useState<DataTableSortStatus<NetworkPolicyListItem>>({
    columnAccessor: "name",
    direction: "asc",
  });

  const deleteMutation = useDeleteNetworkPolicy({ onSuccess: () => setDeleteTarget(null) });

  const allGroups = useMemo(() => collectGroups(policies), [policies]);

  const displayedPolicies = useMemo(() => {
    let list = policies;

    if (groupFilter.size > 0) {
      list = list.filter((p) => p.groups.some((g) => groupFilter.has(g.id)));
    }

    const { columnAccessor, direction } = sortStatus;
    const mult = direction === "asc" ? 1 : -1;
    list = [...list].sort((a, b) => {
      switch (columnAccessor) {
        case "name": return mult * a.name.localeCompare(b.name);
        case "cidr": return mult * a.cidr.localeCompare(b.cidr);
        case "host_count": return mult * (a.host_count - b.host_count);
        case "created_at": return mult * a.created_at.localeCompare(b.created_at);
        default: return 0;
      }
    });

    return list;
  }, [policies, groupFilter, sortStatus]);

  if (policies.length === 0) {
    return (
      <Center py="xl">
        <Stack align="center" gap="sm">
          <IconNetwork size={40} color="var(--mantine-color-dimmed)" />
          <Text fw={500}>No network policies configured.</Text>
          <Text size="sm" c="dimmed" ta="center" maw={380}>
            Add a policy to allow traffic from an IP range without registering individual device
            addresses.
          </Text>
          <Button onClick={onNewPolicy}>+ New policy</Button>
        </Stack>
      </Center>
    );
  }

  return (
    <>
      <GroupFilterBar
        availableGroups={allGroups}
        selected={groupFilter}
        onChange={setGroupFilter}
      />

      <DataTable
        records={displayedPolicies}
        highlightOnHover
        onRowClick={({ record }) => navigate(`/access/network-policies/${record.id}`)}
        sortStatus={sortStatus}
        onSortStatusChange={setSortStatus}
        rowStyle={(r) => (!r.enabled ? { opacity: 0.55 } : undefined)}
        columns={[
          {
            accessor: "enabled",
            title: "Status",
            width: 52,
            render: (p) => (
              <Tooltip
                label={p.enabled ? "Enabled" : "Disabled"}
                withArrow
                position="right"
              >
                <ThemeIcon
                  size="xs"
                  radius="xl"
                  color={p.enabled ? "green" : "gray"}
                  variant="filled"
                  style={{ cursor: "default" }}
                >
                  {" "}
                </ThemeIcon>
              </Tooltip>
            ),
          },
          {
            accessor: "name",
            title: "Name",
            sortable: true,
            render: (p) => <Text size="sm" fw={500}>{p.name}</Text>,
          },
          {
            accessor: "cidr",
            title: "CIDR",
            sortable: true,
            render: (p) => <Text size="sm" ff="monospace">{p.cidr}</Text>,
          },
          {
            accessor: "groups",
            title: "Groups",
            render: (p) =>
              p.groups.length === 0 ? (
                <Text size="sm" c="dimmed">—</Text>
              ) : (
                <Group gap={4} wrap="wrap">
                  {p.groups.map((g) => (
                    <Badge
                      key={g.id}
                      size="xs"
                      variant={groupFilter.has(g.id) ? "filled" : "outline"}
                      color="indigo"
                      style={{ cursor: "pointer" }}
                      onClick={(e) => {
                        e.stopPropagation();
                        const next = new Set(groupFilter);
                        if (next.has(g.id)) next.delete(g.id);
                        else next.add(g.id);
                        setGroupFilter(next);
                      }}
                    >
                      {g.name}
                    </Badge>
                  ))}
                </Group>
              ),
          },
          {
            accessor: "host_count",
            title: "Effective access",
            sortable: true,
            render: (p) => {
              const text = formatEffectiveAccess(p);
              if (p.bypass_host_check) {
                return <Badge size="sm" color="orange" variant="light">{text}</Badge>;
              }
              return <Text size="sm" c={p.host_count === 0 ? "dimmed" : undefined}>{text}</Text>;
            },
          },
          {
            accessor: "created_at",
            title: "Created",
            sortable: true,
            render: (p) => <Text size="sm" c="dimmed">{formatDateTime(p.created_at)}</Text>,
          },
        ]}
      />

      <DeleteNetworkPolicyModal
        policyName={deleteTarget?.name ?? ""}
        opened={deleteTarget != null}
        isDeleting={deleteMutation.isPending}
        onClose={() => setDeleteTarget(null)}
        onConfirm={() => {
          if (!deleteTarget) return;
          deleteMutation.mutate(
            { path: { id: deleteTarget.id } },
            {
              onError: (err) =>
                notifications.show({ color: "red", message: toErrorMessage(err) }),
            },
          );
        }}
      />
    </>
  );
}
