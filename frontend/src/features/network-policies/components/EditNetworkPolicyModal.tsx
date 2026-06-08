import { schemaResolver, useForm } from "@mantine/form";
import { z } from "zod";
import { Alert, Button, Group, Modal, Stack, Text, Textarea, TextInput } from "@mantine/core";
import { IconAlertTriangle } from "@tabler/icons-react";
import { notifications } from "@mantine/notifications";
import { zModifyNetworkPolicyRequest } from "@/lib/api/zod.gen";
import { broadCidrWarning, cidrTooBroadError, CIDR_ERROR, CIDR_EXAMPLE, classifyCidr, isValidCidr, normalCidrNote } from "../constants";
import type { ModifyNetworkPolicyRequest, NetworkPolicyDetail } from "@/lib/api";

const formSchema = zModifyNetworkPolicyRequest
    .pick({ name: true, cidr: true, description: true })
    .superRefine((val, ctx) => {
        if (!val.cidr) return;
        if (!isValidCidr(val.cidr)) {
            ctx.addIssue({ code: "custom", path: ["cidr"], message: CIDR_ERROR });
        } else if (classifyCidr(val.cidr) === "reject") {
            ctx.addIssue({ code: "custom", path: ["cidr"], message: cidrTooBroadError(val.cidr) });
        }
    });

type FormValues = z.infer<typeof formSchema>;

interface Props {
    policy: NetworkPolicyDetail;
    opened: boolean;
    onClose: () => void;
    onUpdate: (fields: Partial<ModifyNetworkPolicyRequest>, opts?: { onSuccess?: () => void }) => void;
    isUpdating?: boolean;
}

export function EditNetworkPolicyModal({ policy, opened, onClose, onUpdate, isUpdating }: Props) {
    const form = useForm<FormValues>({
        validateInputOnBlur: true,
        validateInputOnChange: ["cidr"],
        validate: schemaResolver(formSchema),
        initialValues: {
            name: policy.name,
            cidr: policy.cidr,
            description: policy.description ?? null,
        },
    });

    const cidrWarning = broadCidrWarning(form.values.cidr);
    const cidrNote = normalCidrNote(form.values.cidr);

    function onSubmit(values: FormValues) {
        onUpdate(
            { name: values.name, cidr: values.cidr, description: values.description ?? "" },
            {
                onSuccess: () => {
                    notifications.show({ color: "green", message: "Policy updated." });
                    onClose();
                },
            },
        );
    }

    return (
        <Modal
            opened={opened}
            onClose={onClose}
            title="Edit network policy"
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
                            Host bits are zeroed automatically{cidrNote ? ` · ${cidrNote}` : ""}
                        </Text>
                        {cidrWarning && (
                            <Alert
                                variant="light"
                                color="yellow"
                                icon={<IconAlertTriangle size={16} />}
                                mt="xs"
                                p="xs"
                            >
                                {cidrWarning}
                            </Alert>
                        )}
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
                        <Button variant="outline" onClick={onClose} disabled={isUpdating}>
                            Cancel
                        </Button>
                        <Button type="submit" loading={isUpdating}>
                            Save changes
                        </Button>
                    </Group>
                </Stack>
            </form>
        </Modal>
    );
}
