# Shared Hooks

## Pattern

Reusable cross-feature hooks live in `src/hooks/`. Before writing inline logic for common browser APIs or UI concerns, check if a shared hook already exists.

## Available shared hooks

| Hook | Location | Purpose |
|------|----------|---------|
| `useClipboard` | `src/hooks/useClipboard.ts` | Clipboard write with notifications and browser support check |

### `useClipboard`

Returns `{ copy }` — an async function that writes to clipboard and shows a Mantine notification on success or failure. Handles the case where `navigator.clipboard` is unavailable.

```ts
const { copy } = useClipboard();

// Basic usage
copy(text);

// Custom messages
copy(text, { successMessage: "Code copied", errorMessage: "Copy failed" });
```

Prefer this over direct `navigator.clipboard.writeText()` calls to get consistent error handling and user feedback.

## When to add a new shared hook

A hook moves to `src/hooks/` when it is used by two or more features and has no feature-specific dependencies. Single-use hooks stay in the feature's `hooks/` directory.

---
**Verified against:** `src/hooks/useClipboard.ts`
**Applies to:** any new shared hook
**Known gaps:** none
**Last verified:** 2026-04-15
