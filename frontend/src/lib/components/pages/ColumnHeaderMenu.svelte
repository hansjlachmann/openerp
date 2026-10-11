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
		background: #ffffff;
		border: 1px solid #d1d5db;
		border-radius: 4px;
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
	}

	:global(.dark) .column-menu {
		background: #1f1f1f;
		border-color: #303032;
	}

	.column-menu-item {
		display: flex;
		align-items: center;
		width: 100%;
		padding: 6px 12px 6px 6px;
		font-size: 0.875rem;
		text-align: left;
		color: #212121;
		background: transparent;
		border: none;
		cursor: pointer;
	}

	:global(.dark) .column-menu-item {
		color: #f7f7f7;
	}

	.column-menu-item:hover:not(:disabled),
	.column-menu-item:focus-visible {
		background: #cce8ea;
		outline: none;
	}

	:global(.dark) .column-menu-item:hover:not(:disabled),
	:global(.dark) .column-menu-item:focus-visible {
		background: #00585c;
	}

	.column-menu-item:disabled {
		color: #a4b0c4;
		cursor: default;
	}

	.column-menu-check {
		display: inline-block;
		width: 1.25rem;
		color: #008489;
	}

	:global(.dark) .column-menu-check {
		color: #37a1a5;
	}

	.column-menu-separator {
		height: 1px;
		margin: 4px 0;
		background: #d1d5db;
	}

	:global(.dark) .column-menu-separator {
		background: #303032;
	}
</style>
