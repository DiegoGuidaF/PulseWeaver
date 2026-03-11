import { useState } from "react";
import { Link, useLocation } from "react-router-dom";
import { Menu, Server, LogOut, Settings } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Sheet, SheetContent, SheetTrigger } from "@/components/ui/sheet";
import { cn } from "@/lib/utils";
import { Separator } from "@/components/ui/separator";
import { ModeToggle } from "@/components/mode-toggle.tsx";
import { useLogout } from "@/features/auth/hooks/useLogout";
import { useAuth } from "@/features/auth/AuthContext";
import { useAutoHeartbeat } from "@/features/devices/hooks/useAutoHeartbeat";

interface SidebarProps extends React.HTMLAttributes<HTMLDivElement> {
  className?: string;
  clientIp?: string | null;
  activeDeviceId?: number | null;
}

// Navigation items
const sidebarItems = [
  { name: "Devices", href: "/devices", icon: Server },
  { name: "Settings", href: "/settings", icon: Settings },
];

function Sidebar({ className, clientIp, activeDeviceId }: SidebarProps) {
  const location = useLocation();
  const logoutMutation = useLogout();
  const { user } = useAuth();

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
                variant={
                  location.pathname.startsWith(item.href)
                    ? "secondary"
                    : "ghost"
                }
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
        {user && (
          <div className="px-3 py-2">
            <div className="mb-2 px-4 text-sm text-muted-foreground">
              {user.display_name || user.username}
            </div>
          </div>
        )}
        {activeDeviceId && clientIp && (
          <div className="px-3 py-2">
            <p className="px-4 text-xs text-muted-foreground flex items-center gap-1.5">
              <span className="inline-block w-2 h-2 rounded-full bg-green-500 shrink-0" />
              <span className="font-mono">{clientIp}</span>
            </p>
          </div>
        )}
        <div className="px-3 py-2">
          <Button
            variant="ghost"
            className="w-full justify-start text-red-500 hover:bg-red-500/10 hover:text-red-600"
            onClick={() => logoutMutation.mutate({})}
            disabled={logoutMutation.isPending}
          >
            <LogOut className="mr-2 h-4 w-4" />
            {logoutMutation.isPending ? "Logging out..." : "Logout"}
          </Button>
        </div>
      </div>
    </div>
  );
}

export function AppShell({ children }: { children: React.ReactNode }) {
  const [open, setOpen] = useState(false);
  const { clientIp, activeDeviceId } = useAutoHeartbeat();

  return (
    <div className="bg-background flex min-h-screen flex-col">
      {/* 1. Mobile Header (Visible < md) */}
      <header className="bg-background sticky top-0 z-50 flex h-14 items-center gap-4 border-b px-4 md:hidden">
        <Sheet open={open} onOpenChange={setOpen}>
          <SheetTrigger asChild>
            <Button variant="ghost" size="icon" className="mr-2">
              <Menu className="h-5 w-5" />
              <span className="sr-only">Toggle Menu</span>
            </Button>
          </SheetTrigger>
          <SheetContent side="left" className="pr-0">
            <Sidebar
              className="px-2"
              clientIp={clientIp}
              activeDeviceId={activeDeviceId}
            />
          </SheetContent>
        </Sheet>
        <span className="flex-1 font-bold">WallyDic</span>

        {/* Mobile Theme Toggle */}
        <ModeToggle />
      </header>

      <div className="flex-1 items-start md:grid md:grid-cols-[220px_minmax(0,1fr)] lg:grid-cols-[240px_minmax(0,1fr)]">
        {/* 2. Desktop Sidebar */}
        <aside className="fixed top-0 z-30 hidden h-screen w-full shrink-0 border-r md:sticky md:block">
          <div className="flex h-full flex-col py-6 pr-6 pl-2">
            <Sidebar
              className="flex-1"
              clientIp={clientIp}
              activeDeviceId={activeDeviceId}
            />

            {/* Optional: Put theme toggle at bottom of sidebar on desktop?
                            Or keep it in a top bar. Let's do a top bar for content area.
                        */}
          </div>
        </aside>

        {/* 3. Main Content Area */}
        <div className="flex min-h-screen flex-col">
          {/* Desktop Header (Visible >= md) */}
          <header className="bg-background hidden h-14 items-center justify-end gap-4 border-b px-6 md:flex">
            {/* This header sits above the main content but to the right of sidebar */}
            <ModeToggle />
          </header>

          <main className="w-full flex-1 p-4 md:p-8">{children}</main>
        </div>
      </div>
    </div>
  );
}
