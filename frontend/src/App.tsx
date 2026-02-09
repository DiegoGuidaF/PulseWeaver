import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { Toaster } from "sonner";
import { AppShell } from "./components/layout/AppShell";
import { ThemeProvider } from "./components/theme-provider";
import { AppErrorBoundary } from "./components/ErrorBoundary";
import { DashboardPage } from "./pages/DashboardPage";
import { NotFoundPage } from "./pages/NotFoundPage";

function App() {
  return (
    <ThemeProvider defaultTheme="dark" storageKey="vite-ui-theme">
      <BrowserRouter>
        <AppShell>
          <AppErrorBoundary>
            <Routes>
              <Route path="/" element={<Navigate to="/devices" replace />} />
              <Route path="/devices" element={<DashboardPage />} />
              <Route path="*" element={<NotFoundPage />} />
            </Routes>
          </AppErrorBoundary>
        </AppShell>
        <Toaster />
      </BrowserRouter>
    </ThemeProvider>
  );
}

export default App;
