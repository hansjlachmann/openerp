// PgUp / PgDn in an open dropdown (lookup and option fields): the highlight moves by the rows
// that fit in the dropdown's visible part, and stops at the first / last row

// The number of rows visible in the dropdown body (at least 1)
export function dropdownPageSize(body: HTMLElement | null | undefined, rowSelector: string): number {
	const row = body?.querySelector<HTMLElement>(rowSelector);
	if (!body || !row || row.offsetHeight <= 0) return 10;
	return Math.max(1, Math.floor(body.clientHeight / row.offsetHeight) - 1);
}

// The highlighted index one page down (+1) or up (-1) in a list of count rows
export function pageIndex(index: number, count: number, pageSize: number, direction: 1 | -1): number {
	if (count <= 0) return -1;
	const from = index < 0 ? (direction > 0 ? -1 : count) : index;
	return Math.min(count - 1, Math.max(0, from + direction * Math.max(1, pageSize)));
}
