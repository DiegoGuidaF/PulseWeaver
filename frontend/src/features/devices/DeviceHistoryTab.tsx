import { AddressHistoryView } from "@/features/address-history/components/AddressHistoryView";
import { useLocalAddressHistoryFilters } from "@/features/address-history/hooks/useLocalAddressHistoryFilters";

interface DeviceHistoryTabProps {
    deviceId: number;
}

export function DeviceHistoryTab({ deviceId }: DeviceHistoryTabProps) {
    const filters = useLocalAddressHistoryFilters({ lockedDeviceId: deviceId });
    return <AddressHistoryView filters={filters} />;
}
