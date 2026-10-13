// Multi-row selection on list pages (select several lines with Ctrl/Shift, Ctrl+A).
//
// Rows are identified by their record key (getRecordId), never by their position: lists load
// a window of rows, and search/sort reorder them. Positions are absolute (window offset +
// row) and only used for Shift ranges, between the anchor and the target row.

export interface ListSelection {
	/** Selected records by key (unused when all is set) */
	keys: Set<string>;
	/** Every record of the list (Ctrl+A), also those not loaded */
	all: boolean;
	/** Absolute row a Shift range starts from */
	anchor: number | null;
}

export function emptySelection(): ListSelection {
	return { keys: new Set(), all: false, anchor: null };
}

/** One record selected — the plain click / arrow key state */
export function singleSelection(key: string | null | undefined, row: number | null): ListSelection {
	return { keys: key ? new Set([key]) : new Set(), all: false, anchor: row };
}

/** Number of selected records; total is the number of records in the list */
export function selectionCount(sel: ListSelection, total: number): number {
	return sel.all ? total : sel.keys.size;
}

/** More than one record selected: actions like Delete act on the selection */
export function isMultiSelection(sel: ListSelection, total: number): boolean {
	return selectionCount(sel, total) > 1;
}

export function isSelected(sel: ListSelection, key: string | null | undefined): boolean {
	return !!key && (sel.all || sel.keys.has(key));
}

/**
 * Ctrl+click: add or remove one record. current is the record that was selected before (the
 * cursor row), which counts as selected when nothing else is. allKeys resolves an
 * all-selection into keys first (only needed when sel.all).
 */
export function toggleKey(
	sel: ListSelection,
	key: string,
	row: number,
	current?: string | null,
	allKeys?: string[]
): ListSelection {
	const keys = sel.all ? new Set(allKeys ?? []) : new Set(sel.keys);
	if (!sel.all && keys.size === 0 && current) keys.add(current);
	if (keys.has(key)) keys.delete(key);
	else keys.add(key);
	return { keys, all: false, anchor: row };
}

/**
 * Inclusive range of rows from..to (either order) out of keysInOrder, the keys of the rows
 * starting at absolute row offset (the loaded window, or all keys with offset 0). Rows
 * outside keysInOrder are left out.
 */
export function rangeKeys(keysInOrder: string[], offset: number, from: number, to: number): string[] {
	const lo = Math.max(Math.min(from, to) - offset, 0);
	const hi = Math.min(Math.max(from, to) - offset, keysInOrder.length - 1);
	return lo > hi ? [] : keysInOrder.slice(lo, hi + 1);
}

/** Shift+click / Shift+arrow: the range from the anchor to the target replaces the selection */
export function selectRange(anchor: number, keysInRange: string[]): ListSelection {
	return { keys: new Set(keysInRange), all: false, anchor };
}

/** Ctrl+A: every record of the list */
export function selectAll(sel: ListSelection): ListSelection {
	return { keys: new Set(), all: true, anchor: sel.anchor };
}
