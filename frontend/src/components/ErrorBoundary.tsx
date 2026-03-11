import type { FallbackProps } from "react-error-boundary";
import { ErrorBoundary as ReactErrorBoundary } from "react-error-boundary";
import { Button, Stack, Text, Title } from "@mantine/core";

function ErrorFallback({ error, resetErrorBoundary }: FallbackProps) {
  const message =
    error instanceof Error ? error.message : "An unexpected error occurred";

  return (
    <Stack
      align="center"
      justify="center"
      style={{ minHeight: "50vh" }}
      p="xl"
      gap="md"
    >
      <Title order={2}>Something went wrong</Title>
      <Text c="dimmed" ta="center" maw={400}>{message}</Text>
      <Button onClick={resetErrorBoundary}>Try again</Button>
    </Stack>
  );
}

export function AppErrorBoundary({ children }: { children: React.ReactNode }) {
  return (
    <ReactErrorBoundary FallbackComponent={ErrorFallback}>
      {children}
    </ReactErrorBoundary>
  );
}
