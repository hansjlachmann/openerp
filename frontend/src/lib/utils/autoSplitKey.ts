// AutoSplitKey: document lines are numbered in steps of 10000 (the last primary key field);
// a line inserted after another one gets the number halfway to the next line, so lines can be
// put between existing ones without renumbering.

export const LINE_NO_STEP = 10000;

/**
 * The line number for a new line after `after` (undefined: at the start) and before `before`
 * (undefined: at the end): 10000 steps at the end, halfway between two lines. null when there
 * is no whole number left between them (the lines must be renumbered first).
 */
export function splitLineNo(after: number | undefined, before: number | undefined): number | null {
	const lower = after ?? 0;
	if (before === undefined) return lower + LINE_NO_STEP;
	const middle = Math.floor((lower + before) / 2);
	return middle > lower && middle < before ? middle : null;
}

/** A field value as a line number (undefined when it is not one) */
export function lineNo(value: unknown): number | undefined {
	const n = typeof value === 'number' ? value : Number(value);
	return Number.isFinite(n) && String(value).trim() !== '' ? n : undefined;
}

/**
 * The line number of a new line inserted after the row at `index` (none selected: after the
 * last line) — from every row that has a number, also rows still being saved (Enter on the last
 * line starts the next one before the insert has returned). null when there is no room.
 */
export function newLineNo(rows: Record<string, unknown>[], index: number, field: string): number | null {
	const numbered = (r: Record<string, unknown>) => (lineNo(r[field]) ?? 0) > 0;
	const anchor = index >= 0 ? rows[index] : undefined;
	if (anchor && numbered(anchor)) {
		const next = rows.slice(index + 1).find(numbered);
		return splitLineNo(lineNo(anchor[field]), next ? lineNo(next[field]) : undefined);
	}
	const nos = rows.filter(numbered).map((r) => lineNo(r[field]) as number);
	return splitLineNo(nos.length ? Math.max(...nos) : undefined, undefined);
}
