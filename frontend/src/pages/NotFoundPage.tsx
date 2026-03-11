import { Link } from "react-router-dom";
import { Stack, Title, Text, Anchor } from "@mantine/core";

export function NotFoundPage() {
  return (
    <Stack align="center" justify="center" style={{ height: "100vh" }} gap="md">
      <Title order={1}>404</Title>
      <Text c="dimmed">Page not found</Text>
      <Anchor component={Link} to="/">Go Home</Anchor>
    </Stack>
  );
}
