import { useEffect, useState } from "react";
import { Link, Navigate, useParams } from "react-router-dom";
import { ChevronLeft } from "lucide-react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Skeleton } from "@/components/ui/skeleton";
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { format } from "date-fns";
import type { Address } from "@/lib/api";
import { cn } from "@/lib/utils";
import { useDevice } from "@/features/devices/hooks/useDevice";
import { useDeviceAddresses } from "@/features/devices/hooks/useDeviceAddresses";
import { useAddDeviceAddress } from "@/features/devices/hooks/useAddDeviceAddress";
import { useDisableDeviceAddress } from "@/features/devices/hooks/useDisableDeviceAddress";
import { useDeviceAddressLeaseRule } from "@/features/devices/hooks/useDeviceAddressLeaseRule";
import { usePutDeviceAddressLeaseRule } from "@/features/devices/hooks/usePutDeviceAddressLeaseRule";
import { useDisableDeviceAddressLeaseRule } from "@/features/devices/hooks/useDisableDeviceAddressLeaseRule";
import { toErrorMessage } from "@/lib/api-client";
import { zAddAddressRequest } from "@/lib/api/zod.gen";

type DeviceDetailRouteParams = {
  deviceId?: string;
};

const TTL_UNITS = ["seconds", "minutes", "days"] as const;
const SECONDS_PER_MINUTE = 60;
const SECONDS_PER_DAY = 86400;

type TtlUnit = (typeof TTL_UNITS)[number];

function toSeconds(value: number, unit: TtlUnit): number {
  switch (unit) {
    case "seconds":
      return value;
    case "minutes":
      return value * SECONDS_PER_MINUTE;
    case "days":
      return value * SECONDS_PER_DAY;
  }
}

function fromSeconds(ttlSeconds: number): { value: number; unit: TtlUnit } {
  if (ttlSeconds % SECONDS_PER_DAY === 0) {
    return { value: ttlSeconds / SECONDS_PER_DAY, unit: "days" };
  }
  if (ttlSeconds % SECONDS_PER_MINUTE === 0) {
    return { value: ttlSeconds / SECONDS_PER_MINUTE, unit: "minutes" };
  }
  return { value: ttlSeconds, unit: "seconds" };
}

function formatTtlLabel(ttlSeconds: number): string {
  if (ttlSeconds % SECONDS_PER_DAY === 0) {
    const days = ttlSeconds / SECONDS_PER_DAY;
    return days === 1 ? "1 day" : `${days} days`;
  }

  if (ttlSeconds % SECONDS_PER_MINUTE === 0) {
    const minutes = ttlSeconds / SECONDS_PER_MINUTE;
    if (minutes % 60 === 0) {
      const hours = minutes / 60;
      return hours === 1 ? "1 hour" : `${hours} hours`;
    }
    return minutes === 1 ? "1 minute" : `${minutes} minutes`;
  }

  return ttlSeconds === 1 ? "1 second" : `${ttlSeconds} seconds`;
}

const addressSchema = zAddAddressRequest;

const leaseRuleFormSchema = z.object({
  value: z
    .coerce.number()
    .int("Must be a whole number")
    .min(1, "Minimum is 1"),
  unit: z.enum(TTL_UNITS),
});

type LeaseRuleFormValues = z.infer<typeof leaseRuleFormSchema>;

export function DeviceDetailPage() {
  const params = useParams<DeviceDetailRouteParams>();
  const deviceIdParam = params.deviceId;
  const deviceId = deviceIdParam
    ? Number.parseInt(deviceIdParam, 10)
    : Number.NaN;

  const { data: device, isLoading, isError, error } = useDevice(deviceId);

  if (!deviceIdParam || Number.isNaN(deviceId)) {
    return <Navigate to="/devices" replace />;
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
          <DeviceSettingsRulesTab deviceId={deviceId} />
        </TabsContent>
      </Tabs>
    </div>
  );
}

interface DeviceAddressesTabProps {
  deviceId: number;
}

