import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { MantineProvider } from "@mantine/core";
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
import { SettingsPage } from "./pages/settings/SettingsPage";
import { AccessLogPage } from "./pages/access-log/AccessLogPage";
import { AddressHistoryPage } from "./pages/address-history/AddressHistoryPage";
import { HostsPage } from "./pages/access/hosts/HostsPage";
import { HostGroupsPage } from "./pages/access/host-groups/HostGroupsPage";
import { UsersPage } from "./pages/access/users/UsersPage";
import { UserDetailPage } from "./pages/access/users/UserDetailPage";
import { PolicyAuditPage } from "./pages/policy-audit/PolicyAuditPage";
import { NetworkPoliciesPage } from "./pages/access/network-policies/NetworkPoliciesPage";
import { NetworkPolicyDetailPage } from "./pages/access/network-policies/NetworkPolicyDetailPage";
import { theme, cssVariablesResolver } from "./lib/theme";
import { ROUTES } from "./lib/routes";
import { DateTimePrefsProvider } from "./contexts/DateTimePrefsContext";

function App() {
  return (
    <MantineProvider theme={theme} cssVariablesResolver={cssVariablesResolver} defaultColorScheme="auto">
      <Notifications />
      <DateTimePrefsProvider>
        <DatesProvider settings={{ locale: 'en' }}>
          <BrowserRouter>
            <AuthProvider>
              <AppErrorBoundary>
                <Routes>
                  <Route path={ROUTES.login} element={<LoginPage />} />
                  <Route
                    path="/"
                    element={<Navigate to={ROUTES.dashboard} replace />}
                  />
                  <Route
                    path={ROUTES.dashboard}
                    element={
                      <ProtectedRoute>
                        <AppShell>
                          <TrafficDashboardPage />
                        </AppShell>
                      </ProtectedRoute>
                    }
                  />
                  <Route
                    path={ROUTES.devices}
                    element={
                      <ProtectedRoute>
                        <AppShell>
                          <DevicesPage />
                        </AppShell>
                      </ProtectedRoute>
                    }
                  />
                  <Route
                    path={ROUTES.userDevices}
                    element={
                      <ProtectedRoute>
                        <AppShell>
                          <UserDevicesPage />
                        </AppShell>
                      </ProtectedRoute>
                    }
                  />
                  <Route
                    path={ROUTES.settings}
                    element={
                      <ProtectedRoute>
                        <AppShell>
                          <SettingsPage />
                        </AppShell>
                      </ProtectedRoute>
                    }
                  />
                  <Route
                    path={ROUTES.accessLog}
                    element={
                      <ProtectedRoute>
                        <AppShell>
                          <AccessLogPage />
                        </AppShell>
                      </ProtectedRoute>
                    }
                  />
                  <Route
                    path={ROUTES.addressHistory}
                    element={
                      <ProtectedRoute>
                        <AppShell>
                          <AddressHistoryPage />
                        </AppShell>
                      </ProtectedRoute>
                    }
                  />
                  <Route
                    path={ROUTES.accessHosts}
                    element={
                      <ProtectedRoute>
                        <AppShell>
                          <HostsPage />
                        </AppShell>
                      </ProtectedRoute>
                    }
                  />
                  <Route
                    path={ROUTES.accessHostGroups}
                    element={
                      <ProtectedRoute>
                        <AppShell>
                          <HostGroupsPage />
                        </AppShell>
                      </ProtectedRoute>
                    }
                  />
                  <Route
                    path={ROUTES.accessUsers}
                    element={
                      <ProtectedRoute>
                        <AppShell>
                          <UsersPage />
                        </AppShell>
                      </ProtectedRoute>
                    }
                  />
                  <Route
                    path={ROUTES.accessUserDetail}
                    element={
                      <ProtectedRoute>
                        <AppShell>
                          <UserDetailPage />
                        </AppShell>
                      </ProtectedRoute>
                    }
                  />
                  <Route
                    path={ROUTES.policyAudit}
                    element={
                      <ProtectedRoute>
                        <AppShell>
                          <PolicyAuditPage />
                        </AppShell>
                      </ProtectedRoute>
                    }
                  />
                  <Route
                    path={ROUTES.accessNetworkPolicies}
                    element={
                      <ProtectedRoute>
                        <AppShell>
                          <NetworkPoliciesPage />
                        </AppShell>
                      </ProtectedRoute>
                    }
                  />
                  <Route
                    path={ROUTES.accessNetworkPolicyDetail}
                    element={
                      <ProtectedRoute>
                        <AppShell>
                          <NetworkPolicyDetailPage />
                        </AppShell>
                      </ProtectedRoute>
                    }
                  />
                  <Route path="*" element={<NotFoundPage />} />
                </Routes>
              </AppErrorBoundary>
            </AuthProvider>
          </BrowserRouter>
        </DatesProvider>
      </DateTimePrefsProvider>
    </MantineProvider>
  );
}

export default App;
