<script lang="ts">
	import { onMount, tick } from 'svelte';

	interface Props {
		open?: boolean;
		onclose?: () => void;
		title?: string;
		size?: 'normal' | 'expanded' | 'fullscreen';
	}

	let { open = false, onclose, title, size = 'normal' }: Props = $props();

	// Reference to modal content for focus management
	let modalContentRef: HTMLDivElement;

	// Handle escape key to close modal
	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape' && open) {
			onclose?.();
		}
	}

	// Prevent background scroll when modal is open and manage focus
	$effect(() => {
		if (open) {
			document.body.style.overflow = 'hidden';
			// Focus the modal after it renders
			tick().then(() => {
				if (modalContentRef) {
					// Find the first focusable element inside the modal
					const focusable = modalContentRef.querySelector<HTMLElement>(
						'input:not([disabled]), select:not([disabled]), textarea:not([disabled]), button:not([disabled]), [tabindex]:not([tabindex="-1"])'
					);
					if (focusable) {
						focusable.focus();
					} else {
						// If no focusable element, focus the modal itself
						modalContentRef.focus();
					}
				}
			});
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
			tabindex="-1"
			bind:this={modalContentRef}
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
