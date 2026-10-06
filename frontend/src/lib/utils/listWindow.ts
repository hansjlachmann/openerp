// Windowed list loading: a list page holds only a window of rows — the rows that fit
// on the page plus two pages above and two below (at most MAX_WINDOW) — and fetches
// another window from the server when the user moves past its edge. Row indexes in
// ListPage are positions in the loaded window; offset + index is the position in the
// whole (filtered, searched, sorted) list of `total` records.

export const MAX_WINDOW = 200;
export const PAGES_AROUND = 2; // pages kept above and below the visible page

// Rows that fit on the visible page when the list is not rendered yet. The list fills
// the viewport minus menu bar, toolbar and header (ListPage: height calc(100vh - 180px)).
export function estimateRowsPerPage(viewportHeight: number): number {
	const ROW_HEIGHT = 22;
	const CHROME = 260;
	return Math.max(10, Math.floor((viewportHeight - CHROME) / ROW_HEIGHT));
}

// Number of rows to load: the visible page plus PAGES_AROUND pages on each side.
export function windowSize(rowsPerPage: number): number {
	return Math.min(MAX_WINDOW, Math.max(1, rowsPerPage) * (2 * PAGES_AROUND + 1));
}

// Offset of the window that shows absolute row `target` with PAGES_AROUND pages
// before it (fewer at the start or end of the list).
export function windowOffsetFor(target: number, rowsPerPage: number, total: number): number {
	const size = windowSize(rowsPerPage);
	const start = target - PAGES_AROUND * Math.max(1, rowsPerPage);
	return Math.max(0, Math.min(start, total - size));
}

// True when window position `index` is the last record of the whole list.
export function isLastRecord(index: number, offset: number, total: number): boolean {
	return offset + index >= total - 1;
}

// True when moving to window position `index` needs a new window: the target is
// outside the loaded rows, or within one page of an edge beyond which more rows exist.
export function needsShift(
	index: number,
	rowsPerPage: number,
	windowLength: number,
	offset: number,
	total: number
): boolean {
	if (index < 0) return offset > 0;
	if (index >= windowLength) return offset + windowLength < total;
	const page = Math.max(1, rowsPerPage);
	const moreAbove = offset > 0;
	const moreBelow = offset + windowLength < total;
	return (moreAbove && index < page) || (moreBelow && index >= windowLength - page);
}

// A list page asks its parent (PageRenderer) for another window of rows. Unset
// fields keep their current value; search and sort changes start at offset 0.
export interface ListWindowRequest {
	offset?: number;
	limit?: number;
	search?: string;
	sort?: { field: string; direction: 'asc' | 'desc' } | null;
}
