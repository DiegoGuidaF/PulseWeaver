import { Stack } from "@mantine/core";
import { AddressHistoryView } from "@/features/address-history/components/AddressHistoryView";
import { useAddressHistoryFilters } from "@/features/address-history/hooks/useAddressHistoryFilters";

export function AddressHistoryPage() {
    const filters = useAddressHistoryFilters();

    return (
        <Stack maw={1200} gap="xl">
            <AddressHistoryView filters={filters} subtitle="IP address lease events across all devices" />
        </Stack>
    );
}
