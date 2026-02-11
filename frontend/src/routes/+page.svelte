<script lang="ts">
	import { onMount } from 'svelte';
	import { breadcrumb } from '$lib/stores/breadcrumb';
	import { fetchMenu, clearMenuCache } from '$lib/services/pages';
	import { t, HOME, MSG } from '$lib/services/i18n.svelte';
	import type { MenuDefinition } from '$lib/types/pages';

	let menu: MenuDefinition | null = $state(null);
	let loading = $state(true);
	let error = $state<string | null>(null);

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
	});

	function navigateToPage(pageId: number) {
		window.location.href = `/pages/${pageId}`;
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
			<div class="space-y-6">
				{#each menu.menu as group}
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
											onclick={() => navigateToPage(item.page_id!)}
											class="block text-nav-blue dark:text-blue-400 hover:underline cursor-pointer"
										>
											{item.name}
										</button>
									{/if}
								{/each}
							{:else if group.page_id}
								<!-- Flat menu item (no sub-items) -->
								<button
									onclick={() => navigateToPage(group.page_id!)}
									class="block text-nav-blue dark:text-blue-400 hover:underline cursor-pointer"
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
