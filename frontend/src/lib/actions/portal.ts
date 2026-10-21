/**
 * Svelte action that moves an element to the end of the app's root (the mount element in
 * app.html), so it is drawn above the whole page: no part of the page — a sticky list
 * header, a card section, a scrolling container — can cover or clip it, and the page's own
 * key handlers do not see its keys. It stays inside the app root, where Svelte's event
 * delegation listens, so its onclick / onkeydown keep working. Use it on the single root
 * element of an {#if} block (a dropdown, a modal).
 */
export function portal(node: HTMLElement) {
	const root = document.querySelector<HTMLElement>('body > div[style*="display: contents"]') ?? document.body;
	root.appendChild(node);
	return {
		destroy() {
			node.remove();
		}
	};
}
