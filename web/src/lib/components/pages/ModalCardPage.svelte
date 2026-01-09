<script lang="ts">
	import Modal from '$lib/components/Modal.svelte';
	import CardPage from './CardPage.svelte';
	import NavigationButtons from '$lib/components/NavigationButtons.svelte';
	import type { PageDefinition } from '$lib/types/pages';
	import { api } from '$lib/services/api';
	import { onMount } from 'svelte';

	interface Props {
		open?: boolean;
		page: PageDefinition;
		record?: Record<string, any>;
		captions?: Record<string, string>;
		onclose?: () => void;
		onaction?: (actionName: string) => void;
		onsave?: (record: Record<string, any>) => void;
	}

	let {
		open = false,
		page,
		record = $bindable({}),
		captions = {},
		onclose,
		onaction,
		onsave
	}: Props = $props();

	// Modal size state: normal, expanded, fullscreen
	let modalSize = $state<'normal' | 'expanded' | 'fullscreen'>('expanded');

	// Navigation state
	let recordIds: string[] = $state([]);
	let currentRecordIndex = $state(-1);
	let recordIdsLoaded = $state(false);

	// Computed navigation button states
	const canNavigatePrevious = $derived(recordIdsLoaded && currentRecordIndex > 0);
	const canNavigateNext = $derived(
		recordIdsLoaded && currentRecordIndex >= 0 && currentRecordIndex < recordIds.length - 1
	);

	// Load record IDs for navigation when modal opens
	$effect(() => {
		if (open && page.page.enable_navigation && !recordIdsLoaded) {
			loadRecordIds();
		} else if (!open) {
			// Reset when modal closes
			recordIdsLoaded = false;
			recordIds = [];
			currentRecordIndex = -1;
		}
	});

	// Update current record index when record changes
	$effect(() => {
		if (recordIdsLoaded && record) {
			const currentRecordId = record['no'] || record['code'] || record['id'];
			if (currentRecordId) {
				currentRecordIndex = recordIds.indexOf(currentRecordId);
			}
		}
	});

	// Keyboard shortcuts for navigation
	$effect(() => {
		if (!open || !page.page.enable_navigation) return;

		function handleKeyDown(e: KeyboardEvent) {
			// Ctrl+ArrowUp or Ctrl+Up - Previous
			if (e.ctrlKey && (e.key === 'ArrowUp' || e.key === 'Up')) {
				e.preventDefault();
				if (canNavigatePrevious) navigatePrevious();
			}
			// Ctrl+ArrowDown or Ctrl+Down - Next
			else if (e.ctrlKey && (e.key === 'ArrowDown' || e.key === 'Down')) {
				e.preventDefault();
				if (canNavigateNext) navigateNext();
			}
			// Ctrl+Home - First
			else if (e.ctrlKey && e.key === 'Home') {
				e.preventDefault();
				if (recordIdsLoaded && recordIds.length > 0) navigateFirst();
			}
			// Ctrl+End - Last
			else if (e.ctrlKey && e.key === 'End') {
				e.preventDefault();
				if (recordIdsLoaded && recordIds.length > 0) navigateLast();
			}
		}

		window.addEventListener('keydown', handleKeyDown);
		return () => window.removeEventListener('keydown', handleKeyDown);
	});

	async function loadRecordIds() {
		try {
			// Use lightweight IDs-only endpoint
			recordIds = await api.getRecordIDs(page.page.source_table);

			// Find current record index
			const currentRecordId = record['no'] || record['code'] || record['id'];
			currentRecordIndex = recordIds.indexOf(currentRecordId);

			recordIdsLoaded = true;
		} catch (err) {
			console.error('Error loading record IDs for navigation:', err);
			recordIdsLoaded = true; // Set to true even on error to prevent retries
		}
	}

	// Handle pop-out to new window
	function handlePopOut() {
		const recordId = record['no'] || record['code'] || record['id'];
		const url = `/pages/${page.page.id}${recordId ? `/${recordId}` : ''}`;
		window.open(url, '_blank', 'width=1200,height=800');
		onclose?.();
	}

	// Toggle fullscreen
	function toggleFullscreen() {
		modalSize = modalSize === 'fullscreen' ? 'expanded' : 'fullscreen';
	}

	// Navigation functions - load record without changing URL
	async function navigateToRecord(recordId: string) {
		try {
			const newRecord = await api.getRecord(page.page.source_table, recordId);
			record = newRecord;
			currentRecordIndex = recordIds.indexOf(recordId);
		} catch (err) {
			console.error('Error loading record:', err);
		}
	}

	function navigateFirst() {
		if (recordIds.length > 0) {
			navigateToRecord(recordIds[0]);
		}
	}

	function navigatePrevious() {
		if (currentRecordIndex > 0) {
			navigateToRecord(recordIds[currentRecordIndex - 1]);
		}
	}

	function navigateNext() {
		if (currentRecordIndex < recordIds.length - 1) {
			navigateToRecord(recordIds[currentRecordIndex + 1]);
		}
	}

	function navigateLast() {
		if (recordIds.length > 0) {
			navigateToRecord(recordIds[recordIds.length - 1]);
		}
	}
