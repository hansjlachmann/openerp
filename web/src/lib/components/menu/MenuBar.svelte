<script lang="ts">
	import { onMount } from 'svelte';
	import type { MenuDefinition } from '$lib/types/pages';
	import { fetchMenu } from '$lib/services/pages';
	import MenuGroup from './MenuGroup.svelte';
	import { theme } from '$lib/stores/theme';

	let menu: MenuDefinition | null = $state(null);
	let loading = $state(true);
	let currentTheme = $state<'light' | 'dark'>('light');

	theme.subscribe((value) => {
		currentTheme = value;
	});

	onMount(async () => {
		try {
			menu = await fetchMenu();
		} catch (err) {
			console.error('Error loading menu:', err);
		} finally {
			loading = false;
		}
	});

	function toggleTheme() {
		theme.toggle();
	}
</script>

{#if loading}
	<div class="menu-bar bg-nav-blue text-white">
		<div class="px-4 py-2 text-sm">Loading menu...</div>
	</div>
{:else if menu}
	<nav class="menu-bar bg-nav-blue text-white">
		<div class="flex items-center gap-2 w-full">
			<!-- Home button -->
			<a
				href="/"
				class="flex items-center gap-2 px-4 py-2 hover:bg-white/10 rounded transition-colors font-semibold border-r border-white/20 mr-2"
				title="Home"
			>
				<svg
					xmlns="http://www.w3.org/2000/svg"
					class="h-5 w-5"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6"
					/>
				</svg>
				<span>Home</span>
			</a>

			<!-- Menu groups -->
			<div class="flex items-center gap-2 flex-1">
				{#each menu.menu as group}
					<MenuGroup {group} />
				{/each}
			</div>

			<!-- Theme toggle button -->
			<button
				onclick={toggleTheme}
				class="flex items-center gap-2 px-3 py-2 hover:bg-white/10 rounded transition-colors"
				title={currentTheme === 'light' ? 'Switch to dark mode' : 'Switch to light mode'}
				aria-label="Toggle theme"
			>
				{#if currentTheme === 'light'}
					<!-- Moon icon for dark mode -->
					<svg
						xmlns="http://www.w3.org/2000/svg"
						class="h-5 w-5"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z"
						/>
					</svg>
				{:else}
					<!-- Sun icon for light mode -->
					<svg
						xmlns="http://www.w3.org/2000/svg"
						class="h-5 w-5"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z"
						/>
					</svg>
				{/if}
			</button>
		</div>
	</nav>
{/if}

<style>
	.menu-bar {
		@apply flex items-center px-4 py-2 shadow-md;
		min-height: 3rem;
	}
</style>
