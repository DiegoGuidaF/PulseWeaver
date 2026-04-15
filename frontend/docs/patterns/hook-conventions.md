# Hook Conventions (Query & Mutation)

Hooks are thin wrappers around TanStack Query that own server state. They never own UI concerns.

## Query hooks

Wrap a generated options factory, return the TanStack Query result directly:

```ts
import { useQuery } from '@tanstack/react-query';
import { getDevicesOptions } from '@/lib/api/@tanstack/react-query.gen';

export function useDevices() {
  return useQuery(getDevicesOptions());
}
```

- **Return type**: TanStack Query result directly. Exceptions are allowed when normalization is needed (e.g., `null` for 404).
- **`enabled` default**: Query hooks that accept `enabled` default it to `true`.
- **Cache invalidation**: Mutations invalidate the minimal relevant query key (device-specific where possible).

## Mutation hooks

Own server state only (mutation + cache invalidation). **No `notifications.show()` calls inside hooks.**

```ts
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { createDeviceMutation, getDevicesQueryKey } from '@/lib/api/@tanstack/react-query.gen';

export function useCreateDevice() {
  const queryClient = useQueryClient();
  return useMutation({
    ...createDeviceMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: getDevicesQueryKey() });
    },
  });
}
```

## Component usage

Components own UI feedback. Notifications and coordination go in component callbacks:

```ts
mutation.mutate({ body: values }, {
  onSuccess: (data) => { setCreatedResult(data); },
  onError: (err) => notifications.show({ color: 'red', title: 'Error', message: toErrorMessage(err) }),
});
```

- **Notifications**: `notifications.show()` belongs in component `onSuccess`/`onError` callbacks, not in hooks.
- **Coordination callbacks**: Hooks may accept an `onSuccess` option for coordination logic (form reset, modal close).

---
**Verified against:** `features/devices/hooks/`, `features/auth/hooks/`
**Applies to:** all query and mutation hooks
**Known gaps:** none
**Last verified:** 2026-04-15
