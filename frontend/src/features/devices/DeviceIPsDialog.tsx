import {useState} from "react";
import {useMutation, useQuery, useQueryClient} from "@tanstack/react-query";
import {useForm} from "react-hook-form";
import {zodResolver} from "@hookform/resolvers/zod";
import * as z from "zod";
import {api, toErrorMessage} from "@/lib/api/client";
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogHeader,
    DialogTitle,
    DialogTrigger,
} from "@/components/ui/dialog";
import {Button} from "@/components/ui/button";
import {Input} from "@/components/ui/input";
import {Badge} from "@/components/ui/badge";
import {Form, FormControl, FormField, FormItem, FormLabel, FormMessage} from "@/components/ui/form";
import {format} from "date-fns";

const ipSchema = z.object({
    ip_address: z.ipv4()
});

type DeviceIPsDialogProps = {
    deviceId: number;
    deviceName: string;
};

export function DeviceIPsDialog({deviceId, deviceName}: DeviceIPsDialogProps) {
    const queryClient = useQueryClient();
    const [open, setOpen] = useState(false);
    const [error, setError] = useState<string | null>(null);

    // Fetch IPs for this device
    const {data: ips, isLoading} = useQuery({
        queryKey: ["device-ips", deviceId],
        queryFn: async () => {
            const {data, error} = await api.GET("/devices/{id}/ips", {
                params: {path: {id: deviceId}},
            });
            if (error) throw new Error(toErrorMessage(error));
            return data ?? [];
        },
        enabled: open, // Only fetch when dialog is open
    });

    // Add IP mutation
    const addIPMutation = useMutation({
        mutationFn: async (ip_address: string) => {
            const {data, error} = await api.POST("/devices/{id}/ips", {
                params: {path: {id: deviceId}},
                body: {ip_address},
            });
            if (error) throw new Error(toErrorMessage(error));
            return data;
        },
        onSuccess: () => {
            queryClient.invalidateQueries({queryKey: ["device-ips", deviceId]});
            form.reset();
            setError(null);
        },
        onError: (err) => setError(err.message),
    });

    // Disable IP mutation
    const disableIPMutation = useMutation({
        mutationFn: async (ipId: number) => {
            const {error} = await api.PATCH("/devices/{id}/ips/{ip_id}/disable", {
                params: {path: {id: deviceId, ip_id: ipId}},
            });
            if (error) throw new Error(toErrorMessage(error));
        },
        onSuccess: () => {
            queryClient.invalidateQueries({queryKey: ["device-ips", deviceId]});
        },
    });

    const form = useForm<z.infer<typeof ipSchema>>({
        resolver: zodResolver(ipSchema),
        defaultValues: {ip_address: ""},
    });

    function onSubmit(values: z.infer<typeof ipSchema>) {
        addIPMutation.mutate(values.ip_address);
    }

    return (
        <Dialog open={open} onOpenChange={setOpen}>
            <DialogTrigger asChild>
                <Button variant="outline" size="sm">
                    View IPs
                </Button>
            </DialogTrigger>
            <DialogContent className="max-w-2xl">
                <DialogHeader>
                    <DialogTitle>IPs for {deviceName}</DialogTitle>
                    <DialogDescription>
                        Manage IPv4 addresses assigned to this device.
                    </DialogDescription>
                </DialogHeader>

                <div className="space-y-6">
                    {/* Add IP Form */}
                    <Form {...form}>
                        <form onSubmit={form.handleSubmit(onSubmit)} className="flex gap-4 items-end">
                            <FormField
                                control={form.control}
                                name="ip_address"
                                render={({field}) => (
                                    <FormItem className="flex-1">
                                        <FormLabel>Add New IP</FormLabel>
                                        <FormControl>
                                            <Input placeholder="192.168.1.100" {...field} />
                                        </FormControl>
                                        <FormMessage/>
                                    </FormItem>
                                )}
                            />
                            <Button type="submit" disabled={addIPMutation.isPending}>
                                {addIPMutation.isPending ? "Adding..." : "Add IP"}
                            </Button>
                        </form>
                    </Form>
                    {error && <p className="text-sm text-red-500">{error}</p>}

                    {/* IP List */}
                    <div className="space-y-2">
                        <h4 className="font-semibold text-sm">Assigned IPs</h4>
                        {isLoading ? (
                            <p className="text-sm text-muted-foreground">Loading...</p>
                        ) : ips?.length === 0 ? (
                            <p className="text-sm text-muted-foreground">No IPs assigned yet.</p>
                        ) : (
                            <div className="space-y-2">
                                {ips?.map((ip) => (
                                    <div
                                        key={ip.id}
                                        className="flex items-center justify-between p-3 border rounded-lg"
                                    >
                                        <div className="flex items-center gap-3">
                                            <span className="font-mono font-semibold">{ip.ip_address}</span>
                                            {ip.disabled_at && (
                                                <Badge variant="secondary">
                                                    Disabled {format(new Date(ip.disabled_at), "PP")}
                                                </Badge>
                                            )}
                                        </div>
                                        {!ip.disabled_at && (
                                            <Button
                                                variant="destructive"
                                                size="sm"
                                                onClick={() => disableIPMutation.mutate(ip.id)}
                                                disabled={disableIPMutation.isPending}
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
