import { describe, it, expect } from 'vitest';
import { cardTitle, formatDecimal } from '../fieldHelpers';

describe('formatDecimal', () => {
	it('pads to the minimum places', () => {
		expect(formatDecimal('1', '2:2')).toBe('1.00');
		expect(formatDecimal('0.5', '2:5')).toBe('0.50');
		expect(formatDecimal(3, '0:5')).toBe('3');
	});

	it('keeps up to the maximum places without trailing zeros', () => {
		expect(formatDecimal('0.00001', '2:5')).toBe('0.00001');
		expect(formatDecimal('1.2300', '0:5')).toBe('1.23');
	});

	it('rounds half away from zero on the digits', () => {
		expect(formatDecimal('1.005', '2:2')).toBe('1.01');
		expect(formatDecimal('-1.005', '2:2')).toBe('-1.01');
		expect(formatDecimal('9.995', '2:2')).toBe('10.00');
		expect(formatDecimal('0.994', '2:2')).toBe('0.99');
		expect(formatDecimal('0.5', '0:0')).toBe('1');
		expect(formatDecimal('-0.001', '2:2')).toBe('0.00');
	});

	it('leaves other values as they are', () => {
		expect(formatDecimal('', '2:2')).toBe('');
		expect(formatDecimal(null, '2:2')).toBe('');
		expect(formatDecimal('abc', '2:2')).toBe('abc');
		expect(formatDecimal('1.5', undefined)).toBe('1.5');
		expect(formatDecimal('1.5', 'x')).toBe('1.5');
	});
});

describe('cardTitle', () => {
	it('adds the data caption values to the caption', () => {
		expect(cardTitle('Finance Charge Terms', ['code', 'description'], { code: 'DOM', description: '1.5 % monthly' })).toBe(
			'Finance Charge Terms – DOM · 1.5 % monthly'
		);
	});

	it('shows option captions and skips blank values', () => {
		const options = { status: { '0': 'Open', '1': 'Blocked' } };
		expect(cardTitle('Customer Card', ['no', 'name', 'status'], { no: '10000', name: '', status: 1 }, options)).toBe(
			'Customer Card – 10000 · Blocked'
		);
		expect(cardTitle('Customer Card', ['no'], {})).toBe('Customer Card');
		expect(cardTitle('Customer Card', undefined, { no: '1' })).toBe('Customer Card');
	});
});
