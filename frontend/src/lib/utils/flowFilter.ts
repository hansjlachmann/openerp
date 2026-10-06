// FlowFilter expressions (BC "Filter totals by", e.g. Date Filter). The user types BC syntax
// with dates in their own format — 01.01.26..31.03.26, ..31.03.26, 01.01.26.., 15.05.26,
// alternatives with |, <>date, t = today — and the API expects ISO dates. Other kinds
// (Boolean, Integer, Text, Code) are sent as typed; the backend validates them.

import { parseLocaleDate } from './fieldHelpers';

export interface FlowFilterResult {
	value?: string; // expression for the API ('' = no filter)
	error?: string; // the part that could not be read
}

function isoDate(d: Date): string {
	return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}

// ISO date from year/month/day if that day exists (31.02 does not)
function validIso(y: number, m: number, d: number): string | null {
	if (y < 100) y += 2000;
	const check = new Date(y, m - 1, d);
	if (check.getFullYear() !== y || check.getMonth() !== m - 1 || check.getDate() !== d) return null;
	return `${String(y).padStart(4, '0')}-${String(m).padStart(2, '0')}-${String(d).padStart(2, '0')}`;
}

// One date of the expression in ISO form, or null if it is not a valid date.
// - t = today; ISO (2026-04-01) always works.
// - With dots it is day.month[.year] whatever the UI language — dots are only used for
//   day-first dates (nb, da, de); 01.04 is this year (BC).
// - With slashes/dashes the order of the user's language (en-US 04/01/26 = 1 April).
function dateToken(token: string, locale: string, today: Date): string | null {
	if (token.toLowerCase() === 't') return isoDate(today);
	if (/^\d{4}-\d{1,2}-\d{1,2}$/.test(token)) {
		const [y, m, d] = token.split('-').map(Number);
		return validIso(y, m, d);
	}
	const dotted = token.match(/^(\d{1,2})\.(\d{1,2})(?:\.(\d{2}|\d{4}))?$/);
	if (dotted) {
		const year = dotted[3] ? Number(dotted[3]) : today.getFullYear();
		return validIso(year, Number(dotted[2]), Number(dotted[1]));
	}
	if (!/^\d{1,2}[/-]\d{1,2}[/-](\d{2}|\d{4})$/.test(token)) return null;
	const iso = parseLocaleDate(token.replace(/[/-]/g, localeSeparator(locale)), locale);
	if (!iso) return null;
	const [y, m, d] = iso.split('-').map(Number);
	return validIso(y, m, d);
}

// The date separator of the locale's numeric dates ("/" for en-US, "." for nb-NO)
function localeSeparator(locale: string): string {
	try {
		const parts = new Intl.DateTimeFormat(locale, { year: 'numeric', month: '2-digit', day: '2-digit' }).formatToParts(new Date(2026, 0, 15));
		return parts.find((p) => p.type === 'literal')?.value.trim() || '/';
	} catch {
		return '/';
	}
}

// Hint for a date FlowFilter field in the user's format, e.g. "DD.MM.YYYY..DD.MM.YYYY"
export function dateFilterHint(pattern: string): string {
	return `${pattern}..${pattern}`;
}

// The API form of a FlowFilter expression of the given kind ("date", "bool", "int", "text",
// "code"), typed in the user's locale.
export function toApiFlowFilter(expr: string, kind: string, locale: string, today: Date = new Date()): FlowFilterResult {
	const trimmed = expr.trim();
	if (trimmed === '' || kind !== 'date') return { value: trimmed };

	const parts: string[] = [];
	for (const raw of trimmed.split('|')) {
		let part = raw.trim();
		let prefix = '';
		if (part.startsWith('<>')) {
			prefix = '<>';
			part = part.slice(2).trim();
		}
		const range = part.split('..');
		if (range.length > 2) return { error: raw.trim() };
		const converted: string[] = [];
		for (const token of range.map((s) => s.trim())) {
			if (token === '' && range.length === 2) {
				converted.push(''); // open range end
				continue;
			}
			const iso = dateToken(token, locale, today);
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
