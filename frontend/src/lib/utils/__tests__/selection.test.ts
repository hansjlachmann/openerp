import { describe, it, expect } from 'vitest';
import {
	emptySelection,
	singleSelection,
	selectionCount,
	isMultiSelection,
	isSelected,
	toggleKey,
	rangeKeys,
	selectRange,
	selectAll
} from '../selection';

describe('list selection', () => {
	it('starts empty and holds one record after a click', () => {
		expect(selectionCount(emptySelection(), 10)).toBe(0);
		const one = singleSelection('C01', 0);
		expect(selectionCount(one, 10)).toBe(1);
		expect(isMultiSelection(one, 10)).toBe(false);
		expect(isSelected(one, 'C01')).toBe(true);
		expect(isSelected(one, 'C02')).toBe(false);
	});

	it('Ctrl+click adds and removes records, keeping the cursor row', () => {
		let sel = emptySelection();
		sel = toggleKey(sel, 'C05', 4, 'C01'); // C01 was the cursor row
		expect([...sel.keys].sort()).toEqual(['C01', 'C05']);
		expect(sel.anchor).toBe(4);
		sel = toggleKey(sel, 'C01', 0);
		expect([...sel.keys]).toEqual(['C05']);
		expect(isMultiSelection(sel, 10)).toBe(false);
	});

	it('Ctrl+click on an all-selection removes one record from all', () => {
		const all = selectAll(singleSelection('C01', 0));
		const sel = toggleKey(all, 'C02', 1, null, ['C01', 'C02', 'C03']);
		expect(sel.all).toBe(false);
		expect([...sel.keys]).toEqual(['C01', 'C03']);
	});

	it('takes inclusive ranges in either direction, across the loaded window', () => {
		const window = ['C11', 'C12', 'C13', 'C14']; // rows 10..13
		expect(rangeKeys(window, 10, 11, 13)).toEqual(['C12', 'C13', 'C14']);
		expect(rangeKeys(window, 10, 13, 11)).toEqual(['C12', 'C13', 'C14']);
		expect(rangeKeys(window, 10, 5, 10)).toEqual(['C11']); // rows before the window left out
		expect(rangeKeys(window, 10, 20, 30)).toEqual([]);
		const sel = selectRange(11, rangeKeys(window, 10, 11, 12));
		expect(sel.anchor).toBe(11);
		expect([...sel.keys]).toEqual(['C12', 'C13']);
	});

	it('selects every record with Ctrl+A, counted from the list total', () => {
		const all = selectAll(emptySelection());
		expect(selectionCount(all, 1250)).toBe(1250);
		expect(isSelected(all, 'anything')).toBe(true);
		expect(isMultiSelection(all, 1)).toBe(false); // a one-record list
	});
});