function DeviceAddressesTab({ deviceId }: DeviceAddressesTabProps) {
  const {
    data: addresses,
    isLoading,
  } = useDeviceAddresses(deviceId, true);
  const form = useForm<z.infer<typeof addressSchema>>({
    resolver: zodResolver(addressSchema),
    defaultValues: { ip: "" },
  });
  const addAddressMutation = useAddDeviceAddress(deviceId, {
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
        <CardHeader>
          <CardTitle>Assigned addresses</CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="space-y-2">
              <Skeleton className="h-4 w-full" />
              <Skeleton className="h-4 w-full" />
              <Skeleton className="h-4 w-2/3" />
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
                        title={address.status ? "Active" : "Inactive"}
                      >
                        <span
                          className={cn(
                            "h-2.5 w-2.5 shrink-0 rounded-full",
                            address.status
                              ? "bg-green-500 dark:bg-green-600"
                              : "bg-red-500 dark:bg-red-600",
                          )}
                          aria-hidden
                        />
                        <span className="text-sm text-muted-foreground">
                          {address.status ? "Active" : "Inactive"}
                        </span>
                      </span>
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {format(new Date(address.updated_at), "PP p")}
                    </TableCell>
                    <TableCell className="text-right">
                      {address.status ? (
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

interface DeviceSettingsRulesTabProps {
  deviceId: number;
}

function DeviceSettingsRulesTab({ deviceId }: DeviceSettingsRulesTabProps) {
  const {
    data: rule,
    isLoading,
    isError,
    error,
  } = useDeviceAddressLeaseRule(deviceId, true);
  const putRuleMutation = usePutDeviceAddressLeaseRule(deviceId);
  const disableRuleMutation = useDisableDeviceAddressLeaseRule(deviceId);

  const leaseRuleForm = useForm<LeaseRuleFormValues>({
    resolver: zodResolver(leaseRuleFormSchema),
    defaultValues: { value: 5, unit: "minutes" },
  });
  const [editing, setEditing] = useState(false);

  function handleLeaseRuleSubmit(values: LeaseRuleFormValues) {
    putRuleMutation.mutate({
      body: { ttl_seconds: toSeconds(values.value, values.unit) },
    });
    setEditing(false);
  }

  function handleStartEditing() {
    if (!rule) return;
    const initial = fromSeconds(rule.ttl_seconds);
    leaseRuleForm.reset(initial);
    setEditing(true);
  }

  const isOn = Boolean(rule && rule.enabled);

  useEffect(() => {
    if (!rule || isOn) {
      return;
    }
    const initial = fromSeconds(rule.ttl_seconds);
    leaseRuleForm.reset(initial);
  }, [isOn, leaseRuleForm, rule]);

  const statusLabel = isOn ? "Enabled" : "Disabled";

  const ttlLabel =
    rule && rule.ttl_seconds ? formatTtlLabel(rule.ttl_seconds) : null;

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle>Auto-expiry rule</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {isLoading ? (
            <div className="space-y-2">
              <Skeleton className="h-4 w-40" />
              <Skeleton className="h-4 w-64" />
            </div>
          ) : isError ? (
            <p className="text-sm text-red-500">
              Error loading rule: {toErrorMessage(error)}
            </p>
          ) : !isOn ? (
            <>
              <p className="text-sm text-muted-foreground">
                Auto-expiry is currently{" "}
                <span className="font-medium text-foreground">
                  disabled
                </span>
                . Turn it on to automatically revoke stale
                addresses.
              </p>
              <Form {...leaseRuleForm}>
                <form
                  onSubmit={leaseRuleForm.handleSubmit(
                    handleLeaseRuleSubmit,
                  )}
                  className="flex flex-wrap items-end gap-4"
                >
                  <FormField
                    control={leaseRuleForm.control}
                    name="value"
                    render={({ field }) => (
                      <FormItem className="w-32">
                        <FormLabel>Expires after</FormLabel>
                        <FormControl>
                          <Input
                            type="number"
                            min={1}
                            step={1}
                            placeholder="1"
                            {...field}
                            onChange={(e) =>
                              field.onChange(
                                e.target.value === ""
                                  ? undefined
                                  : Number(e.target.value),
                              )
                            }
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={leaseRuleForm.control}
                    name="unit"
                    render={({ field }) => (
                      <FormItem className="w-32">
                        <FormLabel>Unit</FormLabel>
                        <FormControl>
                          <select
                            className="border-input focus-visible:ring-ring flex h-9 w-full rounded-md border bg-background px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1"
                            {...field}
                          >
                            {TTL_UNITS.map((unit) => (
                              <option key={unit} value={unit}>
                                {unit}
                              </option>
                            ))}
                          </select>
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <Button
                    type="submit"
                    disabled={putRuleMutation.isPending}
                  >
                    {putRuleMutation.isPending
                      ? "Saving..."
                      : "Enable auto-expiry"}
                  </Button>
                </form>
              </Form>
            </>
          ) : (
            <>
              <div className="space-y-1">
                <p className="text-sm">
                  Status:{" "}
                  <span className="font-medium">
                    {statusLabel}
                  </span>
                </p>
                {ttlLabel && (
                  <p className="text-sm text-muted-foreground">
                    Addresses will automatically expire after{" "}
                    <span className="font-medium">
                      {ttlLabel}
                    </span>
                    .
                  </p>
                )}
              </div>

              {editing ? (
                <Form {...leaseRuleForm}>
                  <form
                    onSubmit={leaseRuleForm.handleSubmit(
                      handleLeaseRuleSubmit,
                    )}
                    className="flex flex-wrap items-end gap-4"
                  >
                    <FormField
                      control={leaseRuleForm.control}
                      name="value"
                      render={({ field }) => (
                        <FormItem className="w-32">
                          <FormLabel>Expires after</FormLabel>
                          <FormControl>
                            <Input
                              type="number"
                              min={1}
                              step={1}
                              placeholder="1"
                              {...field}
                              onChange={(e) =>
                                field.onChange(
                                  e.target.value === ""
                                    ? undefined
                                    : Number(
                                        e.target.value,
                                      ),
                                )
                              }
                            />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={leaseRuleForm.control}
                      name="unit"
                      render={({ field }) => (
                        <FormItem className="w-32">
                          <FormLabel>Unit</FormLabel>
                          <FormControl>
                            <select
                              className="border-input focus-visible:ring-ring flex h-9 w-full rounded-md border bg-background px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1"
                              {...field}
                            >
                              {TTL_UNITS.map((unit) => (
                                <option
                                  key={unit}
                                  value={unit}
                                >
                                  {unit}
                                </option>
                              ))}
                            </select>
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                    <Button
                      type="submit"
                      disabled={putRuleMutation.isPending}
                    >
                      {putRuleMutation.isPending
                        ? "Saving..."
                        : "Save"}
                    </Button>
                    <Button
                      type="button"
                      variant="outline"
                      onClick={() => setEditing(false)}
                    >
                      Cancel
                    </Button>
                  </form>
                </Form>
              ) : (
                <div className="flex flex-wrap gap-2">
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={handleStartEditing}
                  >
                    Change TTL
                  </Button>
                  {rule?.enabled ? (
                    <Button
                      type="button"
                      variant="destructive"
                      size="sm"
                      onClick={() =>
                        disableRuleMutation.mutate({})
                      }
                      disabled={disableRuleMutation.isPending}
                    >
                      Turn off auto-expiry
                    </Button>
                  ) : null}
                </div>
              )}
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}


