import { BrowserRouter, Routes, Route } from "react-router-dom";
import { DeviceList } from "./features/devices/DeviceList";
import { CreateDeviceForm } from "./features/devices/CreateDeviceForm";
import {Toaster} from "sonner";

function Dashboard() {
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

function NotFound() {
    return (
        <div className="flex flex-col items-center justify-center h-screen space-y-4">
            <h1 className="text-4xl font-bold">404</h1>
            <p className="text-muted-foreground">Page not found</p>
            <a href="/" className="text-blue-500 hover:underline">Go Home</a>
        </div>
    );
}

function App() {
    return (
        <BrowserRouter>
            <Routes>
                <Route path="/" element={<Dashboard />} />
                {/* Redirect /devices to home for now, or make it a separate page */}
                <Route path="/devices" element={<Dashboard />} />

                {/* Catch-all route for 404s */}
                <Route path="*" element={<NotFound />} />
            </Routes>
            <Toaster />
        </BrowserRouter>
    );
}

export default App;
