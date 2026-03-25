# Frontend Coding Standards & Style Guide

## General Philosophy

- **Modular:** Components should be self-contained where possible (colocation).
- **Low Maintenance:** Prefer standard libraries (TanStack Query, React Router) over custom implementations.

## TypeScript Rules

- **Strict Typing:** No `any`. Use `unknown` if necessary and narrow types.
- **Inferred Return Types:** Let TS infer return types for components and hooks unless complex.
- **Props:** Use `interface` for props definition. Prefix with Component Name (e.g., `DeviceListProps`).
- **No Enums:** Use `const` assertions (`as const`) or string unions.

## Component Patterns

- **Functional Components:** Always use `function ComponentName() {}`. Avoid `const ComponentName = () => {}`.
- **Destructuring:** Destructure props immediately in the function signature.
- **Colocation:** Keep related sub-components in the same file if they are small and private.
- **Exporting:** Named exports (`export function`) preferred over `export default`.

## State Management

- **Server State:** Use `useQuery` for fetching. Handle `isLoading` and `isError` explicitly.
- **Form State:** Use `react-hook-form` + `zod` for all forms.
    - Define schema outside the component.
    - Infer types from Zod schema: `type FormValues = z.infer<typeof formSchema>`.
- **Global State:** Avoid global context for data. Use it only for app-wide configuration (Theme, AuthUser).

## API Interaction

- **Generation:** Run `npm run generate:api` after backend changes.
- **heyapi + React Query:** Query and mutation options and query keys are generated in `@/lib/api/@tanstack/react-query.gen`. Use `useQuery(getXxxOptions(...))` and `useMutation({ ...xxxMutation(), onSuccess, onError })`; pass full options to `mutate()` (e.g. `mutate({ body: values })`).
- **Usage:**
  ```typescript
  // Good: use generated options
  import { getDevicesOptions } from '@/lib/api/@tanstack/react-query.gen';

  const { data } = useQuery(getDevicesOptions());
  ```

## Query Keys & Invalidation

- **Generated keys:** For OpenAPI-defined endpoints, use query keys from `@/lib/api/@tanstack/react-query.gen`: `getDevicesQueryKey()`, `getDeviceAddressesQueryKey({ path: { device_id } })`, `getCurrentUserQueryKey()`, etc.
- **Custom keys:** Only in `@/lib/api-client/queryKeys.ts` for endpoints not yet in the generated React Query layer.
- **Example:**
  ```typescript
  import { getDevicesQueryKey } from '@/lib/api/@tanstack/react-query.gen';

  queryClient.invalidateQueries({ queryKey: getDevicesQueryKey() });
  ```

## Mutations & Cache Invalidation

- **Pattern:** Use generated mutation options: `useMutation({ ...createDeviceMutation(), onSuccess, onError })`. Do not define custom `mutationFn` for HTTP.
- **Error Handling:** Always show Sonner toast for errors (`onError` with `toErrorMessage(err)`).
- **Success Feedback:** Show success toast for user actions.
- **Example:**
  ```typescript
  import { createDeviceMutation, getDevicesQueryKey } from '@/lib/api/@tanstack/react-query.gen';
  import { toErrorMessage } from '@/lib/api-client';

  export function useCreateDevice(options?: { onSuccess?: () => void }) {
    const queryClient = useQueryClient();
    return useMutation({
      ...createDeviceMutation(),
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: getDevicesQueryKey() });
        toast.success("Device created", { description: "The new device has been added successfully." });
        options?.onSuccess?.();
      },
      onError: (err) => {
        toast.error("Error creating device", { description: toErrorMessage(err) });
      },
    });
  }
  // Call site: mutation.mutate({ body: values });
  ```
- **Invalidation Strategy:** Invalidate the most specific query key that needs refreshing (use generated keys). For list mutations, invalidate the list key. For detail mutations, invalidate both detail and list keys if needed.
