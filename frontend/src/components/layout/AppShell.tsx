import { useState } from "react";
import { Link, useLocation } from "react-router-dom";
import { Menu, Server, Shield, LayoutDashboard, Settings, LogOut } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Sheet, SheetContent, SheetTrigger } from "@/components/ui/sheet";
import { cn } from "@/lib/utils";
import { Separator } from "@/components/ui/separator";

interface SidebarProps extends React.HTMLAttributes<HTMLDivElement> {
    className?: string;
}

// Navigation items
const sidebarItems = [
    { name: "Dashboard", href: "/", icon: LayoutDashboard },
    { name: "Devices", href: "/devices", icon: Server },
    // Placeholder for future features
    { name: "Security Rules", href: "/rules", icon: Shield },
    { name: "Settings", href: "/settings", icon: Settings },
];

function Sidebar({ className }: SidebarProps) {
    const location = useLocation();

    return (
        <div className={cn("pb-12", className)}>
            <div className="space-y-4 py-4">
                <div className="px-3 py-2">
                    <h2 className="mb-2 px-4 text-lg font-semibold tracking-tight">
                        WallyDic
                    </h2>
                    <div className="space-y-1">
                        {sidebarItems.map((item) => (
                            <Button
                                key={item.href}
                                variant={location.pathname === item.href ? "secondary" : "ghost"}
                                className="w-full justify-start"
                                asChild
                            >
                                <Link to={item.href}>
                                    <item.icon className="mr-2 h-4 w-4" />
                                    {item.name}
                                </Link>
                            </Button>
                        ))}
                    </div>
                </div>
                <Separator />
                <div className="px-3 py-2">
                    <Button variant="ghost" className="w-full justify-start text-red-500 hover:text-red-600 hover:bg-red-50">
                        <LogOut className="mr-2 h-4 w-4" />
                        Logout
                    </Button>
                </div>
            </div>
        </div>
    );
}

export function AppShell({ children }: { children: React.ReactNode }) {
    const [open, setOpen] = useState(false);

    return (
        <div className="flex min-h-screen flex-col bg-background">
            {/* Mobile Header */}
            <header className="sticky top-0 z-50 w-full border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60 md:hidden">
                <div className="flex h-14 items-center px-4">
                    <Sheet open={open} onOpenChange={setOpen}>
                        <SheetTrigger asChild>
                            <Button variant="ghost" size="icon" className="mr-2">
                                <Menu className="h-5 w-5" />
                                <span className="sr-only">Toggle Menu</span>
                            </Button>
                        </SheetTrigger>
                        <SheetContent side="left" className="pr-0">
                            <Sidebar className="px-2" />
                        </SheetContent>
                    </Sheet>
                    <span className="font-bold">WallyDic Manager</span>
                </div>
            </header>

            <div className="flex-1 items-start md:grid md:grid-cols-[220px_minmax(0,1fr)] lg:grid-cols-[240px_minmax(0,1fr)]">
                {/* Desktop Sidebar */}
                <aside className="fixed top-0 z-30 hidden h-screen w-full shrink-0 border-r md:sticky md:block">
                    <div className="h-full py-6 pl-2 pr-6">
                        <Sidebar />
                    </div>
                </aside>

                {/* Main Content */}
                <main className="flex w-full flex-col overflow-hidden p-4 md:p-8">
                    {children}
                </main>
            </div>
        </div>
    );
}
