import { describe, expect, it } from 'vitest';
import { pageIndex } from '../dropdownPaging';

describe('pageIndex', () => {
	it('moves a page down and up', () => {
		expect(pageIndex(0, 100, 10, 1)).toBe(10);
		expect(pageIndex(25, 100, 10, -1)).toBe(15);
	});
	it('stops at the first and last row', () => {
		expect(pageIndex(95, 100, 10, 1)).toBe(99);
		expect(pageIndex(4, 100, 10, -1)).toBe(0);
	});
	it('starts from the edge without a highlight', () => {
		expect(pageIndex(-1, 100, 10, 1)).toBe(9);
		expect(pageIndex(-1, 100, 10, -1)).toBe(90);
	});
	it('has nothing to highlight in an empty list', () => {
		expect(pageIndex(0, 0, 10, 1)).toBe(-1);
	});
});
