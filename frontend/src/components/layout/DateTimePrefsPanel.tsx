import { SegmentedControl, Stack, Text } from "@mantine/core";
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

/**
 * Browser-local date/time display preferences (order + 12h/24h). Stored in
 * localStorage via useDateTimePrefs, not on the account — these are per-device.
 * Rendered inside the top-bar user menu.
 */
export function DateTimePrefsPanel() {
  const { prefs, setPrefs, formatDateTime } = useDateTimePrefs();

  return (
    <Stack gap="sm">
      <div>
        <Text size="xs" fw={500} mb={4}>Date format</Text>
        <SegmentedControl
          fullWidth
          size="xs"
          data={DATE_ORDER_OPTIONS}
          value={prefs.dateOrder}
          onChange={(val) => setPrefs({ ...prefs, dateOrder: val as DateOrder })}
        />
      </div>
      <div>
        <Text size="xs" fw={500} mb={4}>Time format</Text>
        <SegmentedControl
          fullWidth
          size="xs"
          data={TIME_FORMAT_OPTIONS}
          value={prefs.timeFormat}
          onChange={(val) => setPrefs({ ...prefs, timeFormat: val as TimeFormat })}
        />
      </div>
      <Text size="xs" c="dimmed">
        Preview: <Text component="span" fw={500}>{formatDateTime(PREVIEW_DATE)}</Text>
      </Text>
    </Stack>
  );
}
