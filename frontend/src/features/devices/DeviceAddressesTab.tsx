import { useState } from "react";

const REFRESH_OPTIONS = [
  { label: "Off", value: 0 },
  { label: "1s", value: 1_000 },
  { label: "5s", value: 5_000 },
  { label: "15s", value: 15_000 },
  { label: "30s", value: 30_000 },
  { label: "1 min", value: 60_000 },
  { label: "5 min", value: 300_000 },
] as const;
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { format } from "date-fns";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import type { Address } from "@/lib/api";
import { cn } from "@/lib/utils";
import { zAddAddressRequest } from "@/lib/api/zod.gen";
import { useDeviceAddresses } from "@/features/devices/hooks/useDeviceAddresses";
import { useAddDeviceAddress } from "@/features/devices/hooks/useAddDeviceAddress";
import { useDisableDeviceAddress } from "@/features/devices/hooks/useDisableDeviceAddress";
import { useDeviceHeartbeat } from "@/features/devices/hooks/useDeviceHeartbeat";

const addressSchema = zAddAddressRequest;

interface DeviceAddressesTabProps {
  deviceId: number;
}

export function DeviceAddressesTab({ deviceId }: DeviceAddressesTabProps) {
  const [refreshInterval, setRefreshInterval] = useState<number>(5_000);
  const { data: addresses, isLoading } = useDeviceAddresses(
    deviceId,
    true,
    refreshInterval === 0 ? false : refreshInterval,
  );
  const heartbeatMutation = useDeviceHeartbeat();
  const form = useForm<z.infer<typeof addressSchema>>({
    resolver: zodResolver(addressSchema),
    defaultValues: { ip: "" },
  });
  const addAddressMutation = useAddDeviceAddress({
    onSuccess: () => form.reset(),
  });
  const disableAddressMutation = useDisableDeviceAddress();
  const [addressToDisable, setAddressToDisable] = useState<Address | null>(
    null,
  );

  function handleAddAddressSubmit(values: z.infer<typeof addressSchema>) {
    addAddressMutation.mutate({
      path: { device_id: deviceId },
      body: { ip: values.ip },
    });
  }

  function handleConfirmDisable() {
    if (!addressToDisable) return;
    disableAddressMutation.mutate(
      {
        path: {
          device_id: deviceId,
          address_id: addressToDisable.id,
        },
      },
      {
        onSettled: () => {
          setAddressToDisable(null);
        },
      },
    );
  }

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>Heartbeat</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
            <Button
              type="button"
              onClick={() =>
                heartbeatMutation.mutate({ path: { device_id: deviceId } })
              }
              disabled={heartbeatMutation.isPending}
            >
              {heartbeatMutation.isPending ? "Registering..." : "Register my IP"}
            </Button>
            {heartbeatMutation.data && (
              <span className="text-sm text-muted-foreground">
                Your current IP:{" "}
                <span className="font-mono">{heartbeatMutation.data.ip}</span>
              </span>
            )}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Add IP address</CardTitle>
        </CardHeader>
        <CardContent>
          <Form {...form}>
            <form
              onSubmit={form.handleSubmit(handleAddAddressSubmit)}
              className="flex flex-col gap-4 md:flex-row md:items-end"
            >
              <FormField
                control={form.control}
                name="ip"
                render={({ field }) => (
                  <FormItem className="flex-1">
                    <FormLabel>IP address</FormLabel>
                    <FormControl>
                      <Input
                        placeholder="192.168.1.100"
                        autoComplete="off"
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <Button
                type="submit"
                className="md:self-auto self-start"
                disabled={addAddressMutation.isPending}
              >
                {addAddressMutation.isPending ? "Adding..." : "Add IP"}
              </Button>
            </form>
          </Form>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between gap-4">
          <CardTitle>Assigned addresses</CardTitle>
          <div className="flex items-center gap-2">
            <label
              htmlFor="refresh-interval"
              className="text-sm text-muted-foreground whitespace-nowrap"
            >
              Auto-refresh
            </label>
            <select
              id="refresh-interval"
              value={refreshInterval}
              onChange={(e) => setRefreshInterval(Number(e.target.value))}
              className="rounded-md border border-input bg-background px-2 py-1 text-sm shadow-sm focus:outline-none focus:ring-1 focus:ring-ring"
            >
              {REFRESH_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
          </div>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="space-y-2">
              <div className="h-4 w-full animate-pulse rounded bg-muted" />
              <div className="h-4 w-full animate-pulse rounded bg-muted" />
              <div className="h-4 w-2/3 animate-pulse rounded bg-muted" />
            </div>
          ) : !addresses || addresses.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              No addresses assigned yet.
            </p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>IP</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Updated</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {addresses.map((address) => (
                  <TableRow key={address.id}>
                    <TableCell className="font-mono text-sm">
                      {address.ip}
                    </TableCell>
                    <TableCell>
                      <span
                        className="inline-flex items-center gap-2"
                        title={address.is_enabled ? "Active" : "Inactive"}
                      >
                        <span
                          className={cn(
                            "h-2.5 w-2.5 shrink-0 rounded-full",
                            address.is_enabled
                              ? "bg-green-500 dark:bg-green-600"
                              : "bg-red-500 dark:bg-red-600",
                          )}
                          aria-hidden
                        />
                        <span className="text-sm text-muted-foreground">
                          {address.is_enabled ? "Active" : "Inactive"}
                        </span>
                      </span>
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {format(new Date(address.updated_at), "PP p")}
                    </TableCell>
                    <TableCell className="text-right">
                      {address.is_enabled ? (
                        <Button
                          type="button"
                          variant="destructive"
                          size="sm"
                          onClick={() => setAddressToDisable(address)}
                          disabled={disableAddressMutation.isPending}
                        >
                          Disable
                        </Button>
                      ) : (
                        <Button
                          type="button"
                          size="sm"
                          onClick={() =>
                            addAddressMutation.mutate({
                              path: { device_id: deviceId },
                              body: { ip: address.ip },
                            })
                          }
                          disabled={addAddressMutation.isPending}
                        >
                          Enable
                        </Button>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <Dialog
        open={addressToDisable !== null}
        onOpenChange={(open) => {
          if (!open) {
            setAddressToDisable(null);
          }
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Disable address</DialogTitle>
            <DialogDescription>
              Disable IP{" "}
              <span className="font-mono">
                {addressToDisable?.ip ?? ""}
              </span>{" "}
              for this device? Existing connections may stop working.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => setAddressToDisable(null)}
            >
              Cancel
            </Button>
            <Button
              type="button"
              variant="destructive"
              onClick={handleConfirmDisable}
              disabled={disableAddressMutation.isPending}
            >
              {disableAddressMutation.isPending ? "Disabling..." : "Disable"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
