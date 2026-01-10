<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { session } from '$stores/session';
	import { currentUser } from '$lib/stores/user';
	import { toast } from '$lib/stores/toast';
	import MenuBar from '$lib/components/menu/MenuBar.svelte';
	import ToastContainer from '$lib/components/ToastContainer.svelte';

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
	<main class="flex-1 overflow-hidden">
		<slot />
	</main>
</div>

<!-- Toast notifications -->
<ToastContainer />
