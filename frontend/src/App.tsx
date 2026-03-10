import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { Toaster } from "sonner";
import { AppShell } from "./components/layout/AppShell";
import { ThemeProvider } from "./components/theme-provider";
import { AppErrorBoundary } from "./components/ErrorBoundary";
import { AuthProvider } from "./features/auth/AuthContext";
import { ProtectedRoute } from "./features/auth/ProtectedRoute";
import { DashboardPage } from "./pages/DashboardPage";
import { DeviceDetailPage } from "./pages/DeviceDetailPage";
import { LoginPage } from "./pages/LoginPage";
import { NotFoundPage } from "./pages/NotFoundPage";
import { SettingsPage } from "./pages/SettingsPage";

function App() {
  return (
    <ThemeProvider defaultTheme="dark" storageKey="vite-ui-theme">
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
              <Route path="*" element={<NotFoundPage />} />
            </Routes>
          </AppErrorBoundary>
          <Toaster />
        </AuthProvider>
      </BrowserRouter>
    </ThemeProvider>
  );
}

export default App;
