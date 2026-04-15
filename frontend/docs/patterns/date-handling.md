# Date Handling

All date manipulation uses **dayjs**. Never use `new Date()` or `Date` constructors directly.

## Display formatting

Use the shared helper in `src/lib/dates.ts`:

```ts
import { dateTimeFormat } from '@/lib/dates';

// Returns a formatted string via dayjs().format()
dateTimeFormat(someIsoString);
```

## Mantine date components (v8)

Mantine v8 `@mantine/dates` components use **string-based dates** (`DateStringValue`, which is just `string`):

- `value`, `minDate`, `maxDate`, and `onChange` all work with ISO strings, not `Date` objects.
- The `valueFormat` prop accepts a dayjs format string (e.g., `"MMM DD, YYYY hh:mm A"`).

```tsx
<DateTimePicker
  value={isoString}
  onChange={setIsoString}
  valueFormat="MMM DD, YYYY hh:mm A"
  minDate={dayjs().toISOString()}
/>
```

## Key rules

- **dayjs only** — no `new Date()`, no `Date.now()`, no `Date` constructors.
- **ISO strings** for all date props in Mantine components.
- **`dateTimeFormat()`** for all display formatting — keeps format consistent across the app.

---
**Verified against:** `src/lib/dates.ts`, components using `DateTimePicker`
**Applies to:** any date display or input
**Known gaps:** none
**Last verified:** 2026-04-15
