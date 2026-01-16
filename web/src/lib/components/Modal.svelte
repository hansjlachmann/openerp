<script lang="ts">
	import { onMount } from 'svelte';

	interface Props {
		open?: boolean;
		onclose?: () => void;
		title?: string;
		size?: 'normal' | 'expanded' | 'fullscreen';
	}

	let { open = false, onclose, title, size = 'normal' }: Props = $props();

	// Handle escape key to close modal
	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape' && open) {
			onclose?.();
		}
	}

	// Prevent background scroll when modal is open
	$effect(() => {
		if (open) {
			document.body.style.overflow = 'hidden';
		} else {
			document.body.style.overflow = '';
		}

		return () => {
			document.body.style.overflow = '';
		};
	});

	// Size classes
	const sizeClasses = {
		normal: 'max-w-2xl',
		expanded: 'max-w-6xl',
		fullscreen: 'max-w-full h-full m-0'
	};
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
	<div class="modal-backdrop" onclick={onclose}>
		<div
			class="modal-content {sizeClasses[size]}"
			onclick={(e) => e.stopPropagation()}
			role="dialog"
			aria-modal="true"
			aria-labelledby={title ? 'modal-title' : undefined}
		>
			<slot />
		</div>
	</div>
{/if}

<style>
	.modal-backdrop {
		@apply fixed inset-0 z-50 flex items-center justify-center;
		@apply bg-black/50 backdrop-blur-sm;
		@apply p-4;
	}

	.modal-content {
		@apply bg-white rounded-lg shadow-2xl;
		@apply w-full max-h-[90vh] overflow-hidden;
		@apply flex flex-col;
		animation: modalFadeIn 0.2s ease-out;
	}

	:global(.dark) .modal-content {
		background-color: #1f2937; /* gray-800 */
	}

	@keyframes modalFadeIn {
		from {
			opacity: 0;
			transform: scale(0.95);
		}
		to {
			opacity: 1;
			transform: scale(1);
		}
	}
</style>
