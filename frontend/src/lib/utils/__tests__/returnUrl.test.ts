import { describe, it, expect, beforeEach } from 'vitest';
import { withReturn, safeReturnUrl } from '../returnUrl';

describe('returnUrl', () => {
	beforeEach(() => {
		window.history.replaceState({}, '', '/pages/22?filter=city%3DOslo&back=%2Fold&select=OLD');
	});

	it('adds the current page (without an earlier back/select) as back', () => {
		const url = withReturn('/pages/25?filter=customer_no%3DC00010', 'C00010');
		expect(url.startsWith('/pages/25?filter=customer_no%3DC00010&back=')).toBe(true);
		const back = decodeURIComponent(url.split('back=')[1]);
		expect(back).toBe('/pages/22?filter=city%3DOslo&select=C00010');
	});

	it('uses ? when the target has no query yet', () => {
		expect(withReturn('/pages/25')).toMatch(/^\/pages\/25\?back=/);
	});

	it('only accepts paths of this app', () => {
		expect(safeReturnUrl('/pages/22?select=C1')).toBe('/pages/22?select=C1');
		expect(safeReturnUrl('https://evil.example')).toBeUndefined();
		expect(safeReturnUrl('//evil.example')).toBeUndefined();
		expect(safeReturnUrl('/\\evil.example')).toBeUndefined();
		expect(safeReturnUrl('')).toBeUndefined();
		expect(safeReturnUrl(null)).toBeUndefined();
	});
});
