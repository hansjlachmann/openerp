<script lang="ts">
	import Button from './Button.svelte';

	export let title: string;
	export let subtitle: string | undefined = undefined;

	export interface Action {
		label: string;
		onClick: () => void;
		variant?: 'primary' | 'secondary' | 'danger';
		shortcut?: string;
	}

	export let actions: Action[] = [];
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

			{#if $$slots.leftActions}
				<div class="flex gap-3">
					<slot name="leftActions" />
				</div>
			{/if}
		</div>

		{#if $$slots.actions || $$slots.rightActions}
			<div class="flex gap-3">
				{#if $$slots.rightActions}
					<slot name="rightActions" />
				{:else}
					<slot name="actions" />
				{/if}
			</div>
		{:else if actions.length > 0}
			<div class="flex gap-3">
				{#each actions as action}
					<Button variant={action.variant || 'secondary'} on:click={action.onClick}>
						{action.label}
						{#if action.shortcut}
							<span class="ml-2 text-xs opacity-70">({action.shortcut})</span>
						{/if}
					</Button>
				{/each}
			</div>
		{/if}
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
</style>
