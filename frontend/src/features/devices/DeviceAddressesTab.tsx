import { useState } from "react";
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

const addressSchema = zAddAddressRequest;

interface DeviceAddressesTabProps {
  deviceId: number;
}

export function DeviceAddressesTab({ deviceId }: DeviceAddressesTabProps) {
  const { data: addresses, isLoading } = useDeviceAddresses(deviceId);
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
