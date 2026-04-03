import { Navigate, useLocation } from "react-router-dom";
import { Center, Loader, Stack, Text } from "@mantine/core";
import { useAuth } from "./hooks/useAuth";
import { UserRole } from "@/lib/api";

interface ProtectedRouteProps {
  children: React.ReactNode;
  adminOnly?: boolean;
}

export function ProtectedRoute({ children, adminOnly }: ProtectedRouteProps) {
  const { user, isAuthenticated, isLoading } = useAuth();
  const location = useLocation();

  if (isLoading) {
    return (
      <Center style={{ minHeight: "100vh" }}>
        <Stack align="center" gap="sm">
          <Loader />
          <Text c="dimmed">Loading...</Text>
        </Stack>
      </Center>
    );
  }

  if (!isAuthenticated) {
    const returnTo = location.pathname + location.search;
    return <Navigate to={`/login?returnTo=${encodeURIComponent(returnTo)}`} replace />;
  }

  if (isAuthenticated && user?.must_change_password && location.pathname !== "/settings") {
    return <Navigate to="/settings" replace />;
  }

  if (adminOnly && user?.role !== UserRole.ADMIN) {
    return <Navigate to="/devices" replace />;
  }

  return <>{children}</>;
}
