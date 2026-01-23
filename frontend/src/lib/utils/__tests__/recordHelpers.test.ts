import { describe, it, expect } from 'vitest';
import {
	getRecordId,
	isNewRecord,
	getRecordKey,
	deepCopy,
	hasRecordChanged
} from '../recordHelpers';

describe('getRecordId', () => {
	it('returns undefined for null record', () => {
		expect(getRecordId(null)).toBeUndefined();
	});

	it('returns undefined for undefined record', () => {
		expect(getRecordId(undefined)).toBeUndefined();
	});

	it('returns value from specified primary key field', () => {
		const record = { customer_no: 'CUST-001', name: 'Test' };
		expect(getRecordId(record, 'customer_no')).toBe('CUST-001');
	});

	it('converts number to string', () => {
		const record = { id: 123 };
		expect(getRecordId(record, 'id')).toBe('123');
	});

	it('falls back to "no" field', () => {
		const record = { no: 'ABC-001', name: 'Test' };
		expect(getRecordId(record)).toBe('ABC-001');
	});

	it('falls back to "code" field', () => {
		const record = { code: 'CODE-001', name: 'Test' };
		expect(getRecordId(record)).toBe('CODE-001');
	});

	it('falls back to "user_id" field', () => {
		const record = { user_id: 'USER-001', name: 'Test' };
		expect(getRecordId(record)).toBe('USER-001');
	});

	it('falls back to "id" field', () => {
		const record = { id: 'ID-001', name: 'Test' };
		expect(getRecordId(record)).toBe('ID-001');
	});

	it('returns undefined for empty primary key', () => {
		const record = { no: '', name: 'Test' };
		expect(getRecordId(record)).toBeUndefined();
	});
});

describe('isNewRecord', () => {
	it('returns true for null record', () => {
		expect(isNewRecord(null)).toBe(true);
	});

	it('returns true for record without ID', () => {
		const record = { name: 'Test' };
		expect(isNewRecord(record)).toBe(true);
	});

	it('returns false for record with ID', () => {
		const record = { no: 'ABC-001', name: 'Test' };
		expect(isNewRecord(record)).toBe(false);
	});
});

describe('getRecordKey', () => {
	it('returns _tempId if present', () => {
		const record = { _tempId: 'temp-123', no: 'ABC-001' };
		expect(getRecordKey(record)).toBe('temp-123');
	});

	it('returns primary key value', () => {
		const record = { no: 'ABC-001', name: 'Test' };
		expect(getRecordKey(record)).toBe('ABC-001');
	});

	it('returns fallback for record without ID', () => {
		const record = { name: 'Test' };
		const key = getRecordKey(record);
		expect(key).toMatch(/^record-/);
	});
});

describe('deepCopy', () => {
	it('creates a deep copy of object', () => {
		const original = { a: 1, b: { c: 2 } };
		const copy = deepCopy(original);

		expect(copy).toEqual(original);
		expect(copy).not.toBe(original);
		expect(copy.b).not.toBe(original.b);
	});

	it('handles arrays', () => {
		const original = [1, 2, { a: 3 }];
		const copy = deepCopy(original);

		expect(copy).toEqual(original);
		expect(copy).not.toBe(original);
	});
});

describe('hasRecordChanged', () => {
	it('returns false for identical records', () => {
		const record = { no: 'ABC', name: 'Test', value: 100 };
		expect(hasRecordChanged(record, record)).toBe(false);
	});

	it('returns true when string field changed', () => {
		const current = { no: 'ABC', name: 'Changed' };
		const original = { no: 'ABC', name: 'Original' };
		expect(hasRecordChanged(current, original)).toBe(true);
	});

	it('returns true when number field changed', () => {
		const current = { no: 'ABC', value: 200 };
		const original = { no: 'ABC', value: 100 };
		expect(hasRecordChanged(current, original)).toBe(true);
	});

	it('handles number/string comparison', () => {
		const current = { no: 'ABC', value: '100' };
		const original = { no: 'ABC', value: 100 };
		expect(hasRecordChanged(current, original)).toBe(false);
	});

	it('treats null and empty string as equivalent', () => {
		const current = { no: 'ABC', name: '' };
		const original = { no: 'ABC', name: null };
		expect(hasRecordChanged(current, original)).toBe(false);
	});

	it('ignores fields starting with underscore', () => {
		const current = { no: 'ABC', _internal: 'changed' };
		const original = { no: 'ABC', _internal: 'original' };
		expect(hasRecordChanged(current, original)).toBe(false);
	});
});
