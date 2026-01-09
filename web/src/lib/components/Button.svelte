<script lang="ts">
	import { cn } from '$utils/cn';

	export let variant: 'primary' | 'secondary' | 'danger' | 'success' = 'primary';
	export let size: 'sm' | 'md' | 'lg' = 'md';
	export let disabled = false;
	export let type: 'button' | 'submit' | 'reset' = 'button';
	export let onclick: ((event: MouseEvent) => void) | undefined = undefined;

	const variants = {
		primary: 'btn-primary',
		secondary: 'btn-secondary',
		danger: 'btn-danger',
		success: 'btn-success'
	};

	const sizes = {
		sm: 'px-3 py-1.5 text-sm',
		md: 'px-4 py-2',
		lg: 'px-6 py-3 text-lg'
	};

	function handleClick(event: MouseEvent) {
		if (!disabled && onclick) {
			onclick(event);
		}
	}
</script>

<button
	{type}
	{disabled}
	onclick={handleClick}
	class={cn('btn', variants[variant], sizes[size], $$props.class)}
>
	{#if $$slots.icon}
		<span class="inline-flex items-center mr-1.5">
			<slot name="icon" />
		</span>
	{/if}
	<slot />
</button>
