import { useQuery } from "@tanstack/react-query";
import { api, toErrorMessage } from "@/lib/api/client";
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { format } from "date-fns";
import {DeviceIPsDialog} from "@/features/devices/DeviceIPsDialog.tsx"; // You might need: npm i date-fns

export function DeviceList() {
    const { data: devices, isLoading, error } = useQuery({
        queryKey: ["devices"],
        queryFn: async () => {
            const { data, error } = await api.GET("/devices");
            if (error) throw new Error(toErrorMessage(error));
            return data ?? [];
        },
    });

    if (isLoading) return <div className="p-4">Loading devices...</div>;
    if (error) return <div className="p-4 text-red-500">Error: {error.message}</div>;

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
                            <TableHead>Created At</TableHead>
                            <TableHead className="text-right">Actions</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {devices?.map((device) => (
                            <TableRow key={device.id}>
                                <TableCell className="font-medium">{device.name}</TableCell>
                                <TableCell className="font-mono text-xs">{device.id}</TableCell>
                                <TableCell>{format(new Date(device.created_at!), "PP p")}</TableCell>
                                <TableCell className="text-right">
                                    <DeviceIPsDialog deviceId={device.id} deviceName={device.name} />
                                </TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            </CardContent>
        </Card>
    );
}
