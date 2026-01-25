<script lang="ts">
	import type { PageDefinition, Field } from '$lib/types/pages';
	import type { LookupData } from '$lib/types/api';
	import FieldRenderer from './FieldRenderer.svelte';
	import Button from '$lib/components/Button.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import Card from '$lib/components/Card.svelte';
	import CustomizeFieldsModal from './CustomizeFieldsModal.svelte';
	import PlusIcon from '$lib/components/icons/PlusIcon.svelte';
	import EditIcon from '$lib/components/icons/EditIcon.svelte';
	import TrashIcon from '$lib/components/icons/TrashIcon.svelte';
	import RefreshIcon from '$lib/components/icons/RefreshIcon.svelte';
	import NavigationButtons from '$lib/components/NavigationButtons.svelte';
	import { shortcuts, createShortcutMap, normalizeShortcut } from '$lib/utils/shortcuts';
	import { currentUser } from '$lib/stores/user';
	import { getFieldCaption, isItemVisible, type ItemCustomization } from '$lib/utils/fieldHelpers';
	import { loadPageCustomizations, savePageCustomizations } from '$lib/utils/customizationStorage';
	import { getRecordId, isNewRecord, deepCopy, getPrimaryKeyField } from '$lib/utils/recordHelpers';
	import { toast } from '$lib/stores/toast';
	import { api } from '$lib/services/api';

	interface Props {
		page: PageDefinition;
		record?: Record<string, any>;
		captions?: Record<string, string>;
		options?: Record<string, Record<string, string>>; // Option field values (enum lookups)
		lookups?: Record<string, LookupData>; // Table relation lookup values
		onaction?: (actionName: string) => void;
		onsave?: (record: Record<string, any>) => Promise<boolean> | boolean | void;
		onclose?: () => void; // Callback when closing the card
		saveBlocked?: boolean; // Block editing due to save error (e.g., record already exists)
		saveBlockedMessage?: string; // Error message to display when blocked
		onclearerror?: () => void; // Callback to clear the error state
		navigationEnabled?: boolean;
		initialEditMode?: boolean;
		canNavigateFirst?: boolean;
		canNavigatePrevious?: boolean;
		canNavigateNext?: boolean;
		canNavigateLast?: boolean;
		onNavigateFirst?: () => void;
		onNavigatePrevious?: () => void;
		onNavigateNext?: () => void;
		onNavigateLast?: () => void;
	}

	let {
		page,
		record = $bindable({}),
		captions = {},
		options = {},
		lookups = {},
		onaction,
		onsave,
		onclose,
		saveBlocked = false,
		saveBlockedMessage = '',
		onclearerror,
		navigationEnabled = false,
		initialEditMode,
		canNavigateFirst = false,
		canNavigatePrevious = false,
		canNavigateNext = false,
		canNavigateLast = false,
		onNavigateFirst,
		onNavigatePrevious,
		onNavigateNext,
		onNavigateLast
	}: Props = $props();

	// Get primary key field name from page definition
	const primaryKeyField = $derived(getPrimaryKeyField(page));

	// Customization state
	let customizeModalOpen = $state(false);
	let fieldCustomizations = $state<Record<string, ItemCustomization>>({});


	// Check if record is empty (no data loaded)
	const isEmptyRecord = $derived(() => {
		if (!record) return true;
		const keys = Object.keys(record).filter(k => !k.startsWith('_'));
		return keys.length === 0;
	});

	// Check if this is a new record (no ID)
	function checkIsNewRecord(): boolean {
		return isNewRecord(record);
	}

	// Edit mode state - start in edit mode for new records or if initialEditMode is true
	// svelte-ignore state_referenced_locally - initialEditMode is intentionally only used for initialization
	let editMode = $state(initialEditMode ?? checkIsNewRecord());

	// Track if we've already focused the initial field (to avoid refocusing on every state change)
	let initialFocusDone = $state(false);

	// Track the last known record ID to detect when a truly new record is loaded
	let lastRecordId = $state<string | undefined>(undefined);

	// Auto-enable edit mode when a new record is loaded (not when field values change)
	$effect(() => {
		const id = getRecordId(record, primaryKeyField);
		// Only trigger when the record ID actually changes (not on every field change)
		if (id !== lastRecordId) {
			lastRecordId = id;
			if (!id) {
				editMode = true;
				// Reset focus flag only when loading a truly new record
				initialFocusDone = false;
			}
		}
	});

	// Focus the configured field when page opens in edit mode (only once per new record)
	$effect(() => {
		if (!page?.page) return; // Guard during cleanup
		if (editMode && page.page.focus_field && !initialFocusDone) {
			initialFocusDone = true;
			// Use longer timeout to ensure modal animation completes and DOM is ready
			setTimeout(() => {
				const input = document.getElementById(page.page.focus_field!);
				if (input) {
					input.focus();
					// Select all text if it's an input
					if (input instanceof HTMLInputElement) {
						input.select();
					}
				}
			}, 250);
		}
	});

	// Load customizations from localStorage on mount
	$effect(() => {
		if (!page?.page) return; // Guard during cleanup
		const userId = $currentUser?.user_id || 'anonymous';
		fieldCustomizations = loadPageCustomizations<Record<string, ItemCustomization>>(
			userId,
			page.page.id
		);
	});

	// Auto-save state
	let saveState = $state<'idle' | 'saving' | 'saved'>('idle');
	let saveTimeout: number | null = null;
	let savedTimeout: number | null = null;

	// Track last saved record state for reverting on validation error
	let lastSavedRecord = $state<Record<string, any>>({});

	// Update lastSavedRecord when record is loaded from parent
	$effect(() => {
		const id = getRecordId(record, primaryKeyField);
		// When a record is first loaded (has ID and we don't have it saved yet)
		if (id && (!lastSavedRecord || getRecordId(lastSavedRecord, primaryKeyField) !== id)) {
			lastSavedRecord = deepCopy(record);
		}
	});

	// Auto-save with debouncing
	function autoSave() {
		// Skip if already saving or blocked due to error
		if (saveState === 'saving' || saveBlocked) {
			return;
		}

		// Clear any pending timeouts
		if (saveTimeout) {
			clearTimeout(saveTimeout);
		}
		if (savedTimeout) {
			clearTimeout(savedTimeout);
		}

		// Debounce: wait 300ms after last change before saving
		saveTimeout = setTimeout(async () => {
			// Double-check save state before proceeding
			if (saveState === 'saving') {
				return;
			}

			saveState = 'saving';
			try {
				// onsave returns true if a save actually happened
				const didSave = await onsave?.(record);

				if (didSave) {
					// Update lastSavedRecord after successful save
					lastSavedRecord = deepCopy(record);
					saveState = 'saved';
					// Show "Saved" for 1.5 seconds then hide
					savedTimeout = setTimeout(() => {
						saveState = 'idle';
					}, 1500) as unknown as number;
				} else {
					// No actual save happened (no changes)
					saveState = 'idle';
				}
			} catch (err) {
				saveState = 'idle';
				// Validation error - revert to last saved state
				const errorMessage = err instanceof Error ? err.message : 'Validation failed';
				toast.error(errorMessage);

				// Revert record to last saved state
				if (lastSavedRecord && Object.keys(lastSavedRecord).length > 0) {
					// Copy each field back to revert changes
					for (const key of Object.keys(record)) {
						if (key in lastSavedRecord) {
							record[key] = lastSavedRecord[key];
						}
					}
				}
			}
		}, 300) as unknown as number;
	}

	// Handle field blur (when user leaves a field)
	function handleFieldBlur() {
		autoSave();
	}

	// Handle action clicks
	function handleAction(actionName: string) {
		// Handle built-in actions locally first
		switch (actionName) {
			case 'Edit':
				toggleEditMode();
				return;
			case 'New':
				handleNew();
				return;
		}

		// Pass all other actions to parent (including Delete)
		onaction?.(actionName);
	}

	// Handle run_object actions (codeunits)
	async function handleRunObject(runObject: string) {
		// Parse the run_object string (format: "codeunit:ID")
		const [objectType, objectId] = runObject.split(':');

		if (objectType !== 'codeunit') {
			toast.error(`Unknown object type: ${objectType}`);
			return;
		}

		const codeunitId = parseInt(objectId, 10);
		if (isNaN(codeunitId)) {
			toast.error(`Invalid codeunit ID: ${objectId}`);
			return;
		}

		// Check if we have a record
		if (!record || Object.keys(record).length === 0) {
			toast.error('No record selected');
			return;
		}

		try {
			// Send the full record to the generic codeunit endpoint
			const result = await api.runCodeunit(codeunitId, record);
			toast.success(result.message || 'Operation completed successfully');

			// Trigger refresh to show updated data
			onaction?.('Refresh');
		} catch (err) {
			const errorMessage = err instanceof Error ? err.message : 'Operation failed';
			toast.error(errorMessage);
		}
	}

	function handleNew() {
		record = {};
	}

	function toggleEditMode() {
		editMode = !editMode;
	}

	// Build keyboard shortcut map from actions
	const shortcutMap = $derived(() => {
		const map: Record<string, () => void> = {};

		// Guard against page being undefined during cleanup
		if (!page?.page) return map;

		page.page.actions?.forEach((action) => {
			if (action.shortcut && action.enabled !== false) {
				// Normalize shortcut (e.g., "Esc" -> "Escape") to match keyboard event key names
				const normalizedShortcut = normalizeShortcut(action.shortcut);
				map[normalizedShortcut] = () => handleAction(action.name);
			}
		});

		// Add navigation shortcuts if enabled
		if (navigationEnabled) {
			if (canNavigateFirst) map['Ctrl+Home'] = () => onNavigateFirst?.();
			if (canNavigatePrevious) map['Ctrl+ArrowUp'] = () => onNavigatePrevious?.();
			if (canNavigateNext) map['Ctrl+ArrowDown'] = () => onNavigateNext?.();
			if (canNavigateLast) map['Ctrl+End'] = () => onNavigateLast?.();
		}

		// Add Escape to close
		if (onclose) {
			map['Escape'] = () => onclose();
		}

		return map;
	});

	// Get customized sections (fields reorganized by user preferences)
	const customizedSections = $derived(() => {
		if (!page?.page?.layout?.sections) return [];

		// Create a map of section names to fields
		const sectionMap = new Map<string, Field[]>();

		// Initialize with empty arrays for all sections
		page.page.layout.sections.forEach(section => {
			sectionMap.set(section.name, []);
		});

		// Distribute fields to sections based on customizations
		let globalIndex = 0;
		page.page.layout.sections.forEach(section => {
			section.fields.forEach(field => {
				// Get the target section (customized or original)
				const targetSection =
					(field.source in fieldCustomizations && fieldCustomizations[field.source].section)
						? fieldCustomizations[field.source].section!
						: section.name;

				// Get order (customized or original index)
				const order = fieldCustomizations[field.source]?.order ?? globalIndex;
				globalIndex++;

				// Add field to target section if visible with order info
				if (isItemVisible(field, fieldCustomizations)) {
					const fields = sectionMap.get(targetSection) || [];
					fields.push({ field, order } as any);
					sectionMap.set(targetSection, fields as any);
				}
			});
		});

		// Convert map back to section array (only include non-empty sections)
		// Sort fields within each section by order
		return page.page.layout.sections
			.map(section => ({
				...section,
				fields: ((sectionMap.get(section.name) || []) as any[])
					.sort((a: any, b: any) => a.order - b.order)
					.map((item: any) => item.field)
			}))
			.filter(section => section.fields.length > 0);
	});

	// Open customize modal
	function handleCustomize() {
		customizeModalOpen = true;
	}

	// Save customizations
	function handleSaveCustomizations(customizations: Record<string, ItemCustomization>) {
		fieldCustomizations = customizations;
		const userId = $currentUser?.user_id || 'anonymous';
		savePageCustomizations(userId, page.page.id, customizations);
	}
