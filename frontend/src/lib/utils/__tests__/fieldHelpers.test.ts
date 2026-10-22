import { describe, it, expect } from 'vitest';
import type { LookupData } from '$lib/types/api';
import { isAdvancedLookup, lookupFor, formatLookupValue, isCodeType, toDateTimeInput, fromDateTimeInput } from '../fieldHelpers';

describe('isAdvancedLookup', () => {
	const columns = [{ source: 'code' }];
	it('is a table-style dropdown with columns and rows, or columns and a lazy URL', () => {
		expect(isAdvancedLookup({ columns, rows: [{ _key: 'A' }] })).toBe(true);
		expect(isAdvancedLookup({ columns, lazy_url: '/api/tables/X/lookup/f' })).toBe(true);
		expect(isAdvancedLookup({ columns, rows: null, lazy_url: '/api/tables/X/lookup/f' })).toBe(true);
	});
	it('is also for an empty related table (the dropdown shows no rows)', () => {
		expect(isAdvancedLookup({ columns, rows: [] })).toBe(true);
	});
	it('is not without columns, or without rows and lazy URL', () => {
		expect(isAdvancedLookup(undefined)).toBe(false);
		expect(isAdvancedLookup({ columns: [], rows: [{ _key: 'A' }] })).toBe(false);
		expect(isAdvancedLookup({ columns, rows: null })).toBe(false);
	});
});

describe('formatLookupValue', () => {
	it('shows the key when on-demand rows are not loaded', () => {
		expect(formatLookupValue('C00010', { columns: [{ source: 'no' }] })).toBe('C00010');
	});
});

describe('isCodeType', () => {
	it('recognizes Code fields as the page metadata (YAML) and the table API (Go) name them', () => {
		expect(isCodeType('types.Code')).toBe(true);
		expect(isCodeType('Code')).toBe(true);
		expect(isCodeType('code')).toBe(true);
	});
	it('is false for other types', () => {
		expect(isCodeType('types.Text')).toBe(false);
		expect(isCodeType(undefined)).toBe(false);
	});
});

describe('datetime-local input values', () => {
	it('shows an API datetime as local time and sends it back as UTC', () => {
		const iso = '2026-10-08T10:00:00Z';
		const local = toDateTimeInput(iso);
		expect(local).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/);
		expect(fromDateTimeInput(local)).toBe(iso);
	});

	it('is empty for no or an invalid value', () => {
		expect(toDateTimeInput('')).toBe('');
		expect(toDateTimeInput(undefined)).toBe('');
		expect(toDateTimeInput('not a date')).toBe('');
		expect(fromDateTimeInput('')).toBe('');
	});
});

describe('lookupFor', () => {
	const gl: LookupData = { columns: [{ source: 'no', width: 100 }], rows: [{ _key: '1000' }] };
	const cust: LookupData = { columns: [{ source: 'no', width: 100 }], rows: [{ _key: 'C1' }] };
	const lookups: Record<string, LookupData> = {
		payment_terms_code: gl,
		bal_account_no: { variant_field: 'bal_account_type', variants: { '0': gl, '1': cust } }
	};
	it('picks the variant by the record type, none for a type without a table', () => {
		expect(lookupFor(lookups, 'bal_account_no', { bal_account_type: 1 })).toBe(cust);
		expect(lookupFor(lookups, 'bal_account_no', { bal_account_type: 0 })).toBe(gl);
		expect(lookupFor(lookups, 'bal_account_no', { bal_account_type: 2 })).toBeUndefined();
		expect(lookupFor(lookups, 'payment_terms_code', {})).toBe(gl);
		expect(lookupFor(lookups, 'nothing', {})).toBeUndefined();
	});
});

describe('lookupFor with relation filters', () => {
	it('keeps the rows matching the record', () => {
		const levels: LookupData = {
			columns: [{ source: 'no', width: 60 }],
			rows: [
				{ _key: '1', no: 1, reminder_terms_code: 'DOM' },
				{ _key: '2', no: 2, reminder_terms_code: 'DOM' },
				{ _key: '1', no: 1, reminder_terms_code: 'FOR' }
			],
			filters: [{ column: 'reminder_terms_code', field: 'reminder_terms_code' }]
		};
		expect(lookupFor({ reminder_level: levels }, 'reminder_level', { reminder_terms_code: 'DOM' })?.rows?.length).toBe(2);
		expect(lookupFor({ reminder_level: levels }, 'reminder_level', { reminder_terms_code: 'FOR' })?.rows?.length).toBe(1);
	});
});
