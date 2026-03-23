import { Stack, Title, Text } from "@mantine/core";
import { AddressHistoryView } from "@/features/address-history/components/AddressHistoryView";
import { useAddressHistoryFilters } from "@/features/address-history/hooks/useAddressHistoryFilters";

export function AddressHistoryPage() {
    const filters = useAddressHistoryFilters();

    return (
        <Stack maw={1200} gap="xl">
            <div>
                <Title order={1}>Address Log</Title>
                <Text c="dimmed">IP address lease events across all devices.</Text>
            </div>
            <AddressHistoryView filters={filters} />
        </Stack>
    );
}
