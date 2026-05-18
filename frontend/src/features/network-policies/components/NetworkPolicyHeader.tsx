import { useState } from "react";
import { Link } from "react-router-dom";
import {
    ActionIcon,
    Anchor,
    Badge,
    Group,
    Menu,
    Stack,
    Switch,
    Text,
    TextInput,
    Textarea,
} from "@mantine/core";
import { IconChevronLeft, IconDots, IconTrash } from "@tabler/icons-react";
import type { NetworkPolicyDetail, ModifyNetworkPolicyRequest } from "@/lib/api";
import { CIDR_RE } from "../constants";
import { DeleteNetworkPolicyModal } from "./DeleteNetworkPolicyModal";

interface InlineEditProps {
    value: string;
    onSave: (value: string) => void;
    validate?: (value: string) => string | null;
    placeholder?: string;
    monospace?: boolean;
    size?: string;
    fw?: number;
    c?: string;
}

function InlineEdit({ value, onSave, validate, placeholder, monospace, size, fw, c }: InlineEditProps) {
    const [editing, setEditing] = useState(false);
    const [draft, setDraft] = useState(value);
    const [error, setError] = useState<string | null>(null);

    function startEdit() {
        setDraft(value);
        setError(null);
        setEditing(true);
    }

    function commit() {
        const trimmed = draft.trim();
        if (trimmed === value) { setEditing(false); return; }
        if (validate) {
            const err = validate(trimmed);
            if (err) { setError(err); return; }
        }
        onSave(trimmed);
        setEditing(false);
    }

    function cancel() {
        setEditing(false);
        setError(null);
    }

    if (editing) {
        return (
            <Stack gap={4}>
                <TextInput
                    value={draft}
                    onChange={(e) => { setDraft(e.currentTarget.value); setError(null); }}
                    onBlur={commit}
                    onKeyDown={(e) => {
                        if (e.key === "Enter") commit();
                        if (e.key === "Escape") cancel();
                    }}
                    error={error}
                    ff={monospace ? "monospace" : undefined}
                    autoFocus
                    size="sm"
                />
            </Stack>
        );
    }

    return (
        <Text
            size={size}
            fw={fw}
            c={c}
            style={{ cursor: "text" }}
            ff={monospace ? "monospace" : undefined}
            onClick={startEdit}
            title="Click to edit"
        >
            {value || <Text span c="dimmed">{placeholder}</Text>}
        </Text>
    );
}

interface InlineDescriptionEditProps {
    value: string | null;
    onSave: (value: string | null) => void;
}

function InlineDescriptionEdit({ value, onSave }: InlineDescriptionEditProps) {
    const [editing, setEditing] = useState(false);
    const [draft, setDraft] = useState(value ?? "");

    function commit() {
        const trimmed = draft.trim();
        const next = trimmed || null;
        if (next !== value) onSave(next);
        setEditing(false);
    }

    if (editing) {
        return (
            <Textarea
                value={draft}
                onChange={(e) => setDraft(e.currentTarget.value)}
                onBlur={commit}
                onKeyDown={(e) => { if (e.key === "Escape") { setEditing(false); } }}
                autosize
                maxRows={4}
                autoFocus
                size="sm"
                placeholder="Add a description..."
            />
        );
    }

    return (
        <Text
            size="sm"
            c={value ? undefined : "dimmed"}
            style={{ cursor: "text" }}
            onClick={() => { setDraft(value ?? ""); setEditing(true); }}
            title="Click to edit"
        >
            {value || "Add a description..."}
        </Text>
    );
}

interface Props {
    policy: NetworkPolicyDetail;
    onUpdate: (fields: Partial<ModifyNetworkPolicyRequest>) => void;
    onDelete: () => void;
    isUpdating?: boolean;
    isDeleting?: boolean;
}

export function NetworkPolicyHeader({ policy, onUpdate, onDelete, isUpdating, isDeleting }: Props) {
    const [confirmDeleteOpen, setConfirmDeleteOpen] = useState(false);

    return (
        <>
            <div>
                <Anchor component={Link} to="/access/network-policies" size="sm" c="dimmed">
                    <Group gap={4}>
                        <IconChevronLeft size={14} />
                        Network Policies
                    </Group>
                </Anchor>

                <Group justify="space-between" align="flex-start" mt="xs">
                    <Stack gap={4} style={{ flex: 1 }}>
                        <InlineEdit
                            value={policy.name}
                            onSave={(name) => onUpdate({ name })}
                            validate={(v) => v.length < 1 ? "Name is required" : null}
                            size="xl"
                            fw={700}
                        />
                        <InlineEdit
                            value={policy.cidr}
                            onSave={(cidr) => onUpdate({ cidr })}
                            validate={(v) => CIDR_RE.test(v) ? null : "Enter a valid CIDR range, e.g. 192.168.1.0/24"}
                            monospace
                            size="sm"
                            c="dimmed"
                        />
                        <InlineDescriptionEdit
                            value={policy.description ?? null}
                            onSave={(description) => onUpdate({ description: description ?? "" })}
                        />
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
