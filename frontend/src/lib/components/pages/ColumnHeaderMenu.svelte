<script lang="ts">
	// Column header menu of list pages (BC): opens under the ▾ button of a header. Items are
	// buttons: arrow keys move between them, Enter/Space chooses, Esc and a click outside
	// close the menu. Positioned fixed so the table's scroll container does not clip it.
	import { onMount, tick } from 'svelte';

	export interface ColumnMenuItem {
		label: string;
		action: () => void;
		disabled?: boolean;
		checked?: boolean;
		separatorBefore?: boolean;
	}

	interface Props {
		anchor: HTMLElement;
		items: ColumnMenuItem[];
		onclose: () => void;
	}

	let { anchor, items, onclose }: Props = $props();

	let menuElement: HTMLDivElement | undefined = $state();
	let top = $state(0);
	let left = $state(0);

	function place() {
		const r = anchor.getBoundingClientRect();
		top = r.bottom + 2;
		left = Math.min(r.left, window.innerWidth - 230);
	}

	function buttons(): HTMLButtonElement[] {
		return [...(menuElement?.querySelectorAll<HTMLButtonElement>('button:not([disabled])') ?? [])];
	}

	function choose(item: ColumnMenuItem) {
		if (item.disabled) return;
		onclose();
		item.action();
	}

	function handleKeydown(event: KeyboardEvent) {
		const list = buttons();
		const index = list.indexOf(document.activeElement as HTMLButtonElement);
		if (event.key === 'ArrowDown') {
			event.preventDefault();
			list[(index + 1) % list.length]?.focus();
		} else if (event.key === 'ArrowUp') {
			event.preventDefault();
			list[(index - 1 + list.length) % list.length]?.focus();
		} else if (event.key === 'Escape' || event.key === 'Tab') {
			event.preventDefault();
			onclose();
			anchor.focus();
		}
	}

	onMount(() => {
		place();
		tick().then(() => buttons()[0]?.focus());
		const outside = (e: MouseEvent) => {
			const target = e.target as Node;
			if (!menuElement?.contains(target) && !anchor.contains(target)) onclose();
		};
		const close = () => onclose();
		document.addEventListener('mousedown', outside, true);
		window.addEventListener('resize', close);
		window.addEventListener('scroll', close, true);
		return () => {
			document.removeEventListener('mousedown', outside, true);
			window.removeEventListener('resize', close);
			window.removeEventListener('scroll', close, true);
		};
	});
</script>

<!-- data-own-keys: the list's keyboard shortcuts leave keys inside the menu alone -->
<div
	bind:this={menuElement}
	class="column-menu"
	role="menu"
	tabindex="-1"
	data-own-keys
	style="top: {top}px; left: {left}px"
	onkeydown={handleKeydown}
>
	{#each items as item}
		{#if item.separatorBefore}
			<div class="column-menu-separator" role="separator"></div>
		{/if}
		<button
			type="button"
			role="menuitem"
			class="column-menu-item"
			disabled={item.disabled}
			onclick={() => choose(item)}
		>
			<span class="column-menu-check">{item.checked ? '✓' : ''}</span>
			{item.label}
		</button>
	{/each}
</div>

<style>
	.column-menu {
		position: fixed;
		z-index: 60;
		min-width: 210px;
		padding: 4px 0;
		background: rgb(var(--surface));
		border: 1px solid rgb(var(--gray-300));
		border-radius: 4px;
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
	}

	:global(.dark) .column-menu {
		background: rgb(var(--gray-800));
		border-color: rgb(var(--gray-700));
	}

	.column-menu-item {
		display: flex;
		align-items: center;
		width: 100%;
		padding: 6px 12px 6px 6px;
		font-size: 0.875rem;
		text-align: left;
		color: rgb(var(--fg));
		background: transparent;
		border: none;
		cursor: pointer;
	}

	:global(.dark) .column-menu-item {
		color: rgb(var(--gray-50));
	}

	.column-menu-item:hover:not(:disabled),
	.column-menu-item:focus-visible {
		background: rgb(var(--accent-100));
		outline: none;
	}

	:global(.dark) .column-menu-item:hover:not(:disabled),
	:global(.dark) .column-menu-item:focus-visible {
		background: rgb(var(--accent-800));
	}

	.column-menu-item:disabled {
		color: rgb(var(--gray-400));
		cursor: default;
	}

	.column-menu-check {
		display: inline-block;
		width: 1.25rem;
		color: rgb(var(--accent-600));
	}

	:global(.dark) .column-menu-check {
		color: rgb(var(--accent-400));
	}

	.column-menu-separator {
		height: 1px;
		margin: 4px 0;
		background: rgb(var(--gray-300));
	}

	:global(.dark) .column-menu-separator {
		background: rgb(var(--gray-700));
	}
</style>
