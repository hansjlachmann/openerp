<script lang="ts">
	import Modal from './Modal.svelte';
	import Button from './Button.svelte';

	interface Props {
		open?: boolean;
		title?: string;
		message?: string;
		progress?: number;
		error?: string;
		confirmMode?: boolean;
		confirmMessage?: string;
		onConfirmResponse?: (response: boolean) => void;
		showCancel?: boolean;
		onCancel?: () => void;
	}

	let {
		open = false,
		title = 'Processing...',
		message = '',
		progress = 0,
		error = '',
		confirmMode = false,
		confirmMessage = '',
		onConfirmResponse,
		showCancel = false,
		onCancel
	}: Props = $props();

	const displayPercent = $derived(Math.round(progress));

	function handleYes() {
		onConfirmResponse?.(true);
	}

	function handleNo() {
		onConfirmResponse?.(false);
	}

	function handleCancel() {
		onCancel?.();
	}
</script>

<Modal {open}>
	<div class="progress-modal">
		<div class="progress-header">
			<h3 class="progress-title">{confirmMode ? 'Confirm' : title}</h3>
		</div>

		<div class="progress-body">
			{#if confirmMode}
				<div class="confirm-container">
					<svg class="confirm-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<circle cx="12" cy="12" r="10" />
						<path d="M12 16v-4" />
						<path d="M12 8h.01" />
					</svg>
					<p class="confirm-message">{confirmMessage}</p>
				</div>
				<div class="confirm-buttons">
					<Button variant="secondary" onclick={handleNo}>No</Button>
					<Button variant="primary" onclick={handleYes}>Yes</Button>
				</div>
			{:else if error}
				<div class="error-container">
					<svg class="error-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<circle cx="12" cy="12" r="10" />
						<line x1="12" y1="8" x2="12" y2="12" />
						<line x1="12" y1="16" x2="12.01" y2="16" />
					</svg>
					<p class="error-message">{error}</p>
				</div>
			{:else}
				{#if message}
					<p class="status-message">{message}</p>
				{/if}

				<div class="progress-container">
					<div class="progress-bar-track">
						<div class="progress-bar-fill" style="width: {progress}%"></div>
					</div>
					<span class="progress-percent">{displayPercent}%</span>
				</div>

				{#if showCancel}
					<div class="cancel-container">
						<Button variant="secondary" onclick={handleCancel}>Cancel</Button>
					</div>
				{/if}
			{/if}
		</div>
	</div>
</Modal>

<style>
	.progress-modal {
		@apply p-6 min-w-[320px];
	}

	.progress-header {
		@apply mb-4;
	}

	.progress-title {
		@apply text-lg font-semibold text-gray-900;
	}

	:global(.dark) .progress-title {
		color: #f3f4f6;
	}

	.progress-body {
		@apply space-y-4;
	}

	.status-message {
		@apply text-sm text-gray-600;
	}

	:global(.dark) .status-message {
		color: #9ca3af;
	}

	.progress-container {
		@apply flex items-center gap-3;
	}

	.progress-bar-track {
		@apply flex-1 h-3 bg-gray-200 rounded-full overflow-hidden;
	}

	:global(.dark) .progress-bar-track {
		background-color: #374151;
	}

	.progress-bar-fill {
		@apply h-full bg-blue-500 rounded-full transition-all duration-150 ease-out;
	}

	.progress-percent {
		@apply text-sm font-medium text-gray-700 tabular-nums w-12 text-right;
	}

	:global(.dark) .progress-percent {
		color: #d1d5db;
	}

	.error-container {
		@apply flex items-start gap-3 p-3 bg-red-50 rounded-lg;
	}

	:global(.dark) .error-container {
		background-color: rgba(239, 68, 68, 0.1);
	}

	.error-icon {
		@apply w-5 h-5 text-red-500 flex-shrink-0 mt-0.5;
	}

	.error-message {
		@apply text-sm text-red-700;
	}

	:global(.dark) .error-message {
		color: #fca5a5;
	}

	.confirm-container {
		@apply flex items-start gap-3 p-3 bg-blue-50 rounded-lg;
	}

	:global(.dark) .confirm-container {
		background-color: rgba(59, 130, 246, 0.1);
	}

	.confirm-icon {
		@apply w-5 h-5 text-blue-500 flex-shrink-0 mt-0.5;
	}

	.confirm-message {
		@apply text-sm text-gray-700;
	}

	:global(.dark) .confirm-message {
		color: #d1d5db;
	}

	.confirm-buttons {
		@apply flex justify-end gap-3 mt-4;
	}

	.cancel-container {
		@apply flex justify-end mt-4;
	}
</style>
