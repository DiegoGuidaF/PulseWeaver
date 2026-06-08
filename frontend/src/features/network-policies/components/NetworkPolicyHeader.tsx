import { useState } from "react";
import { Link } from "react-router-dom";
import { ROUTES } from "@/lib/routes";
import {
    ActionIcon,
    Anchor,
    Badge,
    Group,
    Menu,
    Stack,
    Switch,
    Text,
} from "@mantine/core";
import { IconChevronLeft, IconDots, IconPencil, IconTrash } from "@tabler/icons-react";
import type { NetworkPolicyDetail, ModifyNetworkPolicyRequest } from "@/lib/api";
import { DeleteNetworkPolicyModal } from "./DeleteNetworkPolicyModal";
import { EditNetworkPolicyModal } from "./EditNetworkPolicyModal";

interface Props {
    policy: NetworkPolicyDetail;
    onUpdate: (fields: Partial<ModifyNetworkPolicyRequest>, opts?: { onSuccess?: () => void }) => void;
    onDelete: () => void;
    isUpdating?: boolean;
    isDeleting?: boolean;
}

export function NetworkPolicyHeader({ policy, onUpdate, onDelete, isUpdating, isDeleting }: Props) {
    const [confirmDeleteOpen, setConfirmDeleteOpen] = useState(false);
    const [editOpen, setEditOpen] = useState(false);

    return (
        <>
            <div>
                <Anchor component={Link} to={ROUTES.accessNetworkPolicies} size="sm" c="dimmed">
                    <Group gap={4}>
                        <IconChevronLeft size={14} />
                        Network Policies
                    </Group>
                </Anchor>

                <Group justify="space-between" align="flex-start" mt="xs">
                    <Stack gap={4} style={{ flex: 1 }}>
                        <Text size="xl" fw={700}>{policy.name}</Text>
                        <Text size="sm" c="dimmed" ff="monospace">{policy.cidr}</Text>
                        <Text size="sm" c={policy.description ? undefined : "dimmed"}>
                            {policy.description || "No description"}
                        </Text>
                    </Stack>

                    <Group gap="sm" align="center">
                        <Switch
                            checked={policy.enabled}
                            onChange={(e) => onUpdate({ enabled: e.currentTarget.checked })}
                            disabled={isUpdating}
                            label={
                                <Badge
                                    variant="dot"
                                    color={policy.enabled ? "green" : "gray"}
                                    size="sm"
                                >
                                    {policy.enabled ? "Enabled" : "Disabled"}
                                </Badge>
                            }
                        />
                        <Menu position="bottom-end" withArrow>
                            <Menu.Target>
                                <ActionIcon variant="subtle" color="gray">
                                    <IconDots size={18} />
                                </ActionIcon>
                            </Menu.Target>
                            <Menu.Dropdown>
                                <Menu.Item
                                    leftSection={<IconPencil size={14} />}
                                    onClick={() => setEditOpen(true)}
                                >
                                    Edit policy
                                </Menu.Item>
                                <Menu.Item
                                    color="red"
                                    leftSection={<IconTrash size={14} />}
                                    onClick={() => setConfirmDeleteOpen(true)}
                                >
                                    Delete policy
                                </Menu.Item>
                            </Menu.Dropdown>
                        </Menu>
                    </Group>
                </Group>
            </div>

            {editOpen && (
                <EditNetworkPolicyModal
                    policy={policy}
                    opened={editOpen}
                    isUpdating={isUpdating}
                    onUpdate={onUpdate}
                    onClose={() => setEditOpen(false)}
                />
            )}

            <DeleteNetworkPolicyModal
                policyName={policy.name}
                opened={confirmDeleteOpen}
                isDeleting={isDeleting}
                onConfirm={onDelete}
                onClose={() => setConfirmDeleteOpen(false)}
            />
        </>
    );
}
