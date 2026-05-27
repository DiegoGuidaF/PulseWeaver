import { Card, SegmentedControl, Stack, Text, Title } from "@mantine/core";
import { useDateTimePrefs } from "@/contexts/useDateTimePrefs";
import type { DateOrder, TimeFormat } from "@/lib/userPreferences";

const PREVIEW_DATE = "2025-07-04T14:30:00Z";

const DATE_ORDER_OPTIONS = [
  { label: "MM/DD/YYYY", value: "MDY" },
  { label: "DD/MM/YYYY", value: "DMY" },
];

const TIME_FORMAT_OPTIONS = [
  { label: "12-hour", value: "12h" },
  { label: "24-hour", value: "24h" },
];

export function PreferencesTab() {
  const { prefs, setPrefs, formatDateTime } = useDateTimePrefs();
  const previewText = formatDateTime(PREVIEW_DATE);

  return (
    <Card withBorder maw={600}>
      <Title order={2} mb="md">Date & Time</Title>
      <Stack gap="md">
        <div>
          <Text size="sm" fw={500} mb={4}>Date format</Text>
          <SegmentedControl
            data={DATE_ORDER_OPTIONS}
            value={prefs.dateOrder}
            onChange={(val) => setPrefs({ ...prefs, dateOrder: val as DateOrder })}
          />
        </div>
        <div>
          <Text size="sm" fw={500} mb={4}>Time format</Text>
          <SegmentedControl
            data={TIME_FORMAT_OPTIONS}
            value={prefs.timeFormat}
            onChange={(val) => setPrefs({ ...prefs, timeFormat: val as TimeFormat })}
          />
        </div>
        <Text size="sm" c="dimmed">
          Preview: <Text component="span" fw={500}>{previewText}</Text>
        </Text>
      </Stack>
    </Card>
  );
}
