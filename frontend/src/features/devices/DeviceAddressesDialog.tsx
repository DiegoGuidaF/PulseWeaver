import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
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
import { useAddDeviceAddress } from "@/features/devices/hooks/useAddDeviceAddress";
import { useDisableDeviceAddress } from "@/features/devices/hooks/useDisableDeviceAddress";

const addressSchema = z.object({
  ip: z.ipv4({ message: "Must be a valid IPv4 address" }),
});

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

  const form = useForm<z.infer<typeof addressSchema>>({
    resolver: zodResolver(addressSchema),
    defaultValues: { ip: "" },
  });

  const addAddressMutation = useAddDeviceAddress(deviceId, {
    onSuccess: () => form.reset(),
  });

  const disableAddressMutation = useDisableDeviceAddress(deviceId);

  function onSubmit(values: z.infer<typeof addressSchema>) {
    addAddressMutation.mutate(values.ip);
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
                          disableAddressMutation.mutate(address.id)
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
