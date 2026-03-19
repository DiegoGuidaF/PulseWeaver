import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { MantineProvider } from "@mantine/core";
import { Notifications } from "@mantine/notifications";
import { DatesProvider } from "@mantine/dates";
import { AppShell } from "./components/layout/AppShell";
import { AppErrorBoundary } from "./components/ErrorBoundary";
import { AuthProvider } from "./features/auth/AuthContext";
import { ProtectedRoute } from "./features/auth/ProtectedRoute";
import { DashboardPage } from "./pages/DashboardPage";
import { DeviceDetailPage } from "./pages/DeviceDetailPage";
import { LoginPage } from "./pages/LoginPage";
import { NotFoundPage } from "./pages/NotFoundPage";
import { SettingsPage } from "./pages/SettingsPage";
import { RequestAuditLogPage } from "./pages/RequestAuditLogPage";
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
                    element={<Navigate to="/devices" replace />}
                  />
                  <Route
                    path="/devices"
                    element={
                      <ProtectedRoute>
                        <AppShell>
                          <DashboardPage />
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
                    path="/request-audit-log"
                    element={
                      <ProtectedRoute>
                        <AppShell>
                          <RequestAuditLogPage />
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
