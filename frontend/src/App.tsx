import { DeviceList } from "./features/devices/DeviceList";
import { CreateDeviceForm } from "./features/devices/CreateDeviceForm";

function App() {
    return (
        <div className="container mx-auto py-10 max-w-4xl space-y-8">
            <div>
                <h1 className="text-3xl font-bold tracking-tight">WallyDic Manager</h1>
                <p className="text-muted-foreground">Manage your networked devices and IPs.</p>
            </div>

            <div className="grid gap-8">
                <CreateDeviceForm />
                <DeviceList />
            </div>
        </div>
    );
}

export default App;
