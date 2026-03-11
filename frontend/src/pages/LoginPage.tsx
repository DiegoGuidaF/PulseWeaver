import { Navigate } from "react-router-dom";
import { Center, Stack, Title, Text, Paper, Loader } from "@mantine/core";
import { useAuth } from "@/features/auth/AuthContext";
import { LoginForm } from "@/features/auth/components/LoginForm";

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
    return <Navigate to={user?.must_change_password ? "/settings" : "/devices"} replace />;
  }

  return (
    <Center style={{ minHeight: "100vh" }}>
      <Paper withBorder p="xl" w="100%" maw={448}>
        <Stack gap="sm" mb="lg" ta="center">
          <Title order={2}>Welcome to WallyDic</Title>
          <Text c="dimmed">Sign in to your account to continue</Text>
        </Stack>
        <LoginForm />
      </Paper>
    </Center>
  );
}
