<script lang="ts">
	import { cn } from '$utils/cn';

	export let variant: 'primary' | 'secondary' | 'danger' = 'primary';
	export let size: 'sm' | 'md' | 'lg' = 'md';
	export let disabled = false;
	export let type: 'button' | 'submit' | 'reset' = 'button';
	export let onclick: ((event: MouseEvent) => void) | undefined = undefined;

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
	<slot />
</button>
