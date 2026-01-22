<script lang="ts">
	import { onMount } from 'svelte';
	import { breadcrumb } from '$lib/stores/breadcrumb';
	import { fetchMenu, clearMenuCache } from '$lib/services/pages';
	import { tLang } from '$lib/services/i18n';
	import { currentUser } from '$lib/stores/user';
	import type { MenuDefinition } from '$lib/types/pages';

	let menu: MenuDefinition | null = $state(null);
	let loading = $state(true);
	let error = $state<string | null>(null);

	// Track user's translation key with local state updated via effect
	let translationKey = $state('en-US');

	// Update translation key when user store changes
	$effect(() => {
		const unsubscribe = currentUser.subscribe((user) => {
			// Use translation_key if available, otherwise fall back to language
			translationKey = user?.translation_key || user?.language || 'en-US';
		});
		return unsubscribe;
	});

	// Reactive translations that update when translationKey changes
	const translations = $derived({
		homeTitle: tLang('HOME_TITLE', translationKey),
		homeDescription: tLang('HOME_DESCRIPTION', translationKey),
		homeGeneral: tLang('HOME_GENERAL', translationKey),
		homeSettings: tLang('HOME_SETTINGS', translationKey),
		menuCustomers: tLang('MENU_CUSTOMERS', translationKey),
		menuUsers: tLang('MENU_USERS', translationKey),
		menuLanguages: tLang('MENU_LANGUAGES', translationKey),
		menuPaymentTerms: tLang('MENU_PAYMENT_TERMS', translationKey),
		menuMenus: tLang('MENU_MENUS', translationKey)
	});

	// Get translated menu item name
	function getMenuItemName(name: string): string {
		const menuNameToKey: Record<string, string> = {
			'Customers': 'menuCustomers',
			'Users': 'menuUsers',
			'Languages': 'menuLanguages',
			'Payment Terms': 'menuPaymentTerms',
			'Menus': 'menuMenus'
		};
		const key = menuNameToKey[name] as keyof typeof translations;
		return key ? translations[key] : name;
	}

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
			<h1 class="text-3xl font-bold text-nav-blue dark:text-blue-400 mb-2">{translations.homeTitle}</h1>
			<p class="text-gray-600 dark:text-gray-400">
				{translations.homeDescription}
			</p>
		</div>

		<!-- Menu Items -->
		{#if loading}
			<div class="text-gray-500">Loading menu...</div>
		{:else if error}
			<div class="text-red-500">{error}</div>
		{:else if menu && menu.menu && menu.menu.length > 0}
			<div class="space-y-6">
				<!-- General -->
				<div>
					<h3 class="text-lg font-semibold text-gray-700 dark:text-gray-300 mb-2">{translations.homeGeneral}</h3>
					<div class="ml-4 space-y-1">
						{#each menu.menu as item}
							{#if item.page_id && item.name === 'Customers'}
								<button
									onclick={() => navigateToPage(item.page_id!)}
									class="block text-nav-blue dark:text-blue-400 hover:underline cursor-pointer"
								>
									{getMenuItemName(item.name)}
								</button>
							{/if}
						{/each}
					</div>
				</div>

				<!-- Settings -->
				<div>
					<h3 class="text-lg font-semibold text-gray-700 dark:text-gray-300 mb-2">{translations.homeSettings}</h3>
					<div class="ml-4 space-y-1">
						{#each menu.menu as item}
							{#if item.page_id && item.name !== 'Customers'}
								<button
									onclick={() => navigateToPage(item.page_id!)}
									class="block text-nav-blue dark:text-blue-400 hover:underline cursor-pointer"
								>
									{getMenuItemName(item.name)}
								</button>
							{/if}
						{/each}
					</div>
				</div>
			</div>
		{:else}
			<div class="text-gray-500">No menu items available</div>
		{/if}
	</div>
</div>
