import { describe, it, expect } from 'vitest';
import { splitLineNo, lineNo, newLineNo } from '../autoSplitKey';

describe('splitLineNo', () => {
	it('numbers lines in steps of 10000 at the end', () => {
		expect(splitLineNo(undefined, undefined)).toBe(10000);
		expect(splitLineNo(30000, undefined)).toBe(40000);
	});
	it('puts a line halfway between two lines', () => {
		expect(splitLineNo(10000, 20000)).toBe(15000);
		expect(splitLineNo(undefined, 10000)).toBe(5000);
		expect(splitLineNo(10000, 10003)).toBe(10001);
	});
	it('has no number left between neighbours', () => {
		expect(splitLineNo(10000, 10001)).toBeNull();
		expect(splitLineNo(10000, 10000)).toBeNull();
	});
});

describe('lineNo', () => {
	it('reads numbers and number text', () => {
		expect(lineNo(20000)).toBe(20000);
		expect(lineNo('20000')).toBe(20000);
		expect(lineNo('')).toBeUndefined();
		expect(lineNo('x')).toBeUndefined();
		expect(lineNo(undefined)).toBeUndefined();
	});
});

describe('newLineNo', () => {
	const rows = [{ line_no: 10000 }, { line_no: 20000 }, { line_no: 30000 }];
	it('10000 steps at the end, halfway after a selected line', () => {
		expect(newLineNo([], -1, 'line_no')).toBe(10000);
		expect(newLineNo(rows, -1, 'line_no')).toBe(40000);
		expect(newLineNo(rows, 2, 'line_no')).toBe(40000);
		expect(newLineNo(rows, 0, 'line_no')).toBe(15000);
	});
	it('counts lines still being saved, skips lines without a number', () => {
		expect(newLineNo([{ line_no: 10000, _isNew: true }], 0, 'line_no')).toBe(20000);
		expect(newLineNo([{ line_no: 10000 }, { line_no: '' }], 1, 'line_no')).toBe(20000);
		expect(newLineNo([{ line_no: 10000 }, { line_no: '' }, { line_no: 20000 }], 0, 'line_no')).toBe(15000);
	});
	it('no room between neighbours', () => {
		expect(newLineNo([{ line_no: 1 }, { line_no: 2 }], 0, 'line_no')).toBeNull();
	});
});
