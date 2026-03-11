import { useState } from "react";
import { Link, Navigate, useParams } from "react-router-dom";
import { ChevronLeft, RefreshCw } from "lucide-react";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useDevice } from "@/features/devices/hooks/useDevice";
import { useRegenerateApiKey } from "@/features/devices/hooks/useRegenerateApiKey";
import { DeviceAddressesTab } from "@/features/devices/DeviceAddressesTab";
import { DeviceSettingsTab } from "@/features/devices/DeviceSettingsTab";
import { toErrorMessage } from "@/lib/api-client";
import { toast } from "sonner";

type DeviceDetailRouteParams = {
  deviceId?: string;
};

export function DeviceDetailPage() {
  const params = useParams<DeviceDetailRouteParams>();
  const deviceIdParam = params.deviceId;
  const deviceId = deviceIdParam
    ? Number.parseInt(deviceIdParam, 10)
    : Number.NaN;

  const { data: device, isLoading, isError, error } = useDevice(deviceId);
  const regenerateApiKey = useRegenerateApiKey();

  const [regeneratedApiKey, setRegeneratedApiKey] = useState<string | null>(
    null,
  );

  if (!deviceIdParam || Number.isNaN(deviceId)) {
    return <Navigate to="/devices" replace />;
  }

  async function handleCopyRegeneratedKey() {
    if (!regeneratedApiKey) return;

    if (!("clipboard" in navigator) || !navigator.clipboard?.writeText) {
      toast.error("Copy to clipboard is not supported in this browser.");
      return;
    }

    try {
      await navigator.clipboard.writeText(regeneratedApiKey);
      toast.success("Copied to clipboard");
    } catch {
      toast.error("Failed to copy API key");
    }
  }

  function handleConfirmRegenerate() {
    regenerateApiKey.mutate(
      { path: { device_id: deviceId } },
      {
        onSuccess: (data) => {
          setRegeneratedApiKey(data.api_key);
        },
      },
    );
  }

  let headerContent: React.ReactNode;

  if (isLoading && !device) {
    headerContent = (
      <div className="space-y-2">
        <Skeleton className="h-7 w-48" />
        <Skeleton className="h-4 w-72" />
      </div>
    );
  } else if (device) {
    headerContent = (
      <div className="space-y-1">
        <h1 className="text-2xl font-bold tracking-tight">{device.name}</h1>
        <p className="text-sm text-muted-foreground">
          ID{" "}
          <span className="font-mono text-xs md:text-sm">{device.id}</span>{" "}
          · API key prefix{" "}
          <span className="font-mono text-xs md:text-sm">
            {device.api_key_prefix}
          </span>
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                className="ml-1 h-5 w-5"
                title="Regenerate API key"
                disabled={regenerateApiKey.isPending}
              >
                <RefreshCw className="h-3 w-3" />
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>
                  Regenerate API key for &ldquo;{device.name}&rdquo;?
                </AlertDialogTitle>
                <AlertDialogDescription>
                  The current key (
                  <span className="font-mono">{device.api_key_prefix}&hellip;</span>
                  ) will stop working immediately. You will need to update any
                  scripts or services using this device.
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>Cancel</AlertDialogCancel>
                <AlertDialogAction onClick={handleConfirmRegenerate}>
                  Regenerate
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
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
          <DeviceSettingsTab deviceId={deviceId} />
        </TabsContent>
      </Tabs>

      {/* One-time key display dialog after successful regeneration */}
      <Dialog
        open={regeneratedApiKey !== null}
        onOpenChange={(open) => {
          if (!open) {
            setRegeneratedApiKey(null);
          }
        }}
      >
        <DialogContent
          showCloseButton={false}
          onInteractOutside={(e) => e.preventDefault()}
          onEscapeKeyDown={(e) => e.preventDefault()}
        >
          <DialogHeader>
            <DialogTitle>API key regenerated — save your new key</DialogTitle>
            <DialogDescription>
              This API key is shown only once. Copy it now and store it
              securely. The old key is no longer valid.
            </DialogDescription>
          </DialogHeader>
          {regeneratedApiKey && (
            <div className="space-y-4">
              <div className="space-y-2">
                <p className="text-sm font-medium">New API key</p>
                <div className="flex gap-2">
                  <Input
                    readOnly
                    value={regeneratedApiKey}
                    className="font-mono"
                  />
                  <Button
                    type="button"
                    variant="outline"
                    onClick={handleCopyRegeneratedKey}
                  >
                    Copy
                  </Button>
                </div>
              </div>
              <p className="text-xs text-muted-foreground">
                You will not be able to see this full API key again. Make sure
                you have stored it securely.
              </p>
            </div>
          )}
          <DialogFooter>
            <Button type="button" onClick={() => setRegeneratedApiKey(null)}>
              I&apos;ve saved it
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
