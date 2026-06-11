import { useEffect, useRef, useState } from "react";
import { Button, Group, MultiSelect, Select, Stack, TagsInput, Text, TextInput } from "@mantine/core";
import { IconSearch } from "@tabler/icons-react";
import {
    type ColumnFilterState,
    type FilterColumnConfig,
    type FilterOp,
    operatorLabel,
} from "../filterConfig";

interface ColumnFilterProps {
    config: FilterColumnConfig;
    state: ColumnFilterState;
    /** Options for the multi-select widget; absent → free-form tag input. */
    options?: { value: string; label: string }[];
    placeholder?: string;
    /** Commit the edited filter. Fired once, when the popover closes or Apply is pressed. */
    onCommit: (next: ColumnFilterState) => void;
    width?: number;
}

function sameValues(a: string[], b: string[]): boolean {
    return a.length === b.length && a.every((v, i) => v === b[i]);
}

/**
 * Operator-aware filter for one column: an operator selector plus a value widget
 * that adapts to the operator — multi-select/tags for `in`/`not_in`, a single
 * text field for `contains`/`not_contains`, and no input for `is_null`/`not_null`.
 *
 * Edits are staged locally and committed once, when the filter popover closes
 * (this node unmounts) — so fiddling with the operator or adding several values
 * results in a single backend query rather than one per keystroke or selection.
 */
export function ColumnFilter({
    config,
    state,
    options,
    placeholder,
    onCommit,
    width = 240,
}: ColumnFilterProps) {
    const [draft, setDraft] = useState<ColumnFilterState>(() => state);
    const { op, values } = draft;
    const isNullOp = op === "is_null" || op === "not_null";
    const isContains = op === "contains" || op === "not_contains";

    // The popover writes nothing while open; it commits on close, when this node
    // unmounts. The committed snapshot (captured once) lets us skip a no-op write,
    // and the refs — synced in an effect, never during render — give the unmount
    // cleanup the latest draft and callback without re-arming on every change.
    const [committed] = useState(state);
    const draftRef = useRef(draft);
    const commitRef = useRef(onCommit);
    useEffect(() => {
        draftRef.current = draft;
        commitRef.current = onCommit;
    });
    useEffect(
        () => () => {
            const d = draftRef.current;
            if (d.op !== committed.op || !sameValues(d.values, committed.values)) {
                commitRef.current(d);
            }
        },
        [committed],
    );

    function changeOperator(next: FilterOp) {
        if (next === "is_null" || next === "not_null") {
            setDraft({ op: next, values: [] });
        } else if (next === "contains" || next === "not_contains") {
            setDraft((prev) => ({ op: next, values: prev.values.slice(0, 1) }));
        } else {
            setDraft((prev) => ({ op: next, values: prev.values }));
        }
    }

    return (
        <Stack gap="xs">
            {config.operators.length > 1 && (
                <Select
                    size="xs"
                    aria-label="Filter operator"
                    data={config.operators.map((o) => ({ value: o, label: operatorLabel(config, o) }))}
                    value={op}
                    onChange={(v) => v && changeOperator(v as FilterOp)}
                    comboboxProps={{ withinPortal: false }}
                    allowDeselect={false}
                    w={width}
                />
            )}

            {isNullOp ? (
                <Text size="xs" c="dimmed">
                    No value needed.
                </Text>
            ) : isContains ? (
                <TextInput
                    placeholder={placeholder ?? "Containing…"}
                    leftSection={<IconSearch size={16} />}
                    value={values[0] ?? ""}
                    onChange={(e) => {
                        const v = e.currentTarget.value;
                        setDraft({ op, values: v ? [v] : [] });
                    }}
                    w={width}
                />
            ) : options ? (
                <MultiSelect
                    placeholder={placeholder ?? "Select values"}
                    data={options}
                    value={values}
                    onChange={(v) => setDraft({ op, values: v })}
                    searchable
                    clearable
                    comboboxProps={{ withinPortal: false }}
                    w={width}
                />
            ) : (
                <TagsInput
                    placeholder={placeholder ?? "Type and press Enter"}
                    value={values}
                    onChange={(v) => setDraft({ op, values: v })}
                    clearable
                    comboboxProps={{ withinPortal: false }}
                    w={width}
                />
            )}
        </Stack>
    );
}

/** Footer button that closes a filter popover, committing the staged edit. */
export function FilterApplyButton({ onApply }: { onApply: () => void }) {
    return (
        <Group justify="flex-end">
            <Button size="xs" variant="light" onClick={onApply}>
                Apply
            </Button>
        </Group>
    );
}
