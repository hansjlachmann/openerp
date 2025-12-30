<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { session } from '$stores/session';
	import MenuBar from '$lib/components/menu/MenuBar.svelte';

	// Initialize session on app load
	onMount(() => {
		session.initialize();

		// Check if user is logged in (except on login page)
		const currentPath = window.location.pathname;
		const isLoginPage = currentPath === '/login';

		if (!isLoginPage) {
			const currentUser = localStorage.getItem('currentUser');
			if (!currentUser) {
				// Not logged in, redirect to login
				goto('/login');
			}
		}
	});
</script>

<div class="min-h-screen flex flex-col">
	<MenuBar />
	<main class="flex-1 overflow-hidden">
		<slot />
	</main>
</div>
