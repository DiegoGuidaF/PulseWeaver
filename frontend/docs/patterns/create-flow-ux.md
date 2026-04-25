# Create Flow UX Patterns

## Pattern: Modal-based creation

For pages where the **list is the primary view** and creation is occasional, use a modal for the create form. This keeps the list front and center.

```
Page layout:
  Group (justify="space-between")
    Title + subtitle
    Button: "Create X" → opens modal
  List component (front and center)

Modal:
  Form → on success → success/detail panel (or close)
```

### Reference: `UsersTab.tsx`

- "Create user" button opens a `<Modal>` with the form
- On success, modal closes and list refreshes via query invalidation
- Destructive actions (delete, role change) use separate confirmation modals

### State shape

```ts
const [createModalOpen, setCreateModalOpen] = useState(false);
// If the modal has a multi-step flow (form → success):
const [createdItem, setCreatedItem] = useState<Item | null>(null);
```

Modal `onClose` should reset all state so reopening starts fresh.

## Pattern: Inline form with state swap

For pages where creation **is** the primary workflow (e.g., a wizard or onboarding), render the form inline on the page. On success, swap the form for a result panel.

```
Page layout:
  Title
  createdItem ? DetailPanel : CreationForm
  List (below)
```

### When to use which

| Signal | Pattern |
|--------|---------|
| List is the main thing admins visit for | Modal |
| Creation is the primary/first action | Inline |
| Form has many fields / is complex | Modal (reduces visual weight on page) |
| Form has 1-2 fields | Either works |

## Extracting modals to feature components

When a page file exceeds ~200 lines or has **3 or more modals**, extract each modal to its own component file under `features/<domain>/components/`. The page retains only state declarations and wires the props.

```
pages/UsersPage.tsx              ← ~150 lines: state + table
features/auth/components/
  CreateUserModal.tsx            ← form + validation
  RoleChangeModal.tsx            ← promote/demote + password
  DeleteUserModal.tsx            ← destructive confirmation
```

Each extracted modal:
- Owns its own mutation hook calls and `notifications.show()` feedback
- Accepts `opened` (or the target object) + `onClose`
- Exposes its state types via named exports (e.g., `PendingRole`, `DeleteTarget`) so the page can type its own `useState`

## Form inside modal conventions

- Add a Cancel button alongside Submit (`Group justify="flex-end"`)
- Pass `onCancel` from the page to close the modal
- `closeOnClickOutside={false}` for forms with sensitive data or multi-step flows
- Form unmounts on modal close (Mantine default: `keepMounted={false}`), which auto-resets `useForm` state

---
**Verified against:** `features/*/components/` (any tab or page with a create action)
**Applies to:** any new create/add flow
**Known gaps:** none
**Last verified:** 2026-04-15
