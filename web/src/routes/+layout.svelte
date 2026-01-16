<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { session } from '$stores/session';
	import { currentUser } from '$lib/stores/user';
	import { toast } from '$lib/stores/toast';
	import { theme } from '$lib/stores/theme';
	import MenuBar from '$lib/components/menu/MenuBar.svelte';
	import Breadcrumb from '$lib/components/Breadcrumb.svelte';
	import ToastContainer from '$lib/components/ToastContainer.svelte';

	// Subscribe to theme to ensure it stays in sync
	let currentTheme = $state<'light' | 'dark'>('light');
	theme.subscribe((value) => {
		currentTheme = value;
	});

	// Initialize session and user on app load
	onMount(() => {
		session.initialize();
		currentUser.loadFromStorage();

		// Expose toast to console for testing (development only)
		if (typeof window !== 'undefined') {
			(window as any).toast = toast;
		}
	});
</script>

<div class="min-h-screen flex flex-col">
	<MenuBar />
	<Breadcrumb />
	<main class="flex-1 overflow-hidden">
		<slot />
	</main>
</div>

<!-- Toast notifications -->
<ToastContainer />
