<script lang="ts">
	import Button from './Button.svelte';
	import type { Snippet } from 'svelte';

	interface Action {
		label: string;
		onClick: () => void;
		variant?: 'primary' | 'secondary' | 'danger';
		shortcut?: string;
	}

	interface Props {
		title: string;
		subtitle?: string;
		onclose?: () => void;
		actions?: Action[];
		leftActions?: Snippet;
		rightActions?: Snippet;
	}

	let {
		title,
		subtitle,
		onclose,
		actions = [],
		leftActions,
		rightActions
	}: Props = $props();
</script>

<div class="page-header">
	<div class="flex items-center justify-between">
		<div class="flex items-center gap-4">
			<div>
				<h1 class="header-title">{title}</h1>
				{#if subtitle}
					<p class="header-subtitle">{subtitle}</p>
				{/if}
			</div>

			{#if leftActions}
				<div class="flex gap-3">
					{@render leftActions()}
				</div>
			{/if}
		</div>

		<div class="flex items-center gap-3">
			{#if rightActions}
				<div class="flex gap-3">
					{@render rightActions()}
				</div>
			{:else if actions.length > 0}
				<div class="flex gap-3">
					{#each actions as action}
						<Button variant={action.variant || 'secondary'} onclick={action.onClick}>
							{action.label}
							{#if action.shortcut}
								<span class="ml-2 text-xs opacity-70">({action.shortcut})</span>
							{/if}
						</Button>
					{/each}
				</div>
			{/if}

			{#if onclose}
				<button
					class="close-button"
					onclick={onclose}
					title="Close (Esc)"
					aria-label="Close"
				>
					<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</button>
			{/if}
		</div>
	</div>
</div>

<style>
	.page-header {
		@apply bg-white border-b border-gray-200 px-6 py-4;
	}

	:global(.dark) .page-header {
		background-color: #1f2937; /* gray-800 */
		border-color: #374151; /* gray-700 */
	}

	.header-title {
		@apply text-2xl font-bold text-nav-blue;
	}

	:global(.dark) .header-title {
		color: #60a5fa; /* blue-400 */
	}

	.header-subtitle {
		@apply mt-1 text-sm text-gray-600;
	}

	:global(.dark) .header-subtitle {
		color: #9ca3af; /* gray-400 */
	}

	.close-button {
		@apply p-2 rounded-md text-gray-500 hover:text-gray-700 hover:bg-gray-100 transition-colors;
	}

	:global(.dark) .close-button {
		color: #9ca3af;
	}

	:global(.dark) .close-button:hover {
		color: #f3f4f6;
		background-color: #374151;
	}
</style>
