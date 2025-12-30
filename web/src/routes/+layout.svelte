<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { session } from '$stores/session';
	import MenuBar from '$lib/components/menu/MenuBar.svelte';

	// Initialize session on app load
	onMount(() => {
		session.initialize();
	});
</script>

<div class="min-h-screen flex flex-col">
	<MenuBar />
	<main class="flex-1 overflow-hidden">
		<slot />
	</main>
</div>
