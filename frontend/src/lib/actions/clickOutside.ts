/**
 * Svelte action that calls a callback when clicking outside the element.
 *
 * Usage:
 * <div use:clickOutside={() => isOpen = false}>
 *
 * Or with options:
 * <div use:clickOutside={{ callback: () => isOpen = false, enabled: isOpen }}>
 */

export interface ClickOutsideOptions {
	callback: () => void;
	enabled?: boolean;
	// elements that count as inside too (a dropdown's list moved to <body>, see floating.ts)
	also?: () => Array<HTMLElement | null | undefined>;
}

type ClickOutsideParam = (() => void) | ClickOutsideOptions;

export function clickOutside(node: HTMLElement, param: ClickOutsideParam) {
	let callback: () => void;
	let enabled: boolean;
	let also: (() => Array<HTMLElement | null | undefined>) | undefined;

	function parseParam(p: ClickOutsideParam) {
		if (typeof p === 'function') {
			callback = p;
			enabled = true;
			also = undefined;
		} else {
			callback = p.callback;
			enabled = p.enabled ?? true;
			also = p.also;
		}
	}

	parseParam(param);

	function handleClick(event: MouseEvent) {
		if (!enabled) return;

		const target = event.target as Node;
		if (also?.().some((el) => el?.contains(target))) return;
		if (node && !node.contains(target) && !event.defaultPrevented) {
			callback();
		}
	}

	// Use capture phase to catch clicks before they bubble
	document.addEventListener('click', handleClick, true);

	return {
		update(newParam: ClickOutsideParam) {
			parseParam(newParam);
		},
		destroy() {
			document.removeEventListener('click', handleClick, true);
		}
	};
}
