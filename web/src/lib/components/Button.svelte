<script lang="ts">
	import { cn } from '$utils/cn';

	export let variant: 'primary' | 'secondary' | 'danger' = 'primary';
	export let size: 'sm' | 'md' | 'lg' = 'md';
	export let disabled = false;
	export let type: 'button' | 'submit' | 'reset' = 'button';

	const variants = {
		primary: 'btn-primary',
		secondary: 'btn-secondary',
		danger: 'btn-danger'
	};

	const sizes = {
		sm: 'px-3 py-1.5 text-sm',
		md: 'px-4 py-2',
		lg: 'px-6 py-3 text-lg'
	};
</script>

<button
	{type}
	{disabled}
	class={cn('btn', variants[variant], sizes[size], $$props.class)}
	on:click
>
	<slot />
</button>
