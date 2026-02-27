import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { format } from "date-fns";
import { useDeviceAddresses } from "@/features/devices/hooks/useDeviceAddresses";
import { useDeviceAddressLeaseRule } from "@/features/devices/hooks/useDeviceAddressLeaseRule";
import { usePutDeviceAddressLeaseRule } from "@/features/devices/hooks/usePutDeviceAddressLeaseRule";
import { useDisableDeviceAddressLeaseRule } from "@/features/devices/hooks/useDisableDeviceAddressLeaseRule";
import { useAddDeviceAddress } from "@/features/devices/hooks/useAddDeviceAddress";
import { useDisableDeviceAddress } from "@/features/devices/hooks/useDisableDeviceAddress";
import { zAddAddressRequest } from "@/lib/api/zod.gen";
import { z } from "zod";

const addressSchema = zAddAddressRequest;

const TTL_UNITS = ["seconds", "minutes", "days"] as const;
const SECONDS_PER_MINUTE = 60;
const SECONDS_PER_DAY = 86400;

function toSeconds(value: number, unit: (typeof TTL_UNITS)[number]): number {
  switch (unit) {
    case "seconds":
      return value;
    case "minutes":
      return value * SECONDS_PER_MINUTE;
    case "days":
      return value * SECONDS_PER_DAY;
  }
}

const leaseRuleFormSchema = z.object({
  value: z.coerce.number().int("Must be a whole number").min(1, "Minimum is 1"),
  unit: z.enum(TTL_UNITS),
});

type LeaseRuleFormValues = z.infer<typeof leaseRuleFormSchema>;

type DeviceAddressesDialogProps = {
  deviceId: number;
  deviceName: string;
};

