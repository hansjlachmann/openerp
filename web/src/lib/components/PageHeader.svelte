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

<div class="page-header bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 px-6 py-4">
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold text-nav-blue dark:text-blue-400">{title}</h1>
			{#if subtitle}
				<p class="mt-1 text-sm text-gray-600 dark:text-gray-400">{subtitle}</p>
			{/if}
		</div>

		{#if $$slots.actions}
			<div class="flex gap-3">
				<slot name="actions" />
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
