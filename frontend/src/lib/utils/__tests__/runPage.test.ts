import { describe, it, expect } from 'vitest';
import { runPagePath } from '../runPage';

describe('runPagePath', () => {
	it('filters on one field of the record', () => {
		expect(runPagePath({ run_page: 25, run_page_filter_field: 'customer_no', run_page_filter_value: 'no' }, { no: 'C1' })).toBe(
			'/pages/25?filter=customer_no%3DC1'
		);
	});
	it('filters on several fields and constants', () => {
		const action = {
			run_page: 433,
			run_page_filters: [
				{ field: 'reminder_terms_code', value: 'reminder_terms_code' },
				{ field: 'reminder_level', value: 'no' },
				{ field: 'position', const: '0' }
			]
		};
		expect(runPagePath(action, { reminder_terms_code: 'DOM', no: 2 })).toBe(
			'/pages/433?filter=reminder_terms_code%3DDOM&filter=reminder_level%3D2&filter=position%3D0'
		);
	});
	it('nothing without the record value; no filter at all is the page', () => {
		expect(runPagePath({ run_page: 25, run_page_filter_field: 'customer_no', run_page_filter_value: 'no' }, undefined)).toBeNull();
		expect(runPagePath({ run_page: 9 }, undefined)).toBe('/pages/9');
	});
});
