// FlowFilter expressions ("Filter totals by", e.g. Date Filter). The user types syntax
// with dates in their own format — 01.01.26..31.03.26, ..31.03.26, 01.01.26.., 15.05.26,
// alternatives with |, <>date, t = today — and the API expects ISO dates. Other kinds
// (Boolean, Integer, Text, Code) are sent as typed; the backend validates them.

import { dateEntryToIso } from './dateEntry';
import { getWorkDate } from './regional';

export interface FlowFilterResult {
	value?: string; // expression for the API ('' = no filter)
	error?: string; // the part that could not be read
}

// One date of the expression in ISO form, or null if it is not a valid date: what a date field
// takes (utils/dateEntry.ts) — short entries from the work date (05, 0509), t, w, weekdays, date
// formulas, ISO. With dots it is day.month[.year] whatever the order (dots are only used for
// day-first dates); otherwise the user's order (My Settings, else the language's).
function dateToken(token: string, locale: string, today: Date, workDate: Date): string | null {
	const order = token.includes('.') && !/^\d{4}/.test(token) ? 'DMY' : undefined;
	return dateEntryToIso(token, { locale, order, today, workDate });
}

// Hint for a date FlowFilter field in the user's format, e.g. "DD.MM.YYYY..DD.MM.YYYY"
export function dateFilterHint(pattern: string): string {
	return `${pattern}..${pattern}`;
}

// The API form of a FlowFilter expression of the given kind ("date", "bool", "int", "text",
// "code"), typed in the user's locale.
export function toApiFlowFilter(expr: string, kind: string, locale: string, today?: Date): FlowFilterResult {
	// short entries are filled from the work date (a test passes its own today for both)
	const workDate = today ?? getWorkDate();
	const now = today ?? new Date();
	const trimmed = expr.trim();
	if (trimmed === '' || kind !== 'date') return { value: trimmed };

	const parts: string[] = [];
	for (const raw of trimmed.split('|')) {
		let part = raw.trim();
		let prefix = '';
		const op = /^(<>|<=|>=|<|>)/.exec(part);
		if (op) {
			prefix = op[1];
			part = part.slice(prefix.length).trim();
		}
		const range = part.split('..');
		if (range.length > 2) return { error: raw.trim() };
		const converted: string[] = [];
		for (const token of range.map((s) => s.trim())) {
			if (token === '' && range.length === 2) {
				converted.push(''); // open range end
				continue;
			}
			const iso = dateToken(token, locale, now, workDate);
			if (!iso) return { error: token || raw.trim() };
			converted.push(iso);
		}
		if (range.length === 2 && converted[0] === '' && converted[1] === '') return { error: raw.trim() };
		parts.push(prefix + converted.join('..'));
	}
	return { value: parts.join('|') };
}

// The API form of a list's FlowFilters (as typed in the filter pane), for the page's
// FlowFilter fields; filters that cannot be read are left out (the filter pane rejects them).
export function apiFlowFilters(
	flowFilters: Array<{ field: string; expression: string }>,
	fields: Array<{ name: string; kind: string }> | undefined,
	locale: string
): Array<{ field: string; expression: string }> {
	const out: Array<{ field: string; expression: string }> = [];
	for (const f of flowFilters) {
		const kind = fields?.find((x) => x.name === f.field)?.kind ?? 'text';
		const result = toApiFlowFilter(f.expression, kind, locale);
		if (result.value) out.push({ field: f.field, expression: result.value });
	}
	return out;
}

// The API form of a list's filters: a date field's expression with its dates as ISO (typed in
// the user's format, short entries, t, w …); an expression that cannot be read goes as typed
// (the server tells); other fields as typed
export function apiFieldFilters<T extends { field: string; expression: string }>(filters: T[], isDateField: (field: string) => boolean, locale: string): T[] {
	return filters.map((f) => {
		if (!isDateField(f.field)) return f;
		const result = toApiFlowFilter(f.expression, 'date', locale);
		return result.value !== undefined && result.error === undefined ? { ...f, expression: result.value } : f;
	});
}
