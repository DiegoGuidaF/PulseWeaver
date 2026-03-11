import { Link, Navigate, useParams } from "react-router-dom";
import { ChevronLeft } from "lucide-react";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useDeviceDetail } from "@/features/devices/hooks/useDeviceDetail";
import { DeviceAddressesTab } from "@/features/devices/DeviceAddressesTab";
import { DeviceSettingsTab } from "@/features/devices/DeviceSettingsTab";
import { toErrorMessage } from "@/lib/api-client";

type DeviceDetailRouteParams = {
  deviceId?: string;
};

export function DeviceDetailPage() {
  const params = useParams<DeviceDetailRouteParams>();
  const deviceIdParam = params.deviceId;
  const deviceId = deviceIdParam
    ? Number.parseInt(deviceIdParam, 10)
    : Number.NaN;

  const { data: device, isLoading, isError, error } = useDeviceDetail(deviceId);

  if (!deviceIdParam || Number.isNaN(deviceId)) {
    return <Navigate to="/devices" replace />;
  }

  let headerContent: React.ReactNode;

  if (isLoading && !device) {
    headerContent = (
      <div className="space-y-2">
        <Skeleton className="h-7 w-48" />
        <Skeleton className="h-4 w-32" />
      </div>
    );
  } else if (device) {
    headerContent = (
      <div className="space-y-1">
        <h1 className="text-2xl font-bold tracking-tight">{device.name}</h1>
        <p className="text-sm text-muted-foreground">
          ID{" "}
          <span className="font-mono text-xs md:text-sm">{device.id}</span>
        </p>
      </div>
    );
  } else if (isError) {
    headerContent = (
      <p className="text-sm text-red-500">
        Error loading device: {toErrorMessage(error)}
      </p>
    );
  } else {
    headerContent = (
      <p className="text-sm text-muted-foreground">
        Device not found.{" "}
        <Link to="/devices" className="underline">
          Back to devices
        </Link>
      </p>
    );
  }

  return (
    <div className="w-full max-w-5xl space-y-8">
      <div className="space-y-4">
        <div>
          <Link
            to="/devices"
            className="text-muted-foreground hover:text-foreground inline-flex items-center gap-1 text-sm"
          >
            <ChevronLeft className="h-4 w-4" />
            <span>Back to devices</span>
          </Link>
        </div>
        {headerContent}
      </div>

      <Tabs defaultValue="addresses" className="space-y-4">
        <TabsList>
          <TabsTrigger value="addresses">Addresses</TabsTrigger>
          <TabsTrigger value="settings">Settings &amp; Rules</TabsTrigger>
        </TabsList>
        <TabsContent value="addresses">
          <DeviceAddressesTab deviceId={deviceId} />
        </TabsContent>
        <TabsContent value="settings">
          <DeviceSettingsTab deviceId={deviceId} device={device} />
        </TabsContent>
      </Tabs>
    </div>
  );
}
