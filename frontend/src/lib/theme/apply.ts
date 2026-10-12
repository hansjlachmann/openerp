/**
 * Color themes: the backend sends a theme's colors as tokens (backend/foundation/themes);
 * they become CSS variables on <html> (RGB triplets, see src/app.css), which the Tailwind
 * colors and component styles use.
 */

export type Mode = 'light' | 'dark';

export const STEPS = [50, 100, 200, 300, 400, 500, 600, 700, 800, 900, 950] as const;

export interface ThemeTokens {
	version: number;
	mode: Mode;
	gray: string[];
	accent: string[];
	vars: Record<string, string>;
}

export interface Theme {
	code: string;
	name: string;
	mode: Mode;
	source: 'builtin' | 'github';
	author?: string;
	repo_url?: string;
	tokens: ThemeTokens;
}

export interface AppearanceSettings {
	light_theme: string;
	dark_theme: string;
	mode: Mode;
	wallpaper: string;
	wallpaper_opacity: number;
	wallpaper_blur: number;
	panel_opacity: number;
	font: string;
	font_size: number;
	density: 'comfortable' | 'compact';
}

export interface Appearance {
	settings: AppearanceSettings;
	light: Theme;
	dark: Theme;
}

/** The semantic variables a theme may set (names without "--"), as in app.css */
export const THEME_VARS = [
	'page-bg',
	'surface',
	'fg',
	'heading',
	'appbar',
	'appbar-fg',
	'row-selected',
	'row-selected-hover',
	'text-sel-bg',
	'text-sel-fg',
	'scroll-track',
	'scroll-thumb',
	'scroll-thumb-hover'
] as const;

const HEX = /^#([0-9a-f]{6})$/i;

/** "#1a1b26" → "26 27 38"; anything that is not #rrggbb → null (never passed on) */
export function hexToTriplet(hex: string): string | null {
	const m = HEX.exec(hex);
	if (!m) return null;
	const n = parseInt(m[1], 16);
	return `${(n >> 16) & 255} ${(n >> 8) & 255} ${n & 255}`;
}

/** The CSS variables of a theme: { '--gray-50': '247 247 247', …, '--fg': '33 33 33' } */
export function tokensToVars(tokens: ThemeTokens): Record<string, string> {
	const out: Record<string, string> = {};
	STEPS.forEach((step, i) => {
		const g = hexToTriplet(tokens.gray?.[i] ?? '');
		const a = hexToTriplet(tokens.accent?.[i] ?? '');
		if (g) out[`--gray-${step}`] = g;
		if (a) out[`--accent-${step}`] = a;
	});
	for (const name of THEME_VARS) {
		const v = hexToTriplet(tokens.vars?.[name] ?? '');
		if (v) out[`--${name}`] = v;
	}
	return out;
}

/** Every variable a theme can set — cleared before another theme is applied */
export const ALL_THEME_VAR_NAMES = [
	...STEPS.map((s) => `--gray-${s}`),
	...STEPS.map((s) => `--accent-${s}`),
	...THEME_VARS.map((n) => `--${n}`)
];

/** Sets the theme's variables on el (default <html>); variables it lacks fall back to app.css */
export function applyVars(vars: Record<string, string>, el: HTMLElement = document.documentElement) {
	for (const name of ALL_THEME_VAR_NAMES) {
		if (vars[name]) el.style.setProperty(name, vars[name]);
		else el.style.removeProperty(name);
	}
}

/** Switches <html> to mode and applies the variables */
export function applyMode(mode: Mode, vars: Record<string, string>, el: HTMLElement = document.documentElement) {
	el.classList.toggle('dark', mode === 'dark');
	applyVars(vars, el);
}

/**
 * localStorage cache read by the inline script in app.html, so a page load paints in the
 * user's theme before the app has started: { mode, light: vars, dark: vars }.
 */
export const THEME_CACHE_KEY = 'theme-cache';

export interface ThemeCache {
	mode: Mode;
	light: Record<string, string>;
	dark: Record<string, string>;
}

export function cacheFor(appearance: Appearance): ThemeCache {
	return {
		mode: appearance.settings.mode,
		light: tokensToVars(appearance.light.tokens),
		dark: tokensToVars(appearance.dark.tokens)
	};
}
