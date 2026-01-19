<script lang="ts">
	import PageRenderer from '$lib/components/pages/PageRenderer.svelte';

	// Get page props from SvelteKit
	let { data } = $props();
</script>

<div class="page-container">
	<PageRenderer pageid={data.pageId} recordid={data.recordId} />
</div>

<style>
	.page-container {
		@apply h-full w-full p-6;
	}
</style>
