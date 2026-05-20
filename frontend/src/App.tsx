import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { MantineProvider } from "@mantine/core";
import { Notifications } from "@mantine/notifications";
import { DatesProvider } from "@mantine/dates";
import { AppShell } from "./components/layout/AppShell";
import { AppErrorBoundary } from "./components/ErrorBoundary";
import { AuthProvider } from "./features/auth/AuthContext";
import { ProtectedRoute } from "./features/auth/ProtectedRoute";
import { UserDevicesPage } from "./pages/devices/UserDevicesPage";
import { UserDeviceWorkspacePage } from "./pages/devices/UserDeviceWorkspacePage";
import { TrafficDashboardPage } from "./pages/dashboard/TrafficDashboardPage";
import { LoginPage } from "./pages/login/LoginPage";
import { NotFoundPage } from "./pages/NotFoundPage";
import { SettingsPage } from "./pages/settings/SettingsPage";
import { AccessLogPage } from "./pages/access-log/AccessLogPage";
import { AddressHistoryPage } from "./pages/address-history/AddressHistoryPage";
import { DeviceProvisioningPage } from "./pages/devices/DeviceProvisioningPage";
import { HostsPage } from "./pages/access/hosts/HostsPage";
import { HostGroupsPage } from "./pages/access/host-groups/HostGroupsPage";
import { UsersPage } from "./pages/access/users/UsersPage";
import { UserDetailPage } from "./pages/access/users/UserDetailPage";
import { PolicyAuditPage } from "./pages/policy-audit/PolicyAuditPage";
import { NetworkPoliciesPage } from "./pages/access/network-policies/NetworkPoliciesPage";
import { NetworkPolicyDetailPage } from "./pages/access/network-policies/NetworkPolicyDetailPage";
import { theme } from "./lib/theme";
import { DateTimePrefsProvider } from "./contexts/DateTimePrefsContext";

function App() {
  return (
    <MantineProvider theme={theme} defaultColorScheme="auto">
      <Notifications />
      <DateTimePrefsProvider>
        <DatesProvider settings={{ locale: 'en' }}>
          <BrowserRouter>
            <AuthProvider>
              <AppErrorBoundary>
                <Routes>
                  <Route path="/login" element={<LoginPage />} />
                  <Route
                    path="/"
                    element={<Navigate to="/dashboard" replace />}
                  />
                  <Route
                    path="/dashboard"
                    element={
                      <ProtectedRoute>
                        <AppShell>
                          <TrafficDashboardPage />
                        </AppShell>
                      </ProtectedRoute>
                    }
                  />
                  <Route
                    path="/user-devices"
                    element={
                      <ProtectedRoute>
                        <AppShell>
                          <UserDevicesPage />
                        </AppShell>
                      </ProtectedRoute>
                    }
                  />
                  <Route
                    path="/user-devices/:userId"
                    element={
                      <ProtectedRoute>
                        <AppShell>
                          <UserDeviceWorkspacePage />
                        </AppShell>
                      </ProtectedRoute>
                    }
                  />
                  <Route
                    path="/settings"
                    element={
                      <ProtectedRoute>
                        <AppShell>
                          <SettingsPage />
                        </AppShell>
                      </ProtectedRoute>
                    }
                  />
                  <Route
                    path="/access-log"
                    element={
                      <ProtectedRoute>
                        <AppShell>
                          <AccessLogPage />
                        </AppShell>
                      </ProtectedRoute>
                    }
                  />
                  <Route
                    path="/address-history"
                    element={
                      <ProtectedRoute>
                        <AppShell>
                          <AddressHistoryPage />
                        </AppShell>
                      </ProtectedRoute>
                    }
                  />
                  <Route
                    path="/device-provisioning"
                    element={
                      <ProtectedRoute>
                        <AppShell>
                          <DeviceProvisioningPage />
                        </AppShell>
                      </ProtectedRoute>
                    }
                  />
                  <Route
                    path="/access/hosts"
                    element={
                      <ProtectedRoute>
                        <AppShell>
                          <HostsPage />
                        </AppShell>
                      </ProtectedRoute>
                    }
                  />
                  <Route
                    path="/access/host-groups"
                    element={
                      <ProtectedRoute>
                        <AppShell>
                          <HostGroupsPage />
                        </AppShell>
                      </ProtectedRoute>
                    }
                  />
                  <Route
                    path="/access/users"
                    element={
                      <ProtectedRoute>
                        <AppShell>
                          <UsersPage />
                        </AppShell>
                      </ProtectedRoute>
                    }
                  />
                  <Route
                    path="/access/users/:id"
                    element={
                      <ProtectedRoute>
                        <AppShell>
                          <UserDetailPage />
                        </AppShell>
                      </ProtectedRoute>
                    }
                  />
                  <Route
                    path="/policy-audit"
                    element={
                      <ProtectedRoute>
                        <AppShell>
                          <PolicyAuditPage />
                        </AppShell>
                      </ProtectedRoute>
                    }
                  />
                  <Route
                    path="/access/network-policies"
                    element={
                      <ProtectedRoute>
                        <AppShell>
                          <NetworkPoliciesPage />
                        </AppShell>
                      </ProtectedRoute>
                    }
                  />
                  <Route
                    path="/access/network-policies/:id"
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
