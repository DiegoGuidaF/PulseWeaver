/** Value widget used for the multi-value operators (`in` / `not_in`). */
export type ValueWidget = "tags" | "multiselect";

/** A column's filter as held in the URL: an operator plus 0..N values. */
export interface ColumnFilterState<Op extends string = string> {
    op: Op;
    values: string[];
}

export interface FilterColumnConfig<Op extends string = string> {
    operators: Op[];
    widget: ValueWidget;
    /** Whether the column's values are numeric IDs. */
    numeric?: boolean;
    /** Column-tuned wording for the value-less operators. */
    emptyLabel?: string;
    notEmptyLabel?: string;
}

const OP_LABELS: Record<string, string> = {
    in: "is any of",
    not_in: "is none of",
    contains: "contains",
    not_contains: "does not contain",
    is_null: "is empty",
    not_null: "is not empty",
};

export const NULL_OPS = new Set(["is_null", "not_null"]);
export const CONTAINS_OPS = new Set(["contains", "not_contains"]);

/** Human label for an operator, honouring a column's empty/not-empty overrides. */
export function operatorLabel<Op extends string>(config: FilterColumnConfig<Op>, op: Op): string {
    if (op === "is_null" && config.emptyLabel) return config.emptyLabel;
    if (op === "not_null" && config.notEmptyLabel) return config.notEmptyLabel;
    return OP_LABELS[op] ?? op;
}

/** A column filter is active when it has values, or uses a value-less operator. */
export function isFilterActive<Op extends string>(state: ColumnFilterState<Op>): boolean {
    return state.values.length > 0 || NULL_OPS.has(state.op);
}
