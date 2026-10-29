// Date formulas — "30D", "1M+CM", "-CY", "WD5" — applied to a date: the same rules as the
// server's (backend/foundation/dateformula), for the date entry's formulas (+1W, CM …).

/** A date (local, midnight) from year, month (1-12) and day */
function ymd(y: number, m: number, d: number): Date {
	return new Date(y, m - 1, d);
}

/** Monday 1 … Sunday 7 */
function isoWeekday(d: Date): number {
	return d.getDay() === 0 ? 7 : d.getDay();
}

/** Adds months, keeping the last day of a month that is too short */
function addMonths(d: Date, months: number): Date {
	const first = new Date(d.getFullYear(), d.getMonth() + months, 1);
	const last = new Date(first.getFullYear(), first.getMonth() + 1, 0).getDate();
	return new Date(first.getFullYear(), first.getMonth(), Math.min(d.getDate(), last));
}

/** The end (sign +) or start (sign -) of the current day, week, month, quarter or year */
function currentPeriod(d: Date, unit: string, sign: number): Date | null {
	switch (unit) {
		case 'D':
			return d;
		case 'W': {
			const monday = new Date(d.getFullYear(), d.getMonth(), d.getDate() + 1 - isoWeekday(d));
			return sign < 0 ? monday : new Date(monday.getFullYear(), monday.getMonth(), monday.getDate() + 6);
		}
		case 'M':
		case 'Q':
		case 'Y': {
			const months = { M: 1, Q: 3, Y: 12 }[unit];
			const startMonth = Math.floor(d.getMonth() / months) * months;
			const start = new Date(d.getFullYear(), startMonth, 1);
			return sign < 0 ? start : new Date(start.getFullYear(), start.getMonth() + months, 0);
		}
	}
	return null;
}

/** The date a formula gives from a date; null for a formula that cannot be read */
export function applyDateFormula(formula: string, date: Date): Date | null {
	let f = formula.trim().replace(/ /g, '').toUpperCase();
	if (f.startsWith('<') && f.endsWith('>')) f = f.slice(1, -1);
	if (f === '') return null;
	let d = ymd(date.getFullYear(), date.getMonth() + 1, date.getDate());
	let i = 0;
	while (i < f.length) {
		let sign = 1;
		if (f[i] === '+' || f[i] === '-') {
			if (f[i] === '-') sign = -1;
			i++;
		}
		if (f[i] === 'C') {
			i++;
			if (i >= f.length) return null;
			const next = currentPeriod(d, f[i], sign);
			if (!next) return null;
			d = next;
			i++;
		} else if (f.startsWith('WD', i)) {
			if (sign < 0) return null;
			i += 2;
			let j = i;
			while (j < f.length && f[j] >= '0' && f[j] <= '9') j++;
			const n = Number(f.slice(i, j));
			if (j === i || n < 1 || n > 7) return null;
			i = j;
			d = new Date(d.getFullYear(), d.getMonth(), d.getDate() + ((n - isoWeekday(d) + 7) % 7));
		} else {
			let j = i;
			while (j < f.length && f[j] >= '0' && f[j] <= '9') j++;
			if (j === i || j >= f.length) return null;
			const n = Number(f.slice(i, j)) * sign;
			switch (f[j]) {
				case 'D':
					d = new Date(d.getFullYear(), d.getMonth(), d.getDate() + n);
					break;
				case 'W':
					d = new Date(d.getFullYear(), d.getMonth(), d.getDate() + 7 * n);
					break;
				case 'M':
					d = addMonths(d, n);
					break;
				case 'Q':
					d = addMonths(d, 3 * n);
					break;
				case 'Y':
					d = addMonths(d, 12 * n);
					break;
				default:
					return null;
			}
			i = j + 1;
		}
	}
	return d;
}
