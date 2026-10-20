import { describe, it, expect } from 'vitest';
import { splitLineNo, lineNo } from '../autoSplitKey';

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
