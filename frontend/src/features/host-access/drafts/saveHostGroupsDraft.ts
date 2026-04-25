import type { Id } from "@/lib/api";
import type { GroupsDraftState } from "./hostGroupsDraft";
import { diffGroups } from "./hostGroupsDraft";
import type { SaveResult, SaveResultEntry } from "./saveKnownHostsDraft";

export interface SaveGroupsDeps {
  createHostGroupAsync: (input: {
    body: {
      name: string;
      description?: string | null;
      icon?: string | null;
      host_ids?: Id[];
    };
  }) => Promise<unknown>;
  updateHostGroupAsync: (input: {
    path: { group_id: Id };
    body: {
      name: string;
      description?: string | null;
      icon?: string | null;
      host_ids?: Id[];
    };
  }) => Promise<unknown>;
  deleteHostGroupAsync: (input: {
    path: { group_id: Id };
  }) => Promise<unknown>;
}

// `color` is intentionally dropped here — the backend column doesn't exist yet.
// When it lands, add `color: draft.color ?? null` to both create and update bodies.
export async function saveHostGroupsDraft(
  state: GroupsDraftState,
  deps: SaveGroupsDeps,
): Promise<SaveResult> {
  const diff = diffGroups(state);
  const failed: SaveResultEntry[] = [];
  let succeeded = 0;

  const tasks: Promise<void>[] = [];

  for (const draft of diff.added) {
    tasks.push(
      deps
        .createHostGroupAsync({
          body: {
            name: draft.name,
            description: draft.description,
            icon: draft.icon,
            host_ids: draft.hostIds,
          },
        })
        .then(
          () => {
            succeeded += 1;
          },
          (err: unknown) => {
            failed.push({ label: `Create group ${draft.name}`, error: errorMessage(err) });
          },
        ),
    );
  }

  for (const entry of diff.changed) {
    const draft = entry.group;
    if (typeof draft.id !== "number") continue;
    tasks.push(
      deps
        .updateHostGroupAsync({
          path: { group_id: draft.id },
          body: {
            name: draft.name,
            description: draft.description,
            icon: draft.icon,
            host_ids: draft.hostIds,
          },
        })
        .then(
          () => {
            succeeded += 1;
          },
          (err: unknown) => {
            failed.push({ label: `Update group ${draft.name}`, error: errorMessage(err) });
          },
        ),
    );
  }

  for (const removed of diff.removed) {
    tasks.push(
      deps.deleteHostGroupAsync({ path: { group_id: removed.id } }).then(
        () => {
          succeeded += 1;
        },
        (err: unknown) => {
          failed.push({ label: `Delete group ${removed.name}`, error: errorMessage(err) });
        },
      ),
    );
  }

  await Promise.all(tasks);
  return { succeeded, failed };
}

function errorMessage(err: unknown): string {
  if (err instanceof Error) return err.message;
  return String(err);
}
