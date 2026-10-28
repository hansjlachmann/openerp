import { describe, it, expect, beforeEach } from 'vitest';
import { getFieldControls, focusAdjacentField, handleFieldEnterKey, arrowLeavesField, fieldInDirection } from '../fieldNavigation';

function setup(html: string): HTMLElement {
	document.body.innerHTML = `<div id="root">${html}</div>`;
	return document.getElementById('root')!;
}

function pressEnter(target: HTMLElement, init: KeyboardEventInit = {}): KeyboardEvent {
	const event = new KeyboardEvent('keydown', { key: 'Enter', bubbles: true, cancelable: true, ...init });
	target.dispatchEvent(event);
	return event;
}

describe('fieldNavigation', () => {
	let root: HTMLElement;

	beforeEach(() => {
		root = setup(`
			<input id="a" tabindex="1" value="one" />
			<button id="btn">x</button>
			<div id="opt" role="combobox" tabindex="3"></div>
			<input id="b" tabindex="2" />
			<input id="dis" tabindex="4" disabled />
			<span id="arrow" role="combobox" tabindex="-1"></span>
			<select id="s" tabindex="5"><option>1</option></select>
			<textarea id="ta" tabindex="6"></textarea>
		`);
		root.addEventListener('keydown', (e) => handleFieldEnterKey(e, root));
	});

	it('lists field controls in tab order, skipping disabled and tabindex -1', () => {
		expect(getFieldControls(root).map((el) => el.id)).toEqual(['a', 'b', 'opt', 's']);
	});

	it('Enter moves to the next field and selects its text', () => {
		const b = document.getElementById('b') as HTMLInputElement;
		b.value = 'abc';
		const a = document.getElementById('a')!;
		a.focus();
		const event = pressEnter(a);
		expect(event.defaultPrevented).toBe(true);
		expect(document.activeElement).toBe(b);
		expect(b.selectionStart).toBe(0);
		expect(b.selectionEnd).toBe(3);
	});

	it('Enter on a custom combobox moves on; Shift+Enter moves back', () => {
		const opt = document.getElementById('opt')!;
		opt.focus();
		pressEnter(opt);
		expect(document.activeElement?.id).toBe('s');
		pressEnter(document.activeElement as HTMLElement, { shiftKey: true });
		expect(document.activeElement?.id).toBe('opt');
	});

	it('Enter on the last field stays put', () => {
		const s = document.getElementById('s')!;
		s.focus();
		pressEnter(s);
		expect(document.activeElement).toBe(s);
	});

	it('leaves Enter alone when a control already handled it', () => {
		const a = document.getElementById('a')!;
		a.addEventListener('keydown', (e) => e.preventDefault());
		a.focus();
		pressEnter(a);
		expect(document.activeElement).toBe(a);
	});

	it('ignores textareas, buttons and modified Enter', () => {
		const ta = document.getElementById('ta')!;
		ta.focus();
		expect(pressEnter(ta).defaultPrevented).toBe(false);
		const btn = document.getElementById('btn')!;
		btn.focus();
		expect(pressEnter(btn).defaultPrevented).toBe(false);
		const a = document.getElementById('a')!;
		a.focus();
		pressEnter(a, { ctrlKey: true });
		expect(document.activeElement).toBe(a);
	});

	it('focusAdjacentField returns false for a non-field element', () => {
		expect(focusAdjacentField(root, document.getElementById('btn')!)).toBe(false);
	});
});

describe('arrow keys between card fields', () => {
	it('leaves a text field only at the start/end of the text or when all is selected', () => {
		const root = setup('<input id="t" value="hello" /><textarea id="x">a</textarea><input id="c" type="checkbox" />');
		const t = root.querySelector<HTMLInputElement>('#t')!;
		t.setSelectionRange(0, 0);
		expect(arrowLeavesField(t, 'left')).toBe(true);
		expect(arrowLeavesField(t, 'right')).toBe(false);
		t.setSelectionRange(5, 5);
		expect(arrowLeavesField(t, 'right')).toBe(true);
		t.setSelectionRange(2, 2);
		expect(arrowLeavesField(t, 'left')).toBe(false);
		expect(arrowLeavesField(t, 'down')).toBe(true);
		t.setSelectionRange(0, 5);
		expect(arrowLeavesField(t, 'right')).toBe(true);
		expect(arrowLeavesField(root.querySelector('#x')!, 'down')).toBe(false);
		expect(arrowLeavesField(root.querySelector('#c')!, 'right')).toBe(true);
	});

	it('finds the previous/next field and the nearest one in the row above/below', () => {
		const root = setup('<input id="a" /><input id="b" /><input id="c" /><input id="d" />');
		const [a, b, c, d] = ['a', 'b', 'c', 'd'].map((id) => root.querySelector<HTMLElement>('#' + id)!);
		// a two-column card: a b / c d
		const at = (el: HTMLElement, left: number, top: number) =>
			(el.getBoundingClientRect = () => ({ left, top, right: left + 100, bottom: top + 30, width: 100, height: 30, x: left, y: top, toJSON: () => ({}) }) as DOMRect);
		at(a, 0, 0);
		at(b, 200, 0);
		at(c, 0, 50);
		at(d, 200, 50);
		const controls = [a, b, c, d];
		expect(fieldInDirection(controls, a, 'right')).toBe(b);
		expect(fieldInDirection(controls, b, 'right')).toBe(c);
		expect(fieldInDirection(controls, c, 'left')).toBe(b);
		expect(fieldInDirection(controls, b, 'down')).toBe(d);
		expect(fieldInDirection(controls, c, 'up')).toBe(a);
		expect(fieldInDirection(controls, d, 'down')).toBeNull();
		expect(fieldInDirection(controls, a, 'left')).toBeNull();
	});
});
