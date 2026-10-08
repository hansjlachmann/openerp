import { describe, it, expect } from 'vitest';
import { navigationQuery, withNavigationQuery, parseNavigationQuery } from '../recordNavigation';

describe('navigationQuery', () => {
	it('keeps only the parts in effect', () => {
		expect(navigationQuery({})).toBeUndefined();
		expect(navigationQuery({ filters: [], search: '  ', search_fields: ['name'], sort_order: 'asc' })).toBeUndefined();
		expect(navigationQuery({ search: 'oslo', search_fields: [] })).toBeUndefined();
		expect(navigationQuery({ search: ' oslo ', search_fields: ['name'], sort_by: 'name', sort_order: 'desc' })).toEqual({
			search: 'oslo',
			search_fields: ['name'],
			sort_by: 'name',
			sort_order: 'desc'
		});
	});
});

describe('withNavigationQuery / parseNavigationQuery', () => {
	it('round-trips the list query through the card URL', () => {
		const q = { filters: [{ field: 'no', expression: 'C1..C9' }], sort_by: 'name' };
		const url = withNavigationQuery('/pages/21/C00010', q);
		expect(url.startsWith('/pages/21/C00010?nav=')).toBe(true);
		expect(parseNavigationQuery(new URL(url, 'http://x').searchParams.get('nav'))).toEqual(q);
	});
	it('leaves the URL alone without a query', () => {
		expect(withNavigationQuery('/pages/21/C1', undefined)).toBe('/pages/21/C1');
		expect(withNavigationQuery('/pages/21/C1', { filters: [] })).toBe('/pages/21/C1');
	});
	it('ignores malformed values', () => {
		expect(parseNavigationQuery('not json')).toBeUndefined();
		expect(parseNavigationQuery('[1,2]')).toBeUndefined();
		expect(parseNavigationQuery('{"filters":[{"field":1}],"sort_by":5}')).toBeUndefined();
		expect(parseNavigationQuery(null)).toBeUndefined();
	});
});
