import { Stack } from "@mantine/core";
import { AddressHistoryView } from "@/features/address-history/components/AddressHistoryView";
import { useAddressHistoryFilters } from "@/features/address-history/hooks/useAddressHistoryFilters";

export function AddressHistoryPage() {
    const filters = useAddressHistoryFilters();

    return (
        <Stack maw={1200} gap="xl">
            <h1 style={{ position: "absolute", width: 1, height: 1, padding: 0, margin: -1, overflow: "hidden", clip: "rect(0,0,0,0)", whiteSpace: "nowrap", border: 0 }}>Address history</h1>
            <AddressHistoryView filters={filters} subtitle="IP address lease events across all devices" />
        </Stack>
    );
}
