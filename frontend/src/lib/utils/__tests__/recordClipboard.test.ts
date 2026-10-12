import { describe, it, expect, beforeEach } from 'vitest';

import { saveClipboard, loadClipboard, keyValuesFromFilters, recordsToPaste, CLIPBOARD_KEY } from '../recordClipboard';

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
