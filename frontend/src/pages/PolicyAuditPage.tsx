import { useState } from "react";
import { Alert, Card, Center, Loader, Stack, Text, Title } from "@mantine/core";
import { usePolicyMap } from "@/features/policy-audit/hooks/usePolicyMap";
import { SimulateBar } from "@/features/policy-audit/components/SimulateBar";
import { PolicyMapTable } from "@/features/policy-audit/components/PolicyMapTable";

export function PolicyAuditPage() {
  const { data, isPending, isError } = usePolicyMap();
  const [simulateIp, setSimulateIp] = useState("");

  return (
    <Stack maw={1100} gap="md">
      <div>
        <Title order={1}>Policy Cache</Title>
        <Text c="dimmed" mt={4}>
          Inspect the live policy engine state and simulate access decisions.
        </Text>
      </div>

      <Card withBorder>
        <Stack gap="xs">
          <Text size="sm" fw={500}>
            Simulate access
          </Text>
          <Text size="xs" c="dimmed">
            Click an IP in the table below to pre-fill, then enter a host to check the current
            decision without sending real traffic.
          </Text>
          <SimulateBar ip={simulateIp} onIpChange={setSimulateIp} />
        </Stack>
      </Card>

      {isPending && (
        <Center py="xl">
          <Loader />
        </Center>
      )}

      {isError && (
        <Alert color="red" title="Failed to load policy cache">
          Could not fetch the policy map snapshot. Make sure you have admin access.
        </Alert>
      )}

      {data && <PolicyMapTable data={data} onSelectIp={setSimulateIp} />}
    </Stack>
  );
}
