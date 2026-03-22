import { useState, useMemo } from "react";
import { Badge, Card, Group, SegmentedControl, Skeleton, Stack, Table, Text, Title } from "@mantine/core";
import { AreaChart } from "@mantine/charts";
import dayjs from "dayjs";
import { useAddressHistory } from "./hooks/useAddressHistory";
import { useDateFormatter } from "@/contexts/useDateTimePrefs";

interface DeviceHistoryTabProps {
  deviceId: number;
}

type TimeRange = "24h" | "7d" | "30d";

const RANGE_HOURS: Record<TimeRange, number> = {
  "24h": 24,
  "7d": 168,
  "30d": 720,
};

const RANGE_GRANULARITY: Record<TimeRange, "hour" | "day"> = {
  "24h": "hour",
  "7d": "hour",
  "30d": "day",
};

function sourceBadgeColor(source: string): string {
  switch (source) {
    case "heartbeat":
      return "blue";
    case "manual":
      return "grape";
    case "expiry":
      return "orange";
    default:
      return "gray";
  }
}

export function DeviceHistoryTab({ deviceId }: DeviceHistoryTabProps) {
  const [range, setRange] = useState<TimeRange>("24h");
  const formatDateTime = useDateFormatter();

  const from = useMemo(
    () => dayjs().subtract(RANGE_HOURS[range], "hour").toISOString(),
    [range],
  );

  const { data, isLoading } = useAddressHistory({
    device_id: [deviceId],
    from,
    granularity: RANGE_GRANULARITY[range],
  });

  const chartData = useMemo(() => {
    if (!data?.buckets) return [];
    return data.buckets.map((b) => ({
      timestamp: dayjs(b.timestamp).format(range === "30d" ? "MMM DD" : "MMM DD HH:mm"),
      active_count: b.active_count,
    }));
  }, [data, range]);

  if (isLoading) {
    return (
      <Stack gap="md">
        <Skeleton height={32} width={240} />
        <Skeleton height={200} />
        <Skeleton height={200} />
      </Stack>
    );
  }

  const hasData = data && (data.buckets.length > 0 || data.events.length > 0);

  return (
    <Stack gap="md">
      <Group justify="space-between" align="center">
        <Title order={4}>Address Activity</Title>
        <SegmentedControl
          size="xs"
          value={range}
          onChange={(v) => setRange(v as TimeRange)}
          data={[
            { label: "24h", value: "24h" },
            { label: "7 days", value: "7d" },
            { label: "30 days", value: "30d" },
          ]}
        />
      </Group>

      <Card withBorder padding="md">
        <Text fw={500} mb="sm">Active IPs over time</Text>
        {chartData.length > 0 ? (
          <AreaChart
            h={200}
            data={chartData}
            dataKey="timestamp"
            series={[{ name: "active_count", color: "blue.6" }]}
            curveType="monotone"
            gridAxis="xy"
            yAxisProps={{ allowDecimals: false }}
          />
        ) : (
          <Text size="sm" c="dimmed" ta="center" py="xl">
            No activity in this period
          </Text>
        )}
      </Card>

      <Card withBorder padding="md">
        <Text fw={500} mb="sm">Event log</Text>
        {hasData && data.events.length > 0 ? (
          <Table striped highlightOnHover>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>Time</Table.Th>
                <Table.Th>IP</Table.Th>
                <Table.Th>Status</Table.Th>
                <Table.Th>Source</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {data.events.map((event) => (
                <Table.Tr key={event.id}>
                  <Table.Td>
                    <Text size="sm" ff="monospace">
                      {formatDateTime(event.timestamp)}
                    </Text>
                  </Table.Td>
                  <Table.Td>
                    <Text size="sm" ff="monospace">{event.ip}</Text>
                  </Table.Td>
                  <Table.Td>
                    <Badge
                      size="sm"
                      color={event.is_enabled ? "green" : "red"}
                      variant="light"
                    >
                      {event.is_enabled ? "Enabled" : "Disabled"}
                    </Badge>
                  </Table.Td>
                  <Table.Td>
                    <Badge
                      size="sm"
                      color={sourceBadgeColor(event.source)}
                      variant="dot"
                    >
                      {event.source}
                    </Badge>
                  </Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
        ) : (
          <Text size="sm" c="dimmed" ta="center" py="xl">
            No events in this period
          </Text>
        )}
      </Card>
    </Stack>
  );
}
