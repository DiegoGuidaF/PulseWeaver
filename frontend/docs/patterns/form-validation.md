# Form Validation with Generated Zod Schemas

## Pattern

Forms use `@mantine/form` with `schemaResolver` and the **generated zod schema** from `@/lib/api/zod.gen`. The form values type is derived via `z.infer`, not manually duplicated.

```ts
import { useForm, schemaResolver } from "@mantine/form";
import { zCreateUserRequest } from "@/lib/api/zod.gen";
import type { z } from "zod";

const schema = zCreateUserRequest;
type FormValues = z.infer<typeof schema>;

const form = useForm<FormValues>({
  validate: schemaResolver(schema),
  initialValues: { ... },
});
```

## Why

The OpenAPI spec defines constraints (`minLength`, `maxLength`, `format: uri`, `minimum`, `enum`). The code generator (`heyapi`) produces zod schemas that include these constraints. Deriving `FormValues` from the schema avoids maintaining a duplicate type that can drift.

## Reference

- `UsersTab.tsx` — uses `z.infer<typeof zCreateUserRequest>` for the create-user form
- Generated schemas live in `src/lib/api/zod.gen.ts`
- Constraint examples: `z.string().min(1).max(100)`, `z.url()`, `z.int().gte(60)`, `z.union([z.literal(...)])`

## Note on `.default()` fields

Zod schemas with `.optional().default(value)` produce `T` (not `T | undefined`) under `z.infer`. This means `z.infer` gives the **output** type, which is safe for `useForm` initial values and submission.

---
**Verified against:** `UsersTab.tsx`, `src/lib/api/zod.gen.ts`
**Applies to:** any form with validation
**Known gaps:** none
**Last verified:** 2026-04-15
