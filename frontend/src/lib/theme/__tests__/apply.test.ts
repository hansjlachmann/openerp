import { describe, it, expect } from 'vitest';
import { readFileSync } from 'fs';
import { resolve } from 'path';
import {
	ALL_THEME_VAR_NAMES,
	STEPS,
	cacheFor,
	hexToTriplet,
	tokensToVars,
	type Appearance,
	type Theme,
	type ThemeTokens
} from '../apply';
import { APPEARANCE, MENU } from '$lib/services/i18n.svelte';

const tokens = (over: Partial<ThemeTokens> = {}): ThemeTokens => ({
	version: 1,
	mode: 'dark',
	gray: STEPS.map(() => '#1a1b26'),
	accent: STEPS.map(() => '#7aa2f7'),
	vars: { 'page-bg': '#1a1b26', fg: '#c0caf5' },
	...over
});

describe('hexToTriplet', () => {
	it('converts #rrggbb', () => {
		expect(hexToTriplet('#1a1b26')).toBe('26 27 38');
		expect(hexToTriplet('#FFFFFF')).toBe('255 255 255');
	});
	it('rejects anything else', () => {
		for (const bad of ['', 'red', '#fff', '1a1b26', '#1a1b26; color: red', 'url(x)', '#1a1b2g']) {
			expect(hexToTriplet(bad)).toBeNull();
		}
	});
});

describe('tokensToVars', () => {
	it('maps ramps and semantic colors to CSS variables', () => {
		const v = tokensToVars(tokens());
		expect(v['--gray-50']).toBe('26 27 38');
		expect(v['--accent-950']).toBe('122 162 247');
		expect(v['--page-bg']).toBe('26 27 38');
		expect(v['--fg']).toBe('192 202 245');
	});
	it('drops invalid values and unknown names', () => {
		const v = tokensToVars(
			tokens({
				vars: { fg: 'red', 'page-bg': '#000000; display:none', 'evil-var': '#000000', surface: '#ffffff' }
			})
		);
		expect(v['--fg']).toBeUndefined();
		expect(v['--page-bg']).toBeUndefined();
		expect(v['--evil-var']).toBeUndefined();
		expect(v['--surface']).toBe('255 255 255');
	});
});

describe('theme variables and app.css', () => {
	const css = readFileSync(resolve(__dirname, '../../../app.css'), 'utf8');
	const root = css.slice(css.indexOf(':root {'), css.indexOf('}', css.indexOf(':root {')));

	it('every variable a theme sets has a default in app.css', () => {
		const missing = ALL_THEME_VAR_NAMES.filter((n) => !new RegExp(`^\\s+${n}:`, 'm').test(root));
		expect(missing).toEqual([]);
	});

	it('the cache passes the checks of the inline script in app.html', () => {
		const html = readFileSync(resolve(__dirname, '../../../app.html'), 'utf8');
		expect(html).toContain("localStorage.getItem('theme-cache')");
		const name = /^--[a-z0-9-]{1,40}$/;
		const value = /^\d{1,3} \d{1,3} \d{1,3}$/;
		const theme = (mode: 'light' | 'dark'): Theme => ({
			code: mode,
			name: mode,
			mode,
			source: 'builtin',
			tokens: tokens({ mode, vars: Object.fromEntries(ALL_THEME_VAR_NAMES.filter((n) => !/gray|accent/.test(n)).map((n) => [n.slice(2), '#123456'])) })
		});
		const a: Appearance = {
			settings: {
				light_theme: 'light', dark_theme: 'dark', mode: 'light', wallpaper: '', wallpaper_opacity: 25,
				wallpaper_blur: 0, panel_opacity: 92, font: 'default', font_size: 0, density: 'comfortable'
			},
			light: theme('light'),
			dark: theme('dark')
		};
		const cache = cacheFor(a);
		expect(cache.mode).toBe('light');
		for (const vars of [cache.light, cache.dark]) {
			expect(Object.keys(vars).length).toBe(ALL_THEME_VAR_NAMES.length);
			for (const [n, v] of Object.entries(vars)) {
				expect(name.test(n)).toBe(true);
				expect(value.test(v)).toBe(true);
			}
		}
	});
});

describe('appearance translations', () => {
	for (const lang of ['en-US', 'nb-NO', 'da-DK']) {
		it(`has every key in ${lang}`, () => {
			const file = readFileSync(resolve(__dirname, `../../../../../translations/${lang}/messages.yaml`), 'utf8');
			const keys = [...Object.values(APPEARANCE), MENU.APPEARANCE];
			expect(keys.filter((k) => !new RegExp(`^\\s+${k}:`, 'm').test(file))).toEqual([]);
		});
	}
});
