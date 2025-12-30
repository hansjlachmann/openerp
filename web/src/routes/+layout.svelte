<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { session } from '$stores/session';
	import { currentUser } from '$lib/stores/user';
	import MenuBar from '$lib/components/menu/MenuBar.svelte';

	// Initialize session and user on app load
	onMount(() => {
		session.initialize();
		currentUser.loadFromStorage();
	});
</script>

<div class="min-h-screen flex flex-col">
	<MenuBar />
	<main class="flex-1 overflow-hidden">
		<slot />
	</main>
</div>
