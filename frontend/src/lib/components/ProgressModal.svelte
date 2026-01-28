<script lang="ts">
	import Modal from './Modal.svelte';

	interface Props {
		open?: boolean;
		title?: string;
		message?: string;
		progress?: number;
		error?: string;
	}

	let { open = false, title = 'Processing...', message = '', progress = 0, error = '' }: Props = $props();

	const displayPercent = $derived(Math.round(progress));
</script>

<Modal {open}>
	<div class="progress-modal">
		<div class="progress-header">
			<h3 class="progress-title">{title}</h3>
		</div>

		<div class="progress-body">
			{#if error}
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
</style>
