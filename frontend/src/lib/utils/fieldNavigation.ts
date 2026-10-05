// Enter-to-next-field on card pages (NAV/BC: Enter moves to the next field like Tab).

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
