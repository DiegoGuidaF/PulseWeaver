import { CreateDeviceForm } from "@/features/devices/CreateDeviceForm";
import { DeviceList } from "@/features/devices/DeviceList";

export function DashboardPage() {
  return (
    <div className="w-full max-w-5xl space-y-8">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">WallyDic Manager</h1>
        <p className="text-muted-foreground">
          Manage your networked devices and addresses.
        </p>
      </div>
      <div className="grid gap-8">
        <CreateDeviceForm />
        <DeviceList />
      </div>
    </div>
  );
}
