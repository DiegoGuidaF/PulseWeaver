import { useState } from "react";
import { MultiSelect, Select, Stack, TagsInput, Text, TextInput } from "@mantine/core";
import { useDebouncedCallback } from "@mantine/hooks";
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
    onChange: (next: ColumnFilterState) => void;
    width?: number;
}

/**
 * Operator-aware filter for one column: an operator selector plus a value widget
 * that adapts to the operator — multi-select/tags for `in`/`not_in`, a single
 * text field for `contains`/`not_contains`, and no input for `is_null`/`not_null`.
 */
export function ColumnFilter({
    config,
    state,
    options,
    placeholder,
    onChange,
    width = 240,
}: ColumnFilterProps) {
    const { op, values } = state;
    const isNullOp = op === "is_null" || op === "not_null";
    const isContains = op === "contains" || op === "not_contains";

    // Local mirror for the contains text field so typing stays responsive; the
    // URL write is debounced.
    const [text, setText] = useState(values[0] ?? "");
    const writeText = useDebouncedCallback((value: string) => {
        onChange({ op, values: value ? [value] : [] });
    }, 300);

    function changeOperator(next: FilterOp) {
        if (next === "is_null" || next === "not_null") {
            onChange({ op: next, values: [] });
        } else if (next === "contains" || next === "not_contains") {
            onChange({ op: next, values: text ? [text] : [] });
        } else {
            onChange({ op: next, values });
        }
    }

    return (
        <Stack gap="xs" p="xs" w={width + 24}>
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
                    value={text}
                    onChange={(e) => {
                        setText(e.currentTarget.value);
                        writeText(e.currentTarget.value);
                    }}
                    w={width}
                />
            ) : options ? (
                <MultiSelect
                    placeholder={placeholder ?? "Select values"}
                    data={options}
                    value={values}
                    onChange={(v) => onChange({ op, values: v })}
                    searchable
                    clearable
                    comboboxProps={{ withinPortal: false }}
                    w={width}
                />
            ) : (
                <TagsInput
                    placeholder={placeholder ?? "Type and press Enter"}
                    value={values}
                    onChange={(v) => onChange({ op, values: v })}
                    clearable
                    comboboxProps={{ withinPortal: false }}
                    w={width}
                />
            )}
        </Stack>
    );
}
