<script lang="ts">
	import { cn } from '$utils/cn';

	export let title: string | undefined = undefined;
</script>

<div class={cn('card', $$props.class)}>
	{#if title || $$slots.header}
		<div class="card-header">
			{#if $$slots.header}
				<slot name="header" />
			{:else if title}
				<h2 class="text-xl font-semibold text-gray-900">{title}</h2>
			{/if}
		</div>
	{/if}

	<div class="card-body">
		<slot />
	</div>

	{#if $$slots.actions}
		<div class="px-6 py-4 border-t border-gray-200 bg-gray-50 flex justify-end gap-3">
			<slot name="actions" />
		</div>
	{/if}
</div>
