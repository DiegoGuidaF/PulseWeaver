# Feature-Level Constants

## Pattern

Each feature that uses label maps, badge colors, or UI option lists keeps them in a `constants.ts` file at the feature root.

```
features/
  access-log/
    constants.ts        # DENY_REASON_LABELS, re-exports from timePresets
  address-history/
    constants.ts        # SOURCE_LABELS
  provisioning/
    constants.ts        # STATUS_BADGE, FILTER_TAB_OPTIONS, EXPIRING_SOON_MS
```

## What belongs here

- **Label maps** — `Record<EnumValue, string>` mapping API values to display strings
- **Badge color maps** — `Record<Status, { color: string; label: string }>`
- **UI option arrays** — `{ value, label }[]` for `Select`, `SegmentedControl`, etc.
- **Threshold constants** — named magic numbers like `EXPIRING_SOON_MS = 3_600_000`
- **Client-only type aliases** — e.g., `FilterTab` that extends a server enum with UI-only values

## What does NOT belong here

- Types that can be derived from generated OpenAPI types (use indexed access instead)
- Business logic or helper functions (those go in hooks or utils)

## Existing examples

- `access-log/constants.ts` exports `DENY_REASON_LABELS: Record<string, string>`
- `address-history/constants.ts` exports `SOURCE_LABELS: Record<string, string>`

## Badge color convention

The codebase uses a consistent semantic palette for status badges:

| Meaning | Color |
|---------|-------|
| Active / enabled / pending | `green` |
| Inactive / used / disabled | `gray` |
| Denied / expired / error | `red` |
| Warning / expiring soon | `orange` |
| Admin role | `indigo` |

---
**Verified against:** `features/access-log/constants.ts`, `features/provisioning/constants.ts`
**Applies to:** any feature with label maps, badge colors, or UI option lists
**Known gaps:** none
**Last verified:** 2026-04-15
