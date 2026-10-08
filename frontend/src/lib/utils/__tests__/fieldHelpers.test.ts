import { describe, it, expect } from 'vitest';
import { isAdvancedLookup, formatLookupValue, isCodeType, toDateTimeInput, fromDateTimeInput } from '../fieldHelpers';

describe('isAdvancedLookup', () => {
	const columns = [{ source: 'code' }];
	it('is a table-style dropdown with columns and rows, or columns and a lazy URL', () => {
		expect(isAdvancedLookup({ columns, rows: [{ _key: 'A' }] })).toBe(true);
		expect(isAdvancedLookup({ columns, lazy_url: '/api/tables/X/lookup/f' })).toBe(true);
		expect(isAdvancedLookup({ columns, rows: null, lazy_url: '/api/tables/X/lookup/f' })).toBe(true);
	});
	it('is not without columns, or without rows and lazy URL', () => {
		expect(isAdvancedLookup(undefined)).toBe(false);
		expect(isAdvancedLookup({ columns: [], rows: [{ _key: 'A' }] })).toBe(false);
		expect(isAdvancedLookup({ columns, rows: [] })).toBe(false);
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
