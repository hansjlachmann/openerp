import { describe, it, expect, beforeEach } from 'vitest';

import { saveClipboard, loadClipboard, keyValuesFromFilters, recordsToPaste, singleValueFilters, CLIPBOARD_KEY } from '../recordClipboard';

describe('record clipboard', () => {
	beforeEach(() => sessionStorage.clear());

	it('keeps copied records in the browser tab', () => {
		expect(loadClipboard()).toBeNull();
		saveClipboard({ table: 'User_Member', company: 'demo01', records: [{ user_id: 'HANS', role_id: 'READER' }] });
		expect(loadClipboard()).toEqual({ table: 'User_Member', company: 'demo01', records: [{ user_id: 'HANS', role_id: 'READER' }] });
		sessionStorage.setItem(CLIPBOARD_KEY, '{broken');
		expect(loadClipboard()).toBeNull();
	});

	it('takes new key values only from exact filters on key fields', () => {
		const keys = ['user_id', 'role_id', 'company'];
		expect(
			keyValuesFromFilters(
				[
					{ field: 'role_id', expression: 'CLERK' },
					{ field: 'company', expression: 'demo01|demo02' }, // alternatives: no single value
					{ field: 'user_id', expression: 'H*' }, // wildcard
					{ field: 'name', expression: 'Anna' }, // not a key field
					{ field: 'user_id', expression: ' ' }
				],
				keys
			)
		).toEqual({ role_id: 'CLERK' });
	});

	it('applies the key values to the records to paste', () => {
		const copied = [
			{ user_id: 'HANS', role_id: 'READER', company: '' },
			{ user_id: 'ANNA', role_id: 'READER', company: '' }
		];
		expect(recordsToPaste(copied, { role_id: 'CLERK' })).toEqual([
			{ user_id: 'HANS', role_id: 'CLERK', company: '' },
			{ user_id: 'ANNA', role_id: 'CLERK', company: '' }
		]);
		expect(copied[0].role_id).toBe('READER'); // the clipboard itself is unchanged
	});
});

describe('singleValueFilters', () => {
	it('takes the filters on one exact value', () => {
		expect(
			singleValueFilters([
				{ field: 'currency_code', expression: 'EUR' },
				{ field: 'starting_date', expression: '01.01.26..' },
				{ field: 'code', expression: 'A|B' },
				{ field: 'name', expression: 'Ac*' },
				{ field: 'amount', expression: '>10' },
				{ field: 'blank', expression: ' ' }
			])
		).toEqual({ currency_code: 'EUR' });
	});

	it('only for the fields it accepts', () => {
		expect(singleValueFilters([{ field: 'a', expression: '1' }, { field: 'b', expression: '2' }], (f) => f === 'b')).toEqual({ b: '2' });
	});
});
