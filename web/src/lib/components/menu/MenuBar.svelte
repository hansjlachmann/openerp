<script lang="ts">
	import { onMount } from 'svelte';
	import type { MenuDefinition } from '$lib/types/pages';
	import { fetchMenu } from '$lib/services/pages';
	import MenuGroup from './MenuGroup.svelte';

	let menu: MenuDefinition | null = $state(null);
	let loading = $state(true);

	onMount(async () => {
		try {
			menu = await fetchMenu();
		} catch (err) {
			console.error('Error loading menu:', err);
		} finally {
			loading = false;
		}
	});
</script>

{#if loading}
	<div class="menu-bar bg-nav-blue text-white">
		<div class="px-4 py-2 text-sm">Loading menu...</div>
	</div>
{:else if menu}
	<nav class="menu-bar bg-nav-blue text-white">
		<div class="flex items-center gap-2">
			{#each menu.menu as group}
				<MenuGroup {group} />
			{/each}
		</div>
	</nav>
{/if}

<style>
	.menu-bar {
		@apply flex items-center px-4 py-2 shadow-md;
		min-height: 3rem;
	}
</style>
