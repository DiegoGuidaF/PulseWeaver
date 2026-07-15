import { createBrowserRouter, Navigate, Outlet, RouterProvider } from "react-router-dom";
import { MantineProvider } from "@mantine/core";
import { ModalsProvider } from "@mantine/modals";
import { Notifications } from "@mantine/notifications";
import { DatesProvider } from "@mantine/dates";
import { AppShell } from "./components/layout/AppShell";
import { AppErrorBoundary } from "./components/ErrorBoundary";
import { AuthProvider } from "./features/auth/AuthContext";
import { ProtectedRoute } from "./features/auth/ProtectedRoute";
import { DevicesPage } from "./pages/devices/DevicesPage";
import { UserDevicesPage } from "./pages/devices/UserDevicesPage";
import { TrafficDashboardPage } from "./pages/dashboard/TrafficDashboardPage";
import { LoginPage } from "./pages/login/LoginPage";
import { NotFoundPage } from "./pages/NotFoundPage";
import { AccountPage } from "./pages/account/AccountPage";
import { AccessLogPage } from "./pages/access-log/AccessLogPage";
import { AddressHistoryPage } from "./pages/address-history/AddressHistoryPage";
import { HostsPage } from "./pages/access/hosts/HostsPage";
import { HostGroupsPage } from "./pages/access/host-groups/HostGroupsPage";
import { UsersPage } from "./pages/access/users/UsersPage";
import { UserDetailPage } from "./pages/access/users/UserDetailPage";
import { PolicyAuditPage } from "./pages/policy-audit/PolicyAuditPage";
import { NetworkPoliciesPage } from "./pages/access/network-policies/NetworkPoliciesPage";
import { NetworkPolicyDetailPage } from "./pages/access/network-policies/NetworkPolicyDetailPage";
import { AnomaliesPage } from "./pages/anomalies/AnomaliesPage";
import { theme, cssVariablesResolver } from "./lib/theme";
import { ROUTES } from "./lib/routes";
import { DateTimePrefsProvider } from "./contexts/DateTimePrefsContext";

/**
 * Renders the providers that need to live inside the router around an
 * `<Outlet/>`. Using a layout route (rather than wrapping `<RouterProvider>`)
 * is what lets pages call `useBlocker` to guard unsaved changes.
 */
function RootLayout() {
  return (
    <AuthProvider>
      <AppErrorBoundary>
        <Outlet />
      </AppErrorBoundary>
    </AuthProvider>
  );
}

function protectedPage(page: React.ReactNode) {
  return (
    <ProtectedRoute>
      <AppShell>{page}</AppShell>
    </ProtectedRoute>
  );
}

const router = createBrowserRouter([
  {
    element: <RootLayout />,
    children: [
      { path: ROUTES.login, element: <LoginPage /> },
      { path: "/", element: <Navigate to={ROUTES.dashboard} replace /> },
      { path: ROUTES.dashboard, element: protectedPage(<TrafficDashboardPage />) },
      { path: ROUTES.devices, element: protectedPage(<DevicesPage />) },
      { path: ROUTES.userDevices, element: protectedPage(<UserDevicesPage />) },
      { path: ROUTES.userDevicesNew, element: protectedPage(<UserDevicesPage createMode />) },
      { path: ROUTES.account, element: protectedPage(<AccountPage />) },
      { path: ROUTES.accessLog, element: protectedPage(<AccessLogPage />) },
      { path: ROUTES.addressHistory, element: protectedPage(<AddressHistoryPage />) },
      { path: ROUTES.accessHosts, element: protectedPage(<HostsPage />) },
      { path: ROUTES.accessHostGroups, element: protectedPage(<HostGroupsPage />) },
      { path: ROUTES.accessUsers, element: protectedPage(<UsersPage />) },
      { path: ROUTES.accessUserDetail, element: protectedPage(<UserDetailPage />) },
      { path: ROUTES.policyAudit, element: protectedPage(<PolicyAuditPage />) },
      { path: ROUTES.accessNetworkPolicies, element: protectedPage(<NetworkPoliciesPage />) },
      { path: ROUTES.accessNetworkPolicyDetail, element: protectedPage(<NetworkPolicyDetailPage />) },
      { path: ROUTES.anomalies, element: protectedPage(<AnomaliesPage />) },
      { path: "*", element: <NotFoundPage /> },
    ],
  },
]);

function App() {
  return (
    <MantineProvider theme={theme} cssVariablesResolver={cssVariablesResolver} defaultColorScheme="auto">
      <Notifications />
      <DateTimePrefsProvider>
        <DatesProvider settings={{ locale: 'en' }}>
          <ModalsProvider>
            <RouterProvider router={router} />
          </ModalsProvider>
        </DatesProvider>
      </DateTimePrefsProvider>
    </MantineProvider>
  );
}

export default App;
