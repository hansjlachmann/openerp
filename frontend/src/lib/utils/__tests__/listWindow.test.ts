import { describe, it, expect } from 'vitest';
import {
	MAX_WINDOW,
	estimateRowsPerPage,
	windowSize,
	windowOffsetFor,
	isLastRecord,
	needsShift
} from '../listWindow';

describe('listWindow', () => {
	it('loads five pages, at most MAX_WINDOW rows', () => {
		expect(windowSize(30)).toBe(150);
		expect(windowSize(60)).toBe(MAX_WINDOW);
		expect(windowSize(0)).toBe(5);
	});

	it('estimates at least 10 rows per page', () => {
		expect(estimateRowsPerPage(1000)).toBe(33);
		expect(estimateRowsPerPage(200)).toBe(10);
	});

	it('places the target two pages into the window, clamped to the list', () => {
		// 10,000 rows, 30 per page: window of 150
		expect(windowOffsetFor(5000, 30, 10000)).toBe(4940);
		expect(windowOffsetFor(10, 30, 10000)).toBe(0);
		expect(windowOffsetFor(9999, 30, 10000)).toBe(9850);
		// list shorter than a window
		expect(windowOffsetFor(15, 30, 20)).toBe(0);
	});

	it('knows the last record of the whole list', () => {
		expect(isLastRecord(149, 9850, 10000)).toBe(true);
		expect(isLastRecord(149, 0, 10000)).toBe(false);
		expect(isLastRecord(19, 0, 20)).toBe(true);
		expect(isLastRecord(0, 0, 0)).toBe(true);
	});

	it('shifts only near an edge with more rows beyond it', () => {
		// window 0..149 of 10,000, 30 rows per page
		expect(needsShift(100, 30, 150, 0, 10000)).toBe(false);
		expect(needsShift(120, 30, 150, 0, 10000)).toBe(true); // last page of the window
		expect(needsShift(150, 30, 150, 0, 10000)).toBe(true); // past the window
		expect(needsShift(5, 30, 150, 0, 10000)).toBe(false); // start of the list
		expect(needsShift(-1, 30, 150, 0, 10000)).toBe(false);
		// window 4940..5089
		expect(needsShift(5, 30, 150, 4940, 10000)).toBe(true);
		expect(needsShift(-1, 30, 150, 4940, 10000)).toBe(true);
		// whole list loaded: never shift
		expect(needsShift(19, 30, 20, 0, 20)).toBe(false);
		expect(needsShift(20, 30, 20, 0, 20)).toBe(false);
	});
});
