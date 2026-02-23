import { useState } from "react";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { format } from "date-fns";
import { Trash2 } from "lucide-react";
import type { Device } from "@/lib/api";
import { DeviceAddressesDialog } from "@/features/devices/DeviceAddressesDialog";
import { useDevices } from "@/features/devices/hooks/useDevices";
import { useDeleteDevice } from "@/features/devices/hooks/useDeleteDevice";
import { toErrorMessage } from "@/lib/api-client";

export function DeviceList() {
  const { data: devices, isLoading, error } = useDevices();
  const deleteDevice = useDeleteDevice();
  const [deviceToDelete, setDeviceToDelete] = useState<Device | null>(null);

  function handleConfirmDelete() {
    if (!deviceToDelete) return;
    deleteDevice.mutate(
      { path: { device_id: deviceToDelete.id } },
      {
        onSettled: () => {
          setDeviceToDelete(null);
        },
      }
    );
  }

  if (error)
    return (
      <div className="p-4 text-red-500">Error: {toErrorMessage(error)}</div>
    );

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Devices</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-2">
            <div className="flex justify-between border-b py-2">
              <Skeleton className="h-4 w-[100px]" />
              <Skeleton className="h-4 w-[100px]" />
              <Skeleton className="h-4 w-[80px]" />
              <Skeleton className="h-4 w-[150px]" />
            </div>
            {Array.from({ length: 5 }).map((_, i) => (
              <div key={i} className="flex justify-between border-b py-4">
                <Skeleton className="h-4 w-[120px]" />
                <Skeleton className="h-4 w-[80px]" />
                <Skeleton className="h-4 w-[200px]" />
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Devices</CardTitle>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>ID</TableHead>
              <TableHead>Key prefix</TableHead>
              <TableHead>Created At</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {devices?.length === 0 ? (
              <TableRow>
                <TableCell colSpan={5} className="h-32 text-center">
                  <div className="flex flex-col items-center justify-center space-y-2">
                    <p className="text-muted-foreground">No devices found.</p>
                    <p className="text-sm text-gray-400">
                      Add a device above to get started.
                    </p>
                  </div>
                </TableCell>
              </TableRow>
            ) : (
              devices?.map((device) => (
                <TableRow key={device.id}>
                  <TableCell className="font-medium">{device.name}</TableCell>
                  <TableCell className="font-mono text-xs">
                    {device.id}
                  </TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground">
                    {device.api_key_prefix}
                  </TableCell>
                  <TableCell>
                    {format(new Date(device.created_at), "PP p")}
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex items-center justify-end gap-2">
                      <DeviceAddressesDialog
                        deviceId={device.id}
                        deviceName={device.name}
                      />
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        aria-label={`Delete device ${device.name}`}
                        onClick={() => setDeviceToDelete(device)}
                        disabled={deleteDevice.isPending}
                      >
                        <Trash2 className="h-4 w-4 text-muted-foreground" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </CardContent>
      <Dialog
        open={deviceToDelete !== null}
        onOpenChange={(open) => {
          if (!open) setDeviceToDelete(null);
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete device</DialogTitle>
            <DialogDescription>
              Delete device &quot;{deviceToDelete?.name}&quot;? It will be
              hidden from the list and cannot receive addresses.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => setDeviceToDelete(null)}
            >
              Cancel
            </Button>
            <Button
              type="button"
              variant="destructive"
              onClick={handleConfirmDelete}
              disabled={deleteDevice.isPending}
            >
              {deleteDevice.isPending ? "Deleting..." : "Delete"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </Card>
  );
}
