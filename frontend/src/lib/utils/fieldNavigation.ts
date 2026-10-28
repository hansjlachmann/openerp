// Enter-to-next-field on card pages (Enter moves to the next field like Tab).

// Field controls that take part in field navigation: inputs, selects and the
// custom dropdowns (OptionDropdown's combobox, LookupDropdown's inner input).
const FIELD_SELECTOR = 'input:not([type="hidden"]), select, [role="combobox"]';

// Field controls in `container`, in Tab order: positive tabindex ascending, then
// tabindex 0 in document order. Disabled and tabindex=-1 controls are skipped.
export function getFieldControls(container: HTMLElement): HTMLElement[] {
	const controls = Array.from(container.querySelectorAll<HTMLElement>(FIELD_SELECTOR)).filter(
		(el) => el.tabIndex >= 0 && !(el as HTMLInputElement).disabled
	);
	const order = (el: HTMLElement) => (el.tabIndex > 0 ? el.tabIndex : Number.MAX_SAFE_INTEGER);
	// Array.prototype.sort is stable, so equal tabindexes keep document order
	return controls.sort((a, b) => order(a) - order(b));
}

// Move focus from `from` to the next (or previous) field control in `container`, the way
// Tab does: text is selected so typing replaces it. Returns false if there is no field to
// move to (`from` is the last/first field or not a field control).
export function focusAdjacentField(container: HTMLElement, from: HTMLElement, backwards = false): boolean {
	const controls = getFieldControls(container);
	const index = controls.indexOf(from);
	if (index === -1) return false;
	const next = controls[backwards ? index - 1 : index + 1];
	if (!next) return false;
	next.focus();
	if (next instanceof HTMLInputElement && ['text', 'email', 'password', 'search', 'tel', 'url', 'number'].includes(next.type)) {
		next.select();
	}
	return true;
}

// keydown handler for a card page's field area: Enter moves to the next field,
// Shift+Enter to the previous one. Keys a control already handled (an open dropdown
// selecting a row, a lookup rejecting typed text) are left alone.
export function handleFieldEnterKey(event: KeyboardEvent, container: HTMLElement): void {
	if (event.key !== 'Enter' || event.defaultPrevented) return;
	if (event.ctrlKey || event.altKey || event.metaKey || event.isComposing) return;
	const target = event.target as HTMLElement | null;
	if (!target || target instanceof HTMLTextAreaElement || target instanceof HTMLButtonElement) return;
	if (!target.matches(FIELD_SELECTOR)) return;
	// Enter never does anything else in a field; on the last field it just stays put
	event.preventDefault();
	event.stopPropagation();
	focusAdjacentField(container, target, event.shiftKey);
}

// Focus a field as Tab does: its text selected, so typing replaces it
export function focusField(el: HTMLElement): void {
	el.focus();
	if (el instanceof HTMLInputElement && ['text', 'email', 'password', 'search', 'tel', 'url', 'number'].includes(el.type)) {
		el.select();
	}
}

export type ArrowDirection = 'up' | 'down' | 'left' | 'right';

// Whether an arrow key leaves the field (card pages, like the list's cell editing): Up/Down
// always (one-line fields), Left/Right only at the start/end of the text or when all of it is
// selected — inside the text they move the cursor. Date inputs keep Left/Right (their parts);
// selects and textareas keep all arrows.
export function arrowLeavesField(el: HTMLElement, direction: ArrowDirection): boolean {
	if (el instanceof HTMLTextAreaElement || el instanceof HTMLSelectElement) return false;
	if (direction === 'up' || direction === 'down') return true;
	if (!(el instanceof HTMLInputElement)) return true; // a custom dropdown (combobox)
	let start: number | null = null;
	let end: number | null = null;
	try {
		start = el.selectionStart;
		end = el.selectionEnd;
	} catch {
		// inputs without a text selection (checkbox …)
	}
	if (start === null || end === null) return el.type === 'checkbox' || el.type === 'radio';
	const length = el.value.length;
	if (start === 0 && end === length) return true; // all (or nothing) selected
	if (start !== end) return false;
	return direction === 'left' ? start === 0 : end === length;
}

// The field an arrow key moves to: Left/Right the previous/next field in Tab order, Up/Down the
// nearest field in the row above/below — of that row the one nearest horizontally. null: none.
export function fieldInDirection(controls: HTMLElement[], from: HTMLElement, direction: ArrowDirection): HTMLElement | null {
	const index = controls.indexOf(from);
	if (direction === 'left') return index > 0 ? controls[index - 1] : null;
	if (direction === 'right') return index >= 0 && index < controls.length - 1 ? controls[index + 1] : null;
	const r = from.getBoundingClientRect();
	const centre = r.left + r.width / 2;
	const candidates = controls
		.filter((c) => c !== from)
		.map((c) => ({ c, rect: c.getBoundingClientRect() }))
		.filter(({ rect }) => (direction === 'down' ? rect.top >= r.bottom - 1 : rect.bottom <= r.top + 1));
	if (candidates.length === 0) return null;
	const rowTop = direction === 'down' ? Math.min(...candidates.map(({ rect }) => rect.top)) : Math.max(...candidates.map(({ rect }) => rect.top));
	const row = candidates.filter(({ rect }) => Math.abs(rect.top - rowTop) <= 4);
	row.sort((a, b) => Math.abs(a.rect.left + a.rect.width / 2 - centre) - Math.abs(b.rect.left + b.rect.width / 2 - centre));
	return row[0].c;
}
