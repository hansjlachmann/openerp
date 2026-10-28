/**
 * Svelte action for a dropdown's open list: it is moved to <body> and placed (fixed) under its
 * field — above it when there is no room below — so no scrolling container (a card's lines, a
 * list) and no transformed parent (a modal) can clip it. It follows the field while the page
 * scrolls or resizes and is removed with the list.
 *
 * Usage: <div class="panel" use:floating={inputElement}>
 */
export function floating(node: HTMLElement, anchor: HTMLElement | null | undefined) {
	let current = anchor;
	document.body.appendChild(node);
	node.style.position = 'fixed';
	node.style.zIndex = '2000';
	node.style.margin = '0';

	function place() {
		if (!current) return;
		const r = current.getBoundingClientRect();
		const height = node.offsetHeight;
		const width = node.offsetWidth;
		const below = window.innerHeight - r.bottom;
		node.style.minWidth = `${r.width}px`;
		node.style.left = `${Math.max(4, Math.min(r.left, window.innerWidth - width - 4))}px`;
		if (below < height + 4 && r.top > below) {
			node.style.top = `${Math.max(4, r.top - height - 2)}px`;
		} else {
			node.style.top = `${r.bottom + 2}px`;
		}
	}

	place();
	requestAnimationFrame(place); // again once its rows are rendered (its height)
	window.addEventListener('scroll', place, true);
	window.addEventListener('resize', place);

	return {
		update(newAnchor: HTMLElement | null | undefined) {
			current = newAnchor;
			place();
		},
		destroy() {
			window.removeEventListener('scroll', place, true);
			window.removeEventListener('resize', place);
			node.remove();
		}
	};
}
