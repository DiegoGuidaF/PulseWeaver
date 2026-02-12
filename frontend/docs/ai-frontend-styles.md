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

## Styling (Tailwind + shadcn)

- **Utility First:** Use Tailwind classes for layout, spacing, and colors.
- **Conditionals:** Use `cn()` helper for conditional classes.
    - *Bad:* `` className={`p-4 ${isActive ? 'bg-blue-500' : ''}`} ``
    - *Good:* `className={cn("p-4", isActive && "bg-blue-500")}`
- **Spacing:** Use standard Tailwind spacing steps (4, 8, 16, etc.).
- **Colors:** Use CSS variables defined in shadcn (`bg-primary`, `text-muted-foreground`) to support Dark Mode
  automatically.

## API Interaction

- **Generation:** Run `npm run generate:api` after backend changes.
- **Usage:**
  ```typescript
  // Good
  const { data } = useQuery({
      queryKey: ['devices'],
      queryFn: async () => {
          const { data, error } = await api.GET('/devices');
          if (error) throw new Error(toErrorMessage(error));
          return data;
      }
  });
  ```

## Query Keys Pattern

- **Factory Pattern:** Use a centralized `queryKeys` object with factory functions for type-safe, consistent query keys.
- **Location:** `@/lib/api/queryKeys.ts`
- **Structure:** Group by feature domain, use factory functions for parameterized keys.
- **Example:**
  ```typescript
  export const queryKeys = {
    devices: {
      all: ["devices"] as const,
      detail: (id: number) => ["devices", id] as const,
      addresses: (deviceId: number) => ["device-addresses", deviceId] as const,
    },
  };
  
  // Usage in hooks
  const { data } = useQuery({
    queryKey: queryKeys.devices.all,
    queryFn: async () => { /* ... */ }
  });
  ```
- **Benefits:** Type safety, refactoring safety, consistent key structure, easy invalidation.

## Mutations & Cache Invalidation

- **Pattern:** Use `useMutation` with `onSuccess` to invalidate related queries.
- **Error Handling:** Always show Sonner toast for errors (`onError`).
- **Success Feedback:** Show success toast for user actions.
- **Example:**
  ```typescript
  export function useCreateDevice(options?: { onSuccess?: () => void }) {
    const queryClient = useQueryClient();
    
    return useMutation({
      mutationFn: async (values: { name: string }) => {
        const { data, error } = await api.POST("/devices", { body: values });
        if (error) throw new Error(toErrorMessage(error));
        return data;
      },
      onSuccess: () => {
        // Invalidate the list query to refetch
        queryClient.invalidateQueries({ queryKey: queryKeys.devices.all });
        toast.success("Device created", {
          description: "The new device has been added successfully.",
        });
        options?.onSuccess?.();
      },
      onError: (err) => {
        toast.error("Error creating device", {
          description: err.message,
        });
      },
    });
  }
  ```
- **Invalidation Strategy:** Invalidate the most specific query key that needs refreshing. For list mutations, invalidate the list key. For detail mutations, invalidate both detail and list keys if needed.
