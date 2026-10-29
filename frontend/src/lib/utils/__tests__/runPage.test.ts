import { describe, it, expect } from 'vitest';
import { runPagePath, openPagePath } from '../runPage';

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
	it('filters on the range of two fields (the entries of a register)', () => {
		const action = { run_page: 20, run_page_filters: [{ field: 'entry_no', from: 'from_entry_no', to: 'to_entry_no' }] };
		expect(runPagePath(action, { from_entry_no: 5, to_entry_no: 9 })).toBe('/pages/20?filter=entry_no%3D5..9');
		expect(runPagePath(action, { from_entry_no: 5 })).toBeNull();
	});
	it('nothing without the record value; no filter at all is the page', () => {
		expect(runPagePath({ run_page: 25, run_page_filter_field: 'customer_no', run_page_filter_value: 'no' }, undefined)).toBeNull();
		expect(runPagePath({ run_page: 9 }, undefined)).toBe('/pages/9');
	});
	it('opens the page in a field of the record, optional filters left out when blank (Navigate Show)', () => {
		const action = {
			run_page_field: 'page_id',
			run_page_filters: [
				{ field: 'document_no', value: 'document_no_filter', optional: true },
				{ field: 'posting_date', value: 'posting_date_filter', optional: true }
			]
		};
		expect(runPagePath(action, { page_id: 25, document_no_filter: 'G00012', posting_date_filter: '' })).toBe(
			'/pages/25?filter=document_no%3DG00012'
		);
		expect(runPagePath(action, { page_id: 0 })).toBeNull();
		expect(runPagePath(action, undefined)).toBeNull();
	});
	it('sets page parameters from the record (Navigate from an entry)', () => {
		const action = { run_page: 344, run_page_params: { document_no: 'document_no', posting_date: 'posting_date' } };
		expect(runPagePath(action, { document_no: 'G 1/2', posting_date: '2026-01-15' })).toBe(
			'/pages/344?document_no=G%201%2F2&posting_date=2026-01-15'
		);
		expect(runPagePath(action, { document_no: 'G1', posting_date: '' })).toBe('/pages/344?document_no=G1');
	});
});

describe('openPagePath', () => {
	it('is the card of the record a result opens', () => {
		expect(openPagePath({ page_id: 42, key: '1,SO000001' })).toBe('/pages/42/1%2CSO000001');
		// a page with parameters (Posting Preview)
		expect(openPagePath({ page_id: 115, key: '', params: { preview: 'ab12' } })).toBe('/pages/115?preview=ab12');
		expect(openPagePath({ page_id: 9305 })).toBe('/pages/9305');
	});
	it('is null without a page', () => {
		expect(openPagePath(undefined)).toBeNull();
		expect(openPagePath({ key: 'x' })).toBeNull();
		expect(openPagePath('42')).toBeNull();
	});
});
