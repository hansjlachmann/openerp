<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { breadcrumb } from '$lib/stores/breadcrumb';
	import { fetchMenu, clearMenuCache } from '$lib/services/pages';
	import { t, HOME, MSG } from '$lib/services/i18n.svelte';
	import type { MenuDefinition } from '$lib/types/pages';

	let menu: MenuDefinition | null = $state(null);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let menuElement: HTMLElement | null = $state(null);

	// Page opened last from the menu, so returning home (e.g. Escape on a list)
	// puts the keyboard focus back on that item
	const LAST_MENU_PAGE_KEY = 'openerp-last-menu-page';

	// Clear breadcrumb on home page
	onMount(async () => {
		breadcrumb.clear();
		// Clear cache to ensure fresh menu data
		clearMenuCache();
		try {
			menu = await fetchMenu();
		} catch (err) {
			console.error('Error loading menu:', err);
			error = err instanceof Error ? err.message : 'Failed to load menu';
		} finally {
			loading = false;
		}
		await tick();
		focusInitialItem();
	});

	function navigateToPage(pageId: number) {
		sessionStorage.setItem(LAST_MENU_PAGE_KEY, String(pageId));
		window.location.href = `/pages/${pageId}`;
	}

	function menuItems(): HTMLButtonElement[] {
		return Array.from(menuElement?.querySelectorAll<HTMLButtonElement>('button[data-menu-item]') ?? []);
	}

	function focusInitialItem() {
		const items = menuItems();
		const lastPage = sessionStorage.getItem(LAST_MENU_PAGE_KEY);
		const last = items.find((item) => item.dataset.pageId === lastPage);
		(last ?? items[0])?.focus();
	}

	// Keyboard navigation through the menu: ArrowUp/ArrowDown move item by item,
	// ArrowLeft/ArrowRight jump to the previous/next group, Home/End to the first/last
	// item. Enter opens the focused item (native button behavior).
	function handleMenuKeydown(event: KeyboardEvent) {
		if (event.ctrlKey || event.altKey || event.metaKey) return;
		const items = menuItems();
		if (items.length === 0) return;
		const current = items.indexOf(document.activeElement as HTMLButtonElement);
		const groupOf = (i: number) => Number(items[i]?.dataset.group);
		let target = -1;

		switch (event.key) {
			case 'ArrowDown':
				target = current < 0 ? 0 : Math.min(current + 1, items.length - 1);
				break;
			case 'ArrowUp':
				target = current < 0 ? 0 : Math.max(current - 1, 0);
				break;
			case 'Home':
				target = 0;
				break;
			case 'End':
				target = items.length - 1;
				break;
			case 'ArrowRight':
				// First item of the next group
				target = items.findIndex((_, i) => i > current && groupOf(i) !== groupOf(current));
				break;
			case 'ArrowLeft': {
				// First item of the previous group (or of the current group if not on its first item)
				const start = items.findIndex((_, i) => groupOf(i) === groupOf(current));
				const group = current > start ? groupOf(current) : groupOf(start - 1);
				target = items.findIndex((_, i) => groupOf(i) === group);
				break;
			}
			default:
				return;
		}
		event.preventDefault();
		if (target >= 0) items[target].focus();
	}
</script>

<div class="container mx-auto px-4 py-8">
	<div class="max-w-4xl mx-auto">
		<div class="text-center mb-8">
			<h1 class="text-3xl font-bold text-nav-blue dark:text-blue-400 mb-2">{t(HOME.TITLE)}</h1>
			<p class="text-gray-600 dark:text-gray-400">
				{t(HOME.DESCRIPTION)}
			</p>
		</div>

		<!-- Menu Items -->
		{#if loading}
			<div class="text-gray-500">{t(MSG.LOADING_MENU)}</div>
		{:else if error}
			<div class="text-red-500">{error}</div>
		{:else if menu && menu.menu && menu.menu.length > 0}
			<!-- svelte-ignore a11y_no_static_element_interactions -->
			<div class="space-y-6" bind:this={menuElement} onkeydown={handleMenuKeydown}>
				{#each menu.menu as group, groupIndex}
					<div>
						<h3 class="text-lg font-semibold text-gray-700 dark:text-gray-300 mb-2">
							{group.name}
						</h3>
						<div class="ml-4 space-y-1">
							{#if group.items && group.items.length > 0}
								<!-- Grouped menu -->
								{#each group.items as item}
									{#if item.page_id && item.name}
										<button
											data-menu-item
											data-group={groupIndex}
											data-page-id={item.page_id}
											onclick={() => navigateToPage(item.page_id!)}
											class="menu-item block text-nav-blue dark:text-blue-400 hover:underline cursor-pointer"
										>
											{item.name}
										</button>
									{/if}
								{/each}
							{:else if group.page_id}
								<!-- Flat menu item (no sub-items) -->
								<button
									data-menu-item
									data-group={groupIndex}
									data-page-id={group.page_id}
									onclick={() => navigateToPage(group.page_id!)}
									class="menu-item block text-nav-blue dark:text-blue-400 hover:underline cursor-pointer"
								>
									{group.name}
								</button>
							{/if}
						</div>
					</div>
				{/each}
			</div>
		{:else}
			<div class="text-gray-500">{t(MSG.NO_MENU_ITEMS)}</div>
		{/if}
	</div>
</div>

<style>
	/* Keyboard focus on a menu item: BC teal highlight */
	.menu-item {
		@apply rounded px-2 -mx-2;
	}

	.menu-item:focus {
		@apply outline-none bg-blue-50 underline;
	}

	:global(.dark) .menu-item:focus {
		@apply bg-blue-900;
	}
</style>
