import { schemaResolver, useForm } from "@mantine/form";
import { z } from "zod";
import { Button, Group, Modal, Stack, Text, Textarea, TextInput } from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { zCreateNetworkPolicyRequest } from "@/lib/api/zod.gen";
import { toApiError, toErrorMessage } from "@/lib/api-client";
import { useCreateNetworkPolicy } from "../hooks/useCreateNetworkPolicy";
import { CIDR_ERROR, CIDR_EXAMPLE, isValidCidr } from "../constants";
import type { NetworkPolicyDetail } from "@/lib/api";

const formSchema = zCreateNetworkPolicyRequest.superRefine((val, ctx) => {
    if (val.cidr && !isValidCidr(val.cidr)) {
        ctx.addIssue({
            code: "custom",
            path: ["cidr"],
            message: CIDR_ERROR,
        });
    }
});

type FormValues = z.infer<typeof formSchema>;

interface Props {
    opened: boolean;
    onClose: () => void;
    onCreated: (policy: NetworkPolicyDetail) => void;
}

export function CreateNetworkPolicyModal({ opened, onClose, onCreated }: Props) {
    const form = useForm<FormValues>({
        validateInputOnBlur: true,
        validate: schemaResolver(formSchema),
        initialValues: { name: "", cidr: "", description: null },
    });

    const createMutation = useCreateNetworkPolicy({
        onSuccess: (data) => {
            form.reset();
            onCreated(data);
        },
    });

    function handleClose() {
        form.reset();
        onClose();
    }

    function onSubmit(values: FormValues) {
        createMutation.mutate(
            { body: { name: values.name, cidr: values.cidr, description: values.description ?? null } },
            {
                onError: (err) => {
                    const status = toApiError(err).status;
                    if (status === 409) {
                        form.setFieldError("cidr", "A policy with this CIDR already exists.");
                    } else {
                        notifications.show({ color: "red", title: "Error creating policy", message: toErrorMessage(err) });
                    }
                },
            },
        );
    }

    return (
        <Modal
            opened={opened}
            onClose={handleClose}
            title="New network policy"
            size="md"
            closeOnClickOutside={false}
        >
            <form onSubmit={form.onSubmit(onSubmit)}>
                <Stack gap="md">
                    <TextInput
                        label="Name"
                        placeholder="e.g. Home Office"
                        data-autofocus
                        withAsterisk
                        {...form.getInputProps("name")}
                    />
                    <div>
                        <TextInput
                            label="CIDR range"
                            placeholder={`e.g. ${CIDR_EXAMPLE}`}
                            ff="monospace"
                            withAsterisk
                            {...form.getInputProps("cidr")}
                        />
                        <Text size="xs" c="dimmed" mt={4}>
                            Host bits are zeroed automatically
                        </Text>
                    </div>
                    <Textarea
                        label="Description"
                        placeholder="Optional notes about this policy"
                        autosize
                        maxRows={3}
                        value={form.values.description ?? ""}
                        onChange={(e) =>
                            form.setFieldValue("description", e.currentTarget.value || null)
                        }
                        error={form.errors.description}
                    />
                    <Group justify="flex-end" gap="sm">
                        <Button variant="outline" onClick={handleClose} disabled={createMutation.isPending}>
                            Cancel
                        </Button>
                        <Button type="submit" loading={createMutation.isPending}>
                            Create policy
                        </Button>
                    </Group>
                </Stack>
            </form>
        </Modal>
    );
}
