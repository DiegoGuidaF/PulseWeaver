import { Button, Group, Modal, Stack, Text } from "@mantine/core";

interface Props {
    policyName: string;
    opened: boolean;
    isDeleting?: boolean;
    onConfirm: () => void;
    onClose: () => void;
}

export function DeleteNetworkPolicyModal({ policyName, opened, isDeleting, onConfirm, onClose }: Props) {
    return (
        <Modal
            opened={opened}
            onClose={onClose}
            title="Delete network policy"
            size="sm"
            closeOnClickOutside={!isDeleting}
        >
            <Stack gap="md">
                <Text size="sm">
                    Are you sure you want to delete{" "}
                    <Text span fw={600}>{policyName}</Text>? This will permanently remove the
                    policy and all host access configuration. This action cannot be undone.
                </Text>
                <Group justify="flex-end" gap="sm">
                    <Button variant="outline" onClick={onClose} disabled={isDeleting}>
                        Cancel
                    </Button>
                    <Button color="red" onClick={onConfirm} loading={isDeleting}>
                        Delete policy
                    </Button>
                </Group>
            </Stack>
        </Modal>
    );
}
