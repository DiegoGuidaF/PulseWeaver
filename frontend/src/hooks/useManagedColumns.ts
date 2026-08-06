import { useEffect } from "react";
import { useDataTableColumns, type DataTableColumn } from "mantine-datatable";

/**
 * Declaration of one user-manageable column: what the chooser calls it, whether
 * it can be hidden, and whether it starts visible. Kept separate from the
 * `DataTableColumn` array so the column factory stays render/filter-only.
 */
export interface ManagedColumnMeta {
    accessor: string;
    label: string;
    /**
     * Always shown, never toggleable, and never draggable. Column drag is a
     * swap, and a non-draggable column is neither a swap source nor target, so
     * a mandatory column stays locked at index 0 — which is what keeps a static
     * `pinFirstColumn` pinned to the intended column.
     */
    mandatory?: boolean;
    /** Visible on first visit, before the user customises the chooser. */
    defaultVisible?: boolean;
}

export interface UseManagedColumnsOptions<T> {
    /**
     * Store key for column order, visibility and width. The same value must
     * reach `<DataTable storeColumnsKey>` or persistence silently breaks. Bump
     * the suffix (`…:v4`) when the persisted shape changes.
     */
    storeKey: string;
    /** Render-focused columns. Any accessor absent from `meta` passes through unmanaged. */
    columns: DataTableColumn<T>[];
    meta: ManagedColumnMeta[];
    /**
     * Lean first-visit column set for narrow viewports, used in place of
     * `meta`'s `defaultVisible` when `compact` is true. Seeds initial state
     * only — a stored choice wins at any width.
     */
    compactVisible?: string[];
    compact?: boolean;
    /** Narrowest a column may be dragged before its header controls get clipped. */
    minColumnWidth?: number;
}

const DEFAULT_MIN_COLUMN_WIDTH = 110;

/**
 * User-managed table columns — resize, drag-to-reorder, and show/hide with
 * persistence — over mantine-datatable's `useDataTableColumns`.
 *
 * Pass `effectiveColumns` to `<DataTable>` along with the same `storeKey` as
 * `storeColumnsKey`, and render the chooser from `columnVisible` /
 * `setColumnVisible` (see `ColumnsMenu`).
 */
export function useManagedColumns<T>({
    storeKey,
    columns,
    meta,
    compactVisible,
    compact = false,
    minColumnWidth = DEFAULT_MIN_COLUMN_WIDTH,
}: UseManagedColumnsOptions<T>) {
    const managedAccessors = new Set(meta.map((m) => m.accessor));
    const mandatory = new Set(meta.filter((m) => m.mandatory).map((m) => m.accessor));
    const defaultVisible = new Set(
        compact && compactVisible
            ? compactVisible
            : meta.filter((m) => !m.mandatory && m.defaultVisible).map((m) => m.accessor),
    );

    const managedColumns = columns.map((col) => {
        const accessor = String(col.accessor);
        // Columns the caller didn't declare (a trailing actions column) are left
        // exactly as given — not resizable, not draggable, not in the chooser.
        if (!managedAccessors.has(accessor)) return col;
        const isMandatory = mandatory.has(accessor);
        return {
            ...col,
            resizable: true,
            draggable: !isMandatory,
            // The per-header hide icon is suppressed in favour of the chooser,
            // which is the single place visibility is driven from.
            toggleable: false,
            defaultToggle: isMandatory || defaultVisible.has(accessor),
        };
    });

    const {
        effectiveColumns,
        columnsToggle,
        setColumnsToggle,
        columnsWidth,
        setColumnsWidth,
        resetColumnsOrder,
        resetColumnsToggle,
        resetColumnsWidth,
    } = useDataTableColumns<T>({
        key: storeKey,
        columns: managedColumns,
        getInitialValueInEffect: false,
    });

    // The resize feature switches the table to `table-layout: fixed` (where CSS
    // min-width is ignored) and its own drag floor of 50px is low enough to clip
    // the sort/filter/hide controls out of a header. The width store is the only
    // input the table honours, so clamp it there. The debounce lets an
    // in-progress drag run unhindered — an undersized column snaps back to the
    // floor on release. Unmanaged (fixed-width) columns are exempt.
    useEffect(() => {
        const timer = setTimeout(() => {
            const updates: { accessor: string; width: string }[] = [];
            for (const entry of columnsWidth) {
                for (const [accessor, width] of Object.entries(entry)) {
                    if (!managedAccessors.has(accessor) || typeof width !== "string") continue;
                    const px = parseInt(width, 10);
                    if (!Number.isNaN(px) && px < minColumnWidth) {
                        updates.push({ accessor, width: `${minColumnWidth}px` });
                    }
                }
            }
            if (updates.length > 0) setColumnsWidth(updates);
        }, 200);
        return () => clearTimeout(timer);
        // `managedAccessors` is rebuilt each render from `meta`; keying the effect
        // on the stable store key instead keeps it from re-arming every render.
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [columnsWidth, setColumnsWidth, minColumnWidth, storeKey]);

    const visible = new Map(columnsToggle.map((c) => [c.accessor, c.toggled]));

    function setColumnVisible(accessor: string, isVisible: boolean) {
        setColumnsToggle(columnsToggle.map((c) => (c.accessor === accessor ? { ...c, toggled: isVisible } : c)));
    }

    function resetColumns() {
        resetColumnsOrder();
        resetColumnsToggle();
        resetColumnsWidth();
    }

    return {
        effectiveColumns,
        /** Accessor → currently visible. Mandatory columns are always shown regardless. */
        columnVisible: visible,
        setColumnVisible,
        resetColumns,
    };
}