</script>

<div class="card-page" use:shortcuts={shortcutMap()}>
	<!-- Keyboard shortcuts hint -->
	{#if navigationEnabled}
		<div class="keyboard-hint">
			<span class="text-xs text-gray-500 dark:text-gray-400">
				<kbd>Ctrl+↑/↓</kbd> Navigate • <kbd>Ctrl+Home/End</kbd> First/Last
			</span>
		</div>
	{/if}

	<!-- Edge Navigation Buttons (Business Central style) -->
	{#if navigationEnabled}
		<NavigationButtons
			onPrevious={onNavigatePrevious}
			onNext={onNavigateNext}
			canNavigatePrevious={canNavigatePrevious}
			canNavigateNext={canNavigateNext}
		/>
	{/if}

	<PageHeader title={page.page.caption} {onclose}>
		{#snippet leftActions()}
			<!-- Save state indicator - fixed width container to prevent layout shift -->
			<div class="save-state-container">
				<div class="saving-indicator" class:visible={saveState === 'saving'}>
					<svg class="animate-spin h-4 w-4 text-blue-600" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
						<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
						<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
					</svg>
					<span class="text-sm text-gray-600 dark:text-gray-400">Saving...</span>
				</div>
				<div class="saved-indicator" class:visible={saveState === 'saved'}>
					<svg class="h-4 w-4 text-green-600" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
					</svg>
					<span class="text-sm text-green-600 dark:text-green-400 font-medium">Saved</span>
				</div>
			</div>

			{#each page.page.actions?.filter((a) => a.promoted) || [] as action}
				{@const variant = 'secondary'}
				<Button
					variant={variant}
					size="sm"
					tabindex={-1}
					onclick={() => {
						if (action.run_page) {
							window.location.href = `/pages/${action.run_page}`;
						} else if (action.run_object) {
							handleRunObject(action.run_object);
						} else {
							handleAction(action.name);
						}
					}}
					disabled={action.enabled === false}
				>
					{#snippet icon()}
						{#if action.name === 'New'}
							<PlusIcon size={16} color="currentColor" />
						{:else if action.name === 'Edit'}
							<EditIcon size={16} color="currentColor" />
						{:else if action.name === 'Delete'}
							<TrashIcon size={16} color="currentColor" />
						{:else if action.name === 'Refresh'}
							<RefreshIcon size={16} color="currentColor" />
						{/if}
					{/snippet}
					{action.caption}
					{#if action.shortcut}
						<span class="ml-2 text-xs opacity-70">{action.shortcut}</span>
					{/if}
				</Button>
			{/each}
		{/snippet}

		{#snippet rightActions()}
			<!-- Customize button -->
			<Button variant="secondary" size="sm" tabindex={-1} onclick={handleCustomize} title="Customize page">
				<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4" />
				</svg>
				<span class="ml-1">Customize</span>
			</Button>

		{/snippet}
	</PageHeader>

	<!-- Error banner when save is blocked -->
	{#if saveBlocked}
		<div class="error-banner">
			<div class="error-content">
				<svg class="error-icon" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
				</svg>
				<span class="error-message">{saveBlockedMessage || 'Save failed. Please correct the error before continuing.'}</span>
			</div>
			<div class="error-actions">
				<button class="error-clear-btn" onclick={() => onclearerror?.()}>
					Clear Form
				</button>
			</div>
		</div>
	{/if}

	<div class="sections-container">
		{#if isEmptyRecord() && !editMode}
			<!-- Empty state -->
			<div class="empty-state">
				<div class="empty-state-icon">
					<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
					</svg>
				</div>
				<h3 class="empty-state-title">No record loaded</h3>
				<p class="empty-state-text">Select a record from the list or create a new one.</p>
				<Button variant="primary" onclick={() => handleAction('New')}>
					{#snippet icon()}
						<PlusIcon size={16} color="currentColor" />
					{/snippet}
					Create New
				</Button>
			</div>
		{:else}
			{@const allFields = customizedSections().flatMap(s => s.fields)}
			{#each customizedSections() as section}
				<Card>
					{#snippet header()}
						<h3 class="text-lg font-semibold text-nav-blue dark:text-blue-400">
							{section.caption}
						</h3>
					{/snippet}

					<div class="section-fields">
						{#each section.fields as field (field.source)}
							{@const fieldEditable = field.editable !== false && editMode && !saveBlocked}
							{@const fieldTabIndex = allFields.findIndex(f => f.source === field.source) + 1}
							<FieldRenderer
								{field}
								bind:value={record[field.source]}
								caption={getFieldCaption(field.source, captions, field.caption)}
								editable={fieldEditable}
								options={options[field.source]}
								lookups={lookups[field.source]}
								fieldCaptions={captions}
								tabindex={fieldTabIndex}
								onblur={handleFieldBlur}
							/>
						{/each}
					</div>
				</Card>
			{/each}
		{/if}
	</div>
</div>

<!-- Customize Page Modal -->
{#if customizeModalOpen}
	<CustomizeFieldsModal
		open={customizeModalOpen}
		{page}
		customizations={fieldCustomizations}
		mode="card"
		onclose={() => customizeModalOpen = false}
		onsave={handleSaveCustomizations}
	/>
{/if}

<style>
	.card-page {
		@apply flex flex-col flex-1 min-h-0;
		@apply relative; /* For keyboard-hint positioning */
		@apply bg-white;
	}

	:global(.dark) .card-page {
		background-color: #111827; /* gray-900 */
	}

	.keyboard-hint {
		@apply absolute top-4 right-4 z-40;
		@apply bg-white dark:bg-gray-800;
		@apply px-3 py-1.5 rounded-md;
		@apply shadow-sm border border-gray-200 dark:border-gray-700;
		@apply opacity-70 hover:opacity-100;
		@apply transition-opacity duration-200;
	}

	.keyboard-hint kbd {
		@apply bg-gray-100 dark:bg-gray-700;
		@apply px-1.5 py-0.5 rounded;
		@apply text-xs font-mono;
		@apply border border-gray-300 dark:border-gray-600;
	}

	.sections-container {
		@apply flex flex-col gap-4 overflow-y-auto flex-1 min-h-0;
		@apply px-1; /* Small padding for scrollbar spacing */
	}

	.section-fields {
		@apply grid grid-cols-1 md:grid-cols-2 gap-4;
	}

	/* Save state container - fixed width to prevent layout shift */
	.save-state-container {
		@apply relative;
		width: 80px; /* Fixed width to accommodate "Saving..." text */
		height: 32px;
	}

	.saving-indicator,
	.saved-indicator {
		@apply flex items-center gap-2;
		@apply absolute inset-0;
		@apply opacity-0 transition-opacity duration-200;
		pointer-events: none;
	}

	.saving-indicator.visible,
	.saved-indicator.visible {
		@apply opacity-100;
		pointer-events: auto;
	}

	/* Empty state styles */
	.empty-state {
		@apply flex flex-col items-center justify-center;
		@apply py-16 px-8;
		@apply text-center;
	}

	.empty-state-icon {
		@apply w-16 h-16 mb-4;
		@apply text-gray-300;
	}

	:global(.dark) .empty-state-icon {
		color: #4b5563; /* gray-600 */
	}

	.empty-state-icon svg {
		@apply w-full h-full;
	}

	.empty-state-title {
		@apply text-xl font-semibold text-gray-700 mb-2;
	}

	:global(.dark) .empty-state-title {
		color: #d1d5db; /* gray-300 */
	}

	.empty-state-text {
		@apply text-gray-500 mb-6;
	}

	:global(.dark) .empty-state-text {
		color: #9ca3af; /* gray-400 */
	}

	.card-page :global(.edge-nav-btn:not(:disabled):hover) {
		@apply shadow-xl;
	}

	/* Ensure PageHeader stays at top */
	.card-page :global(.page-header) {
		@apply sticky top-0 z-10 bg-white;
		@apply mb-4;
	}

	:global(.dark) .card-page :global(.page-header) {
		background-color: #111827; /* gray-900 */
	}

	/* Error banner styles */
	.error-banner {
		@apply flex items-center justify-between;
		@apply mx-4 mb-4 px-4 py-3;
		@apply bg-red-50 border border-red-200 rounded-lg;
	}

	:global(.dark) .error-banner {
		background-color: rgba(127, 29, 29, 0.3); /* red-900 with opacity */
		border-color: #991b1b; /* red-800 */
	}

	.error-content {
		@apply flex items-center gap-3;
	}

	.error-icon {
		@apply w-5 h-5 text-red-600 flex-shrink-0;
	}

	:global(.dark) .error-icon {
		color: #f87171; /* red-400 */
	}

	.error-message {
		@apply text-sm text-red-700 font-medium;
	}

	:global(.dark) .error-message {
		color: #fca5a5; /* red-300 */
	}

	.error-actions {
		@apply flex items-center gap-2;
	}

	.error-clear-btn {
		@apply px-3 py-1.5 text-sm font-medium;
		@apply bg-red-600 text-white rounded;
		@apply hover:bg-red-700 transition-colors;
	}

	:global(.dark) .error-clear-btn {
		background-color: #dc2626; /* red-600 */
	}

	:global(.dark) .error-clear-btn:hover {
		background-color: #b91c1c; /* red-700 */
	}
</style>
