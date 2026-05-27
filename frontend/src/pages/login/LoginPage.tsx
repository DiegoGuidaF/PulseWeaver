import { Navigate } from "react-router-dom";
import { Center, Stack, Text, Paper, Loader } from "@mantine/core";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { LoginForm } from "@/features/auth/components/LoginForm";
import { ROUTES } from "@/lib/routes";
import { BrandName } from "@/components/BrandName";

export function LoginPage() {
  const { user, isAuthenticated, isLoading } = useAuth();

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

  if (isAuthenticated) {
    return <Navigate to={user?.must_change_password ? ROUTES.settings : ROUTES.dashboard} replace />;
  }

  return (
    <Center style={{ minHeight: "100vh" }}>
      <Paper withBorder p="xl" w="100%" maw={448}>
        <h1 style={{ position: "absolute", width: 1, height: 1, padding: 0, margin: -1, overflow: "hidden", clip: "rect(0,0,0,0)", whiteSpace: "nowrap", border: 0 }}>Sign in</h1>
        <Stack gap="xs" mb="lg" ta="center">
          <BrandName size="2.5rem" style={{ display: "block" }} />
          <Text c="dimmed">Sign in to your account to continue</Text>
        </Stack>
        <LoginForm />
      </Paper>
    </Center>
  );
}
