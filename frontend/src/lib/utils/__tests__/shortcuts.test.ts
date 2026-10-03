import { describe, it, expect } from 'vitest';
import { getShortcutKey, commonShortcuts } from '../shortcuts';

function key(init: Partial<KeyboardEvent>): KeyboardEvent {
	return { ctrlKey: false, metaKey: false, altKey: false, shiftKey: false, code: '', key: '', ...init } as KeyboardEvent;
}

describe('getShortcutKey', () => {
	it('builds modifier + key strings', () => {
		expect(getShortcutKey(key({ key: 'Insert', code: 'Insert', ctrlKey: true }))).toBe('Ctrl+Insert');
		expect(getShortcutKey(key({ key: 'F5', code: 'F5' }))).toBe('F5');
		expect(getShortcutKey(key({ key: 'e', code: 'KeyE', ctrlKey: true }))).toBe('Ctrl+E');
	});

	it('recognizes Alt+N (New)', () => {
		expect(getShortcutKey(key({ key: 'n', code: 'KeyN', altKey: true }))).toBe(commonShortcuts.NEW);
	});

	it('uses the physical key for Option+N on macOS', () => {
		expect(getShortcutKey(key({ key: '˜', code: 'KeyN', altKey: true }))).toBe('Alt+N');
		expect(getShortcutKey(key({ key: 'Dead', code: 'KeyN', altKey: true }))).toBe('Alt+N');
	});

	it('leaves Alt+Arrow keys alone', () => {
		expect(getShortcutKey(key({ key: 'ArrowDown', code: 'ArrowDown', altKey: true }))).toBe('Alt+ArrowDown');
	});
});

describe('commonShortcuts', () => {
	it('never uses browser-reserved Ctrl+N for New', () => {
		expect(commonShortcuts.NEW).toBe('Alt+N');
	});
});
