<script lang="ts">
	import { breadcrumb, type BreadcrumbItem } from '$lib/stores/breadcrumb';

	let items: BreadcrumbItem[] = $state([]);

	breadcrumb.subscribe((state) => {
		items = state.items;
	});
</script>

{#if items.length > 1}
	<nav class="breadcrumb" aria-label="Breadcrumb">
		<ol class="breadcrumb-list">
			{#each items as item, index}
				<li class="breadcrumb-item">
					{#if index < items.length - 1}
						<a href={item.href} class="breadcrumb-link">
							{#if index === 0}
								<svg
									xmlns="http://www.w3.org/2000/svg"
									class="h-4 w-4 mr-1"
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
							{/if}
							{item.label}
						</a>
						<svg
							class="breadcrumb-separator"
							xmlns="http://www.w3.org/2000/svg"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M9 5l7 7-7 7"
							/>
						</svg>
					{:else}
						<span class="breadcrumb-current">{item.label}</span>
					{/if}
				</li>
			{/each}
		</ol>
	</nav>
{/if}

<style>
	.breadcrumb {
		@apply bg-gray-100 dark:bg-gray-800 px-6 py-2 border-b border-gray-200 dark:border-gray-700;
	}

	.breadcrumb-list {
		@apply flex items-center gap-1 text-sm;
	}

	.breadcrumb-item {
		@apply flex items-center;
	}

	.breadcrumb-link {
		@apply flex items-center text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300 hover:underline transition-colors;
	}

	.breadcrumb-separator {
		@apply h-4 w-4 mx-2 text-gray-400 dark:text-gray-500;
	}

	.breadcrumb-current {
		@apply text-gray-700 dark:text-gray-300 font-medium;
	}
</style>
