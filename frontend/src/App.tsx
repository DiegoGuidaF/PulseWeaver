import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { MantineProvider } from "@mantine/core";
import { Notifications } from "@mantine/notifications";
import { DatesProvider } from "@mantine/dates";
import { AppShell } from "./components/layout/AppShell";
import { AppErrorBoundary } from "./components/ErrorBoundary";
import { AuthProvider } from "./features/auth/AuthContext";
import { ProtectedRoute } from "./features/auth/ProtectedRoute";
import { DevicesPage } from "./pages/DevicesPage";
import { DeviceDetailPage } from "./pages/DeviceDetailPage";
import { TrafficDashboardPage } from "./pages/TrafficDashboardPage";
import { LoginPage } from "./pages/LoginPage";
import { NotFoundPage } from "./pages/NotFoundPage";
import { SettingsPage } from "./pages/SettingsPage";
import { AccessLogPage } from "./pages/AccessLogPage";
import { AddressHistoryPage } from "./pages/AddressHistoryPage";
import { DeviceProvisioningPage } from "./pages/DeviceProvisioningPage";
import { HostsPage } from "./pages/HostsPage";
import { UsersPage } from "./pages/UsersPage";
import { PolicyAuditPage } from "./pages/PolicyAuditPage";
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
                    path="/devices"
                    element={
                      <ProtectedRoute>
                        <AppShell>
                          <DevicesPage />
                        </AppShell>
                      </ProtectedRoute>
                    }
                  />
                  <Route
                    path="/devices/:deviceId"
                    element={
                      <ProtectedRoute>
                        <AppShell>
                          <DeviceDetailPage />
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
                    path="/hosts"
                    element={
                      <ProtectedRoute>
                        <AppShell>
                          <HostsPage />
                        </AppShell>
                      </ProtectedRoute>
                    }
                  />
                  <Route
                    path="/users"
                    element={
                      <ProtectedRoute>
                        <AppShell>
                          <UsersPage />
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