</script>

<Modal {open} onclose={onclose} size={modalSize}>
	<!-- Edge Navigation Buttons (Business Central style) -->
	{#if page.page.enable_navigation && recordIdsLoaded}
		<NavigationButtons
			onPrevious={navigatePrevious}
			onNext={navigateNext}
			canNavigatePrevious={canNavigatePrevious}
			canNavigateNext={canNavigateNext}
		/>
	{/if}

	<div class="modal-header">
		<h2 class="modal-title">{page.page.caption}</h2>

		<div class="modal-controls">
			<!-- Fullscreen toggle -->
			<button
				onclick={toggleFullscreen}
				class="control-btn"
				title={modalSize === 'fullscreen' ? 'Exit fullscreen' : 'Fullscreen'}
				aria-label={modalSize === 'fullscreen' ? 'Exit fullscreen' : 'Fullscreen'}
			>
				{#if modalSize === 'fullscreen'}
					<!-- Exit fullscreen icon -->
					<svg
						xmlns="http://www.w3.org/2000/svg"
						class="h-5 w-5"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M6 18L18 6M6 6l12 12"
						/>
					</svg>
				{:else}
					<!-- Fullscreen icon -->
					<svg
						xmlns="http://www.w3.org/2000/svg"
						class="h-5 w-5"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4"
						/>
					</svg>
				{/if}
			</button>

			<!-- Pop-out to new window -->
			<button
				onclick={handlePopOut}
				class="control-btn"
				title="Open in new window"
				aria-label="Open in new window"
			>
				<svg
					xmlns="http://www.w3.org/2000/svg"
					class="h-5 w-5"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
					/>
				</svg>
			</button>

			<!-- Close button -->
			<button onclick={onclose} class="control-btn" title="Close" aria-label="Close">
				<svg
					xmlns="http://www.w3.org/2000/svg"
					class="h-5 w-5"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M6 18L18 6M6 6l12 12"
					/>
				</svg>
			</button>
		</div>
	</div>

	<div class="modal-body">
		<!-- Keyboard shortcuts hint -->
		{#if page.page.enable_navigation && recordIdsLoaded}
			<div class="keyboard-hint">
				<span class="text-xs text-gray-500 dark:text-gray-400">
					<kbd>Ctrl+↑/↓</kbd> Navigate • <kbd>Ctrl+Home/End</kbd> First/Last
				</span>
			</div>
		{/if}

		<CardPage
			{page}
			bind:record
			{captions}
			{onaction}
			{onsave}
			navigationEnabled={false}
		/>
	</div>
</Modal>

<style>
	.modal-header {
		@apply flex items-center justify-between px-6 py-4;
		@apply border-b border-gray-200;
		@apply bg-white;
		@apply shrink-0;
	}

	:global(.dark) .modal-header {
		border-color: #374151; /* gray-700 */
		background-color: #1f2937; /* gray-800 */
	}

	.modal-title {
		@apply text-xl font-bold text-nav-blue;
	}

	:global(.dark) .modal-title {
		color: #60a5fa; /* blue-400 */
	}

	.modal-controls {
		@apply flex items-center gap-2;
	}

	.control-btn {
		@apply p-2 rounded;
		@apply text-gray-600;
		@apply transition-colors;
	}

	.control-btn:hover {
		@apply bg-gray-100;
		@apply text-gray-900;
	}

	:global(.dark) .control-btn {
		color: #9ca3af; /* gray-400 */
	}

	:global(.dark) .control-btn:hover {
		background-color: #374151; /* gray-700 */
		color: #f3f4f6; /* gray-100 */
	}

	.modal-body {
		@apply flex-1 flex flex-col p-6;
		@apply bg-gray-50;
		@apply overflow-hidden; /* Prevent modal-body from scrolling */
		@apply relative; /* For keyboard-hint positioning */
	}

	:global(.dark) .modal-body {
		background-color: #111827; /* gray-900 */
	}

	.keyboard-hint {
		@apply absolute top-4 right-4 z-40;
		@apply bg-white;
		@apply px-3 py-1.5 rounded-md;
		@apply shadow-sm border border-gray-200;
		@apply opacity-70 hover:opacity-100;
		@apply transition-opacity duration-200;
	}

	:global(.dark) .keyboard-hint {
		background-color: #1f2937; /* gray-800 */
		border-color: #374151; /* gray-700 */
	}

	.keyboard-hint kbd {
		@apply bg-gray-100;
		@apply px-1.5 py-0.5 rounded;
		@apply text-xs font-mono;
		@apply border border-gray-300;
	}

	:global(.dark) .keyboard-hint kbd {
		background-color: #374151; /* gray-700 */
		border-color: #4b5563; /* gray-600 */
	}


	/* Hover effect */
	:global(.edge-nav-btn:not(:disabled):hover) {
		@apply shadow-xl;
	}
</style>
