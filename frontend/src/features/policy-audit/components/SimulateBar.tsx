import { useState } from "react";
import { Link } from "react-router-dom";
import { Alert, Anchor, Button, Group, Text, TextInput } from "@mantine/core";
import { IconAlertTriangle, IconCircleCheck, IconCircleX, IconPlayerPlay } from "@tabler/icons-react";
import { usePolicySimulate } from "../hooks/usePolicySimulate";
import { POLICY_DENY_REASON_LABELS } from "@/lib/policyDenyReasons";
import { toErrorMessage } from "@/lib/api-client/errors";

interface SimulateBarProps {
  ip: string;
  onIpChange: (ip: string) => void;
}

export function SimulateBar({ ip, onIpChange }: SimulateBarProps) {
  const [host, setHost] = useState("");
  const [dirty, setDirty] = useState(false);
  const { result, isFetching, isError, error, refetch } = usePolicySimulate(ip, host);

  function handleIpChange(value: string) {
    onIpChange(value);
    setDirty(true);
  }

  function handleHostChange(value: string) {
    setHost(value);
    setDirty(true);
  }

  function handleSubmit() {
    if (ip.trim() && host.trim()) {
      setDirty(false);
      void refetch();
    }
  }

  const canSubmit = ip.trim().length > 0 && host.trim().length > 0;
  // React Query keeps the previous data on error, so check isError first to
  // avoid presenting a stale decision as the outcome of the failed request.
  const showError = !dirty && isError;
  const showResult = !dirty && !isError && result != null;

  return (
    <div>
      <Group align="flex-end" gap="sm">
        <TextInput
          label="Source IP"
          placeholder="e.g. 192.168.1.148"
          value={ip}
          onChange={(e) => handleIpChange(e.currentTarget.value)}
          style={{ flex: 1 }}
          ff="monospace"
        />
        <TextInput
          label="Host (FQDN)"
          placeholder="e.g. immich.myhome.org"
          value={host}
          onChange={(e) => handleHostChange(e.currentTarget.value)}
          style={{ flex: 1 }}
          ff="monospace"
        />
        <Button
          onClick={handleSubmit}
          loading={isFetching}
          disabled={!canSubmit}
          leftSection={<IconPlayerPlay size={14} />}
        >
          Test
        </Button>
      </Group>

      {showError && (
        <Alert
          mt="sm"
          color="red"
          icon={<IconAlertTriangle size={18} />}
          title="Simulation failed"
        >
          <Text size="sm">{toErrorMessage(error)}</Text>
        </Alert>
      )}

      {showResult && (
        <Alert
          mt="sm"
          color={result.allowed ? "green" : "red"}
          icon={result.allowed ? <IconCircleCheck size={18} /> : <IconCircleX size={18} />}
          title={result.allowed ? "Allowed" : "Denied"}
        >
          <Text size="sm" ff="monospace">
            {result.ip} → {result.host}
          </Text>
          {result.allowed && result.match_source === "network_policy" && result.network_policy_name && (
            <Text size="sm" mt={4}>
              Matched by Network Policy:{" "}
              <Anchor component={Link} to={`/network-policies/${result.network_policy_id}`}>
                {result.network_policy_name}
              </Anchor>
            </Text>
          )}
          {result.allowed && result.match_source === "device" && (
            <Text size="sm" mt={4} c="dimmed">Matched by device address</Text>
          )}
          {!result.allowed && result.deny_reason && (
            <Text size="sm" mt={4}>
              {POLICY_DENY_REASON_LABELS[result.deny_reason]}
            </Text>
          )}
        </Alert>
      )}
    </div>
  );
}
