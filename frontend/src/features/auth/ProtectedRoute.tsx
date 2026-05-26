import { Navigate, useLocation } from "react-router-dom";
import { ROUTES } from "@/lib/routes";
import { Center, Loader, Stack, Text } from "@mantine/core";
import { useAuth } from "./hooks/useAuth";
import React from "react";

interface ProtectedRouteProps {
  children: React.ReactNode;
}

export function ProtectedRoute({ children }: ProtectedRouteProps) {
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

  if (isAuthenticated && user?.must_change_password && location.pathname !== ROUTES.settings) {
    return <Navigate to={ROUTES.settings} replace />;
  }

  return <>{children}</>;
}
