<script lang="ts">
	import type { PageDefinition, Field } from '$lib/types/pages';
	import FieldRenderer from './FieldRenderer.svelte';
	import Button from '$lib/components/Button.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import Card from '$lib/components/Card.svelte';
	import CustomizePageModal from './CustomizePageModal.svelte';
	import PlusIcon from '$lib/components/icons/PlusIcon.svelte';
	import EditIcon from '$lib/components/icons/EditIcon.svelte';
	import TrashIcon from '$lib/components/icons/TrashIcon.svelte';
	import RefreshIcon from '$lib/components/icons/RefreshIcon.svelte';
	import NavigationButtons from '$lib/components/NavigationButtons.svelte';
	import { shortcuts, createShortcutMap } from '$lib/utils/shortcuts';
	import { currentUser } from '$lib/stores/user';
	import { getFieldCaption } from '$lib/utils/fieldHelpers';
	import { loadPageCustomizations, savePageCustomizations } from '$lib/utils/customizationStorage';

	interface Props {
		page: PageDefinition;
		record?: Record<string, any>;
		captions?: Record<string, string>;
		onaction?: (actionName: string) => void;
		onsave?: (record: Record<string, any>) => void;
		navigationEnabled?: boolean;
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
		onaction,
		onsave,
		navigationEnabled = false,
		canNavigateFirst = false,
		canNavigatePrevious = false,
		canNavigateNext = false,
		canNavigateLast = false,
		onNavigateFirst,
		onNavigatePrevious,
		onNavigateNext,
		onNavigateLast
	}: Props = $props();

	// Field customization type
	interface FieldCustomization {
		visible: boolean;
		section?: string;
		order?: number;
	}

	// Customization state
	let customizeModalOpen = $state(false);
	let fieldCustomizations = $state<Record<string, FieldCustomization>>({});

	// Edit mode state
	let editMode = $state(false);

	// Load customizations from localStorage on mount
	$effect(() => {
		const userId = $currentUser?.user_id || 'anonymous';
		fieldCustomizations = loadPageCustomizations<Record<string, FieldCustomization>>(
			userId,
			page.page.id
		);
	});

	// Auto-save state
	let saveState = $state<'idle' | 'saving' | 'saved'>('idle');
	let saveTimeout: number | null = null;
	let savedTimeout: number | null = null;

	// Auto-save with debouncing
	function autoSave() {
		// Skip if already saving
		if (saveState === 'saving') {
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
				await onsave?.(record);
				saveState = 'saved';

				// Show "Saved" for 1.5 seconds then hide
				savedTimeout = setTimeout(() => {
					saveState = 'idle';
				}, 1500) as unknown as number;
			} catch (err) {
				saveState = 'idle';
				console.error('Auto-save failed:', err);
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

	function handleNew() {
		record = {};
	}

	function toggleEditMode() {
		editMode = !editMode;
	}

	// Build keyboard shortcut map from actions
	const shortcutMap = $derived(() => {
		const map: Record<string, () => void> = {};

		page.page.actions?.forEach((action) => {
			if (action.shortcut && action.enabled !== false) {
				map[action.shortcut] = () => handleAction(action.name);
			}
		});

		// Add navigation shortcuts if enabled
		if (navigationEnabled) {
			if (canNavigateFirst) map['Ctrl+Home'] = () => onNavigateFirst?.();
			if (canNavigatePrevious) map['Ctrl+ArrowUp'] = () => onNavigatePrevious?.();
			if (canNavigateNext) map['Ctrl+ArrowDown'] = () => onNavigateNext?.();
			if (canNavigateLast) map['Ctrl+End'] = () => onNavigateLast?.();
		}

		return map;
	});

	// Check if field should be visible based on customizations
	function isFieldVisible(field: Field): boolean {
		// If user has customized this field, use that preference
		if (field.source in fieldCustomizations) {
			return fieldCustomizations[field.source].visible;
		}
		// Otherwise use the field's visible property (default true)
		return field.visible !== false;
	}

	// Get customized sections (fields reorganized by user preferences)
	const customizedSections = $derived(() => {
		if (!page.page.layout.sections) return [];

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
				if (isFieldVisible(field)) {
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
	function handleSaveCustomizations(customizations: Record<string, FieldCustomization>) {
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

	<PageHeader title={page.page.caption}>
		<svelte:fragment slot="leftActions">
			{#if saveState === 'saving'}
				<div class="saving-indicator">
					<svg class="animate-spin h-4 w-4 text-blue-600" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
						<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
						<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
					</svg>
					<span class="text-sm text-gray-600 dark:text-gray-400">Saving...</span>
				</div>
			{:else if saveState === 'saved'}
				<div class="saved-indicator">
					<svg class="h-4 w-4 text-green-600" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
					</svg>
					<span class="text-sm text-green-600 dark:text-green-400 font-medium">Saved</span>
				</div>
			{/if}

			{#each page.page.actions?.filter((a) => a.promoted) || [] as action}
				{@const variant = action.name === 'Delete' ? 'danger' : action.name === 'New' ? 'success' : 'secondary'}
				<Button
					variant={variant}
					size="sm"
					onclick={() => {
						if (action.run_page) {
							window.location.href = `/pages/${action.run_page}`;
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
		</svelte:fragment>

		<svelte:fragment slot="rightActions">
			<!-- Customize button -->
			<Button variant="secondary" size="sm" onclick={handleCustomize} title="Customize page">
				<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4" />
				</svg>
				<span class="ml-1">Customize</span>
			</Button>

		</svelte:fragment>
	</PageHeader>

	<div class="sections-container">
		{#each customizedSections() as section}
			<Card>
				<svelte:fragment slot="header">
					<h3 class="text-lg font-semibold text-nav-blue dark:text-blue-400">{section.caption}</h3>
				</svelte:fragment>

				<div class="section-fields">
					{#each section.fields as field}
						<FieldRenderer
							{field}
							bind:value={record[field.source]}
							caption={getFieldCaption(field.source, captions, field.caption)}
							editable={field.editable && editMode}
							onblur={handleFieldBlur}
						/>
					{/each}
				</div>
			</Card>
		{/each}
	</div>
</div>

<!-- Customize Page Modal -->
{#if customizeModalOpen}
	<CustomizePageModal
		open={customizeModalOpen}
		{page}
		customizations={fieldCustomizations}
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

	.saving-indicator,
	.saved-indicator {
		@apply flex items-center gap-2 px-3 py-1.5;
	}

	.nav-buttons {
		@apply flex items-center gap-1;
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
</style>
