<script lang="ts">
	interface Props {
		width?: string;
		height?: string;
		rounded?: 'none' | 'sm' | 'md' | 'lg' | 'full';
		class?: string;
	}

	let {
		width = '100%',
		height = '1rem',
		rounded = 'md',
		class: className = ''
	}: Props = $props();

	const roundedClasses = {
		none: '',
		sm: 'rounded-sm',
		md: 'rounded',
		lg: 'rounded-lg',
		full: 'rounded-full'
	};
</script>

<div
	class="skeleton {roundedClasses[rounded]} {className}"
	style="width: {width}; height: {height};"
></div>

<style>
	.skeleton {
		background: linear-gradient(
			90deg,
			#e5e7e9 0%,
			#f2f2f3 50%,
			#e5e7e9 100%
		);
		background-size: 200% 100%;
		animation: shimmer 1.5s infinite;
	}

	:global(.dark) .skeleton {
		background: linear-gradient(
			90deg,
			#303032 0%,
			#505c6d 50%,
			#303032 100%
		);
		background-size: 200% 100%;
	}

	@keyframes shimmer {
		0% {
			background-position: 200% 0;
		}
		100% {
			background-position: -200% 0;
		}
	}
</style>
