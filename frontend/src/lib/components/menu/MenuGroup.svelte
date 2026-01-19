<script lang="ts">
	import type { MenuGroup as MenuGroupType } from '$lib/types/pages';

	interface Props {
		group: MenuGroupType;
	}

	let { group }: Props = $props();

	let isOpen = $state(false);

	function toggleMenu() {
		isOpen = !isOpen;
	}

	function closeMenu() {
		isOpen = false;
	}

	function handleMenuItemClick(pageId: number) {
		window.location.href = `/pages/${pageId}`;
		closeMenu();
	}

	// Close menu when clicking outside
	function handleClickOutside(event: MouseEvent) {
		const target = event.target as HTMLElement;
		if (!target.closest('.menu-group')) {
			closeMenu();
		}
	}

	$effect(() => {
		if (isOpen) {
			document.addEventListener('click', handleClickOutside);
			return () => {
				document.removeEventListener('click', handleClickOutside);
			};
		}
	});
</script>

<div class="menu-group relative">
	<button
		class="menu-button px-4 py-2 hover:bg-white/10 rounded transition-colors flex items-center gap-2"
		onclick={toggleMenu}
	>
		{#if group.icon}
			<span class="icon">{group.icon}</span>
		{/if}
		<span class="font-medium">{group.name}</span>
		<span class="text-xs">▼</span>
	</button>

	{#if isOpen}
		<div class="menu-dropdown absolute top-full left-0 mt-1 bg-white text-gray-800 rounded-lg shadow-xl border border-gray-200 min-w-64 z-50">
			{#each group.items as item}
				{#if item.separator}
					<div class="menu-separator border-t border-gray-200 my-1"></div>
				{:else if item.enabled !== false}
					<button
						class="menu-item w-full px-4 py-3 hover:bg-blue-50 transition-colors flex items-start gap-3 text-left"
						onclick={() => item.page_id && handleMenuItemClick(item.page_id)}
						disabled={!item.page_id}
					>
						{#if item.icon}
							<span class="icon text-nav-blue mt-0.5">{item.icon}</span>
						{/if}
						<div class="flex-1">
							<div class="font-medium text-gray-900">{item.name}</div>
							{#if item.description}
								<div class="text-sm text-gray-500 mt-0.5">{item.description}</div>
							{/if}
						</div>
					</button>
				{:else}
					<div class="menu-item-disabled w-full px-4 py-3 flex items-start gap-3 opacity-50 cursor-not-allowed">
						{#if item.icon}
							<span class="icon text-gray-400 mt-0.5">{item.icon}</span>
						{/if}
						<div class="flex-1">
							<div class="font-medium text-gray-500">{item.name}</div>
							{#if item.description}
								<div class="text-sm text-gray-400 mt-0.5">{item.description}</div>
							{/if}
						</div>
					</div>
				{/if}
			{/each}
		</div>
	{/if}
</div>

<style>
	.menu-group {
		@apply relative;
	}

	.menu-button {
		@apply text-sm font-medium cursor-pointer select-none;
	}

	.menu-dropdown {
		animation: slideDown 0.15s ease-out;
	}

	@keyframes slideDown {
		from {
			opacity: 0;
			transform: translateY(-10px);
		}
		to {
			opacity: 1;
			transform: translateY(0);
		}
	}

	.menu-item {
		@apply border-none outline-none;
	}

	.menu-item:first-child {
		@apply rounded-t-lg;
	}

	.menu-item:last-child {
		@apply rounded-b-lg;
	}
</style>