export function DeviceAddressesDialog({
  deviceId,
  deviceName,
}: DeviceAddressesDialogProps) {
  const [open, setOpen] = useState(false);

  const { data: addresses, isLoading } = useDeviceAddresses(deviceId, open);
  const {
    data: rule,
    isLoading: ruleLoading,
  } = useDeviceAddressLeaseRule(deviceId, open);
  const putRuleMutation = usePutDeviceAddressLeaseRule(deviceId);
  const disableRuleMutation = useDisableDeviceAddressLeaseRule(deviceId);

  const form = useForm<z.infer<typeof addressSchema>>({
    resolver: zodResolver(addressSchema),
    defaultValues: { ip: "" },
  });

  const addAddressMutation = useAddDeviceAddress(deviceId, {
    onSuccess: () => form.reset(),
  });

  const disableAddressMutation = useDisableDeviceAddress();

  const leaseRuleForm = useForm<LeaseRuleFormValues>({
    resolver: zodResolver(leaseRuleFormSchema),
    defaultValues: { value: 3600, unit: "seconds" },
  });

  const [editingRule, setEditingRule] = useState(false);

  function onSubmit(values: z.infer<typeof addressSchema>) {
    addAddressMutation.mutate({
      path: { device_id: deviceId },
      body: { ip: values.ip },
    });
  }

  function onLeaseRuleSubmit(values: LeaseRuleFormValues) {
    putRuleMutation.mutate({
      body: { ttl_seconds: toSeconds(values.value, values.unit) },
    });
    leaseRuleForm.reset();
    setEditingRule(false);
  }

  function handleEnableRule() {
    if (rule) {
      putRuleMutation.mutate({ body: { ttl_seconds: rule.ttl_seconds } });
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="outline" size="sm">
          View Addresses
        </Button>
      </DialogTrigger>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>Addresses for {deviceName}</DialogTitle>
          <DialogDescription>
            Manage IP addresses assigned to this device.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6">
          {/* Address lease rule */}
          <div className="space-y-2">
            <h4 className="text-sm font-semibold">Address lease rule</h4>
            {ruleLoading ? (
              <p className="text-muted-foreground text-sm">Loading...</p>
            ) : rule === null ? (
              <Form {...leaseRuleForm}>
                <form
                  onSubmit={leaseRuleForm.handleSubmit(onLeaseRuleSubmit)}
                  className="flex flex-wrap items-end gap-4"
                >
                  <FormField
                    control={leaseRuleForm.control}
                    name="value"
                    render={({ field }) => (
                      <FormItem className="w-32">
                        <FormLabel>TTL</FormLabel>
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
                      <FormItem className="w-28">
                        <FormLabel>Unit</FormLabel>
                        <FormControl>
                          <select
                            className="border-input focus-visible:ring-ring flex h-9 w-full rounded-md border bg-transparent px-3 py-1 text-base shadow-sm transition-colors focus-visible:ring-1 focus-visible:outline-none md:text-sm"
                            {...field}
                          >
                            {TTL_UNITS.map((u) => (
                              <option key={u} value={u}>
                                {u}
                              </option>
                            ))}
                          </select>
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <Button type="submit" disabled={putRuleMutation.isPending}>
                    {putRuleMutation.isPending ? "Creating..." : "Create"}
                  </Button>
                </form>
              </Form>
            ) : editingRule ? (
              <Form {...leaseRuleForm}>
                <form
                  onSubmit={leaseRuleForm.handleSubmit(onLeaseRuleSubmit)}
                  className="flex flex-wrap items-end gap-4"
                >
                  <FormField
                    control={leaseRuleForm.control}
                    name="value"
                    render={({ field }) => (
                      <FormItem className="w-32">
                        <FormLabel>TTL</FormLabel>
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
                      <FormItem className="w-28">
                        <FormLabel>Unit</FormLabel>
                        <FormControl>
                          <select
                            className="border-input focus-visible:ring-ring flex h-9 w-full rounded-md border bg-transparent px-3 py-1 text-base shadow-sm transition-colors focus-visible:ring-1 focus-visible:outline-none md:text-sm"
                            {...field}
                          >
                            {TTL_UNITS.map((u) => (
                              <option key={u} value={u}>
                                {u}
                              </option>
                            ))}
                          </select>
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <Button type="submit" disabled={putRuleMutation.isPending}>
                    {putRuleMutation.isPending ? "Saving..." : "Save"}
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => {
                      setEditingRule(false);
                      leaseRuleForm.reset({
                        value: rule.ttl_seconds,
                        unit: "seconds",
                      });
                    }}
                  >
                    Cancel
                  </Button>
                </form>
              </Form>
            ) : (
              <div className="space-y-2 rounded-lg border p-3">
                <p className="text-muted-foreground text-sm">
                  TTL: {rule.ttl_seconds} seconds · Enabled:{" "}
                  {rule.enabled ? "Yes" : "No"}
                </p>
                <div className="flex gap-2">
                  {rule.enabled ? (
                    <>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => {
                          setEditingRule(true);
                          leaseRuleForm.reset({
                            value: rule.ttl_seconds,
                            unit: "seconds",
                          });
                        }}
                      >
                        Edit
                      </Button>
                      <Button
                        variant="destructive"
                        size="sm"
                        onClick={() => disableRuleMutation.mutate({})}
                        disabled={disableRuleMutation.isPending}
                      >
                        Disable
                      </Button>
                    </>
                  ) : (
                    <Button
                      size="sm"
                      onClick={handleEnableRule}
                      disabled={putRuleMutation.isPending}
                    >
                      Enable
                    </Button>
                  )}
                </div>
              </div>
            )}
          </div>

          {/* Add Address Form */}
          <Form {...form}>
            <form
              onSubmit={form.handleSubmit(onSubmit)}
              className="flex items-end gap-4"
            >
              <FormField
                control={form.control}
                name="ip"
                render={({ field }) => (
                  <FormItem className="flex-1">
                    <FormLabel>Add New Address</FormLabel>
                    <FormControl>
                      <Input placeholder="192.168.1.100" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <Button type="submit" disabled={addAddressMutation.isPending}>
                {addAddressMutation.isPending ? "Adding..." : "Add Address"}
              </Button>
            </form>
          </Form>

          {/* Address List */}
          <div className="space-y-2">
            <h4 className="text-sm font-semibold">Assigned Addresses</h4>
            {isLoading ? (
              <p className="text-muted-foreground text-sm">Loading...</p>
            ) : addresses?.length === 0 ? (
              <p className="text-muted-foreground text-sm">
                No addresses assigned yet.
              </p>
            ) : (
              <div className="space-y-2">
                {addresses?.map((address) => (
                  <div
                    key={address.id}
                    className="flex items-center justify-between rounded-lg border p-3"
                  >
                    <div className="flex items-center gap-3">
                      <span className="font-mono font-semibold">
                        {address.ip}
                      </span>
                      {!address.status && (
                        <Badge variant="secondary">
                          Disabled {format(new Date(address.updated_at), "PP")}
                        </Badge>
                      )}
                    </div>
                    {address.status && (
                      <Button
                        variant="destructive"
                        size="sm"
                        onClick={() =>
                          disableAddressMutation.mutate({
                            path: {
                              device_id: deviceId,
                              address_id: address.id,
                            },
                          })
                        }
                        disabled={disableAddressMutation.isPending}
                      >
                        Disable
                      </Button>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
