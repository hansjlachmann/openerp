<script lang="ts">
	import type { PageDefinition, Field } from '$lib/types/pages';
	import type { TableFilter } from '$lib/types/api';
	import Button from '$lib/components/Button.svelte';
	import Card from '$lib/components/Card.svelte';
	import { currentUser } from '$lib/stores/user';

	interface Props {
		page: PageDefinition;
		captions?: Record<string, string>;
		currentFilters?: TableFilter[];
		onApply: (filters: TableFilter[]) => void;
		onClose: () => void;
	}

	let { page, captions = {}, currentFilters = [], onApply, onClose }: Props = $props();

	// Saved filter presets (views)
	interface FilterPreset {
		name: string;
		filters: Array<{ field: string; expression: string }>;
	}

	let savedPresets = $state<Record<string, FilterPreset>>({});
	let activePresetName = $state<string | null>(null);
	let showPresetMenu = $state<string | null>(null);
	let editingPresetName = $state<string | null>(null);
	let newPresetNameInput = $state('');

	// Active filters - array of { field, expression }
	let activeFilters = $state<Array<{ field: string; expression: string }>>([]);

	// Initialize active filters from current filters
	$effect(() => {
		activeFilters = currentFilters.map((f) => ({ field: f.field, expression: f.expression }));
	});

	// Load saved presets on mount
	$effect(() => {
		const userId = $currentUser?.user_id || 'anonymous';
		const key = `filter-preset-${userId}-${page.page.id}`;
		const stored = localStorage.getItem(key);

		if (stored) {
			try {
				const presets = JSON.parse(stored);
				savedPresets = presets;
			} catch (e) {
				console.error('Failed to load filter presets:', e);
			}
		}
	});

	// Get all fields from page definition
	const allFields = $derived(() => {
		if (page.page.type === 'List' && page.page.layout.repeater?.fields) {
			return page.page.layout.repeater.fields;
		}
		return [];
	});

	// Get fields that haven't been added yet
	const availableFields = $derived(() => {
		const activeFieldSet = new Set(activeFilters.map((f) => f.field));
		return allFields().filter((field) => !activeFieldSet.has(field.source));
	});

	// Selected field for adding new filter
	let selectedFieldToAdd = $state('');

	// Get field caption by source
	function getFieldCaptionBySource(fieldSource: string): string {
		const field = allFields().find((f) => f.source === fieldSource);
		if (field) {
			return captions[field.source] || field.caption || field.source;
		}
		return fieldSource;
	}

	// Get field caption
	function getFieldCaption(field: Field): string {
		return captions[field.source] || field.caption || field.source;
	}

	// Add a new filter field
	function handleAddField() {
		if (!selectedFieldToAdd) return;

		activeFilters = [...activeFilters, { field: selectedFieldToAdd, expression: '' }];
		selectedFieldToAdd = '';
	}

	// Remove a filter field
	function handleRemoveFilter(index: number) {
		activeFilters = activeFilters.filter((_, i) => i !== index);
		activePresetName = null; // Clear active preset when manually changing filters
		applyFilters();
	}

	// Handle filter expression change
	function handleFilterChange(index: number, value: string) {
		activeFilters[index].expression = value;
		activePresetName = null; // Clear active preset when manually changing filters
	}

	// Apply filters
	function applyFilters() {
		const filters: TableFilter[] = activeFilters
			.filter((f) => f.expression && f.expression.trim() !== '')
			.map((f) => ({
				field: f.field,
				expression: f.expression.trim()
			}));

		onApply(filters);
	}

	// Clear all filters
	function handleClearAll() {
		activeFilters = [];
		activePresetName = null;
		onApply([]);
	}

	// Apply a saved preset
	function handleApplyPreset(presetName: string) {
		const preset = savedPresets[presetName];
		if (preset) {
			activeFilters = [...preset.filters];
			activePresetName = presetName;
			showPresetMenu = null;
			applyFilters();
		}
	}

	// Save current filters as new preset
	function handleSaveAsNewPreset() {
		const presetName = prompt('Enter a name for this view:');
		if (!presetName || presetName.trim() === '') return;

		const userId = $currentUser?.user_id || 'anonymous';
		const key = `filter-preset-${userId}-${page.page.id}`;

		savedPresets[presetName] = {
			name: presetName,
			filters: [...activeFilters]
		};

		localStorage.setItem(key, JSON.stringify(savedPresets));
		activePresetName = presetName;
	}

	// Start renaming a preset
	function handleStartRename(presetName: string) {
		editingPresetName = presetName;
		newPresetNameInput = presetName;
		showPresetMenu = null;
	}

	// Save renamed preset
	function handleSaveRename(oldName: string) {
		if (!newPresetNameInput || newPresetNameInput.trim() === '') {
			editingPresetName = null;
			return;
		}

		const newName = newPresetNameInput.trim();
		if (newName === oldName) {
			editingPresetName = null;
			return;
		}

		const userId = $currentUser?.user_id || 'anonymous';
		const key = `filter-preset-${userId}-${page.page.id}`;

		// Copy preset with new name
		savedPresets[newName] = {
			name: newName,
			filters: savedPresets[oldName].filters
		};

		// Delete old preset
		delete savedPresets[oldName];

		// Update active preset name if this was the active one
		if (activePresetName === oldName) {
			activePresetName = newName;
		}

		localStorage.setItem(key, JSON.stringify(savedPresets));
		editingPresetName = null;
	}

	// Cancel rename
	function handleCancelRename() {
		editingPresetName = null;
		newPresetNameInput = '';
	}

	// Delete a preset
	function handleDeletePreset(presetName: string) {
		if (!confirm(`Delete view "${presetName}"?`)) return;

		const userId = $currentUser?.user_id || 'anonymous';
		const key = `filter-preset-${userId}-${page.page.id}`;

		delete savedPresets[presetName];

		if (activePresetName === presetName) {
			activePresetName = null;
		}

		localStorage.setItem(key, JSON.stringify(savedPresets));
		showPresetMenu = null;
	}

	// Toggle preset menu
	function handleTogglePresetMenu(presetName: string, event: MouseEvent) {
		event.stopPropagation();
		showPresetMenu = showPresetMenu === presetName ? null : presetName;
	}

	// Handle Enter key to apply filters
	function handleKeyDown(e: KeyboardEvent) {
		if (e.key === 'Enter' && !e.shiftKey) {
			e.preventDefault();
			applyFilters();
		}
	}

	// Handle Enter key for rename
	function handleRenameKeyDown(e: KeyboardEvent, oldName: string) {
		if (e.key === 'Enter') {
			e.preventDefault();
			handleSaveRename(oldName);
		} else if (e.key === 'Escape') {
			e.preventDefault();
			handleCancelRename();
		}
	}

	// Close preset menu when clicking outside
	function handleClickOutside(event: MouseEvent) {
		const target = event.target as HTMLElement;
		if (!target.closest('.preset-menu-container')) {
			showPresetMenu = null;
		}
	}

	$effect(() => {
		document.addEventListener('click', handleClickOutside);
		return () => {
			document.removeEventListener('click', handleClickOutside);
		};
	});
</script>

<div class="filter-pane">
	<Card>
		<svelte:fragment slot="header">
			<div class="filter-header">
				<h3 class="filter-title">Filter {page.page.caption}</h3>
				<button onclick={onClose} class="close-btn" title="Close filter pane">
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
		</svelte:fragment>

		<!-- Views Section -->
		<div class="views-section">
			<h4 class="section-title">Views</h4>

			<!-- All (default view) -->
			<button
				class="view-item"
				class:active={activePresetName === null}
				onclick={() => handleClearAll()}
			>
				<span class="view-name">*All</span>
			</button>

			<!-- Saved presets -->
			{#each Object.keys(savedPresets) as presetName}
				{#if editingPresetName === presetName}
					<div class="view-item-editing">
						<input
							type="text"
							class="rename-input"
							bind:value={newPresetNameInput}
							onkeydown={(e) => handleRenameKeyDown(e, presetName)}
							onblur={() => handleSaveRename(presetName)}
							autofocus
						/>
					</div>
				{:else}
					<div class="view-item-container preset-menu-container">
						<button
							class="view-item"
							class:active={activePresetName === presetName}
							onclick={() => handleApplyPreset(presetName)}
						>
							<span class="view-name">{presetName}</span>
						</button>
						<button
							class="preset-menu-btn"
							onclick={(e) => handleTogglePresetMenu(presetName, e)}
							title="View options"
						>
							<svg
								xmlns="http://www.w3.org/2000/svg"
								class="h-4 w-4"
								fill="none"
								viewBox="0 0 24 24"
								stroke="currentColor"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="2"
									d="M12 5v.01M12 12v.01M12 19v.01M12 6a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2z"
								/>
							</svg>
						</button>

						{#if showPresetMenu === presetName}
							<div class="preset-menu">
								<button
									class="preset-menu-item"
									onclick={() => handleStartRename(presetName)}
								>
									<svg
										xmlns="http://www.w3.org/2000/svg"
										class="h-4 w-4"
										fill="none"
										viewBox="0 0 24 24"
										stroke="currentColor"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											stroke-width="2"
											d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
										/>
									</svg>
									Rename
								</button>
								<button
									class="preset-menu-item delete"
									onclick={() => handleDeletePreset(presetName)}
								>
									<svg
										xmlns="http://www.w3.org/2000/svg"
										class="h-4 w-4"
										fill="none"
										viewBox="0 0 24 24"
										stroke="currentColor"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											stroke-width="2"
											d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
										/>
									</svg>
									Delete
								</button>
							</div>
						{/if}
					</div>
				{/if}
			{/each}

			<!-- Save current as new view -->
			{#if activeFilters.length > 0}
				<button class="view-item new-view" onclick={handleSaveAsNewPreset}>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						class="h-4 w-4"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M12 4v16m8-8H4"
						/>
					</svg>
					<span class="view-name">Save as new view</span>
				</button>
			{/if}
		</div>

		<!-- Filter list by Section -->
		<div class="filter-section">
			<h4 class="section-title">Filter list by:</h4>

			<!-- Active filters -->
			{#if activeFilters.length > 0}
				<div class="active-filters">
					{#each activeFilters as filter, index}
						<div class="filter-row">
							<div class="filter-field-tag">
								<span class="filter-field-name"
									>{getFieldCaptionBySource(filter.field)}</span
								>
								<button
									class="remove-filter-icon"
									onclick={() => handleRemoveFilter(index)}
									title="Remove filter"
								>
									<svg
										xmlns="http://www.w3.org/2000/svg"
										class="h-3 w-3"
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
							<input
								type="text"
								class="filter-expression-input"
								value={filter.expression}
								oninput={(e) => handleFilterChange(index, e.currentTarget.value)}
								onkeydown={handleKeyDown}
								onblur={applyFilters}
								placeholder="Enter filter expression..."
							/>
						</div>
					{/each}
				</div>
			{/if}

			<!-- Add filter button -->
			{#if availableFields().length > 0}
				<div class="add-filter-row">
					<button class="add-filter-btn" onclick={() => (selectedFieldToAdd = 'show')}>
						<svg
							xmlns="http://www.w3.org/2000/svg"
							class="h-4 w-4"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M12 4v16m8-8H4"
							/>
						</svg>
						<span>Filter...</span>
					</button>

					{#if selectedFieldToAdd === 'show'}
						<select
							class="field-selector-inline"
							bind:value={selectedFieldToAdd}
							onchange={handleAddField}
							autofocus
						>
							<option value="">Select field...</option>
							{#each availableFields() as field}
								<option value={field.source}>{getFieldCaption(field)}</option>
							{/each}
						</select>
					{/if}
				</div>
			{/if}
		</div>

		<!-- Filter help -->
		<div class="filter-help">
			<p class="text-xs text-gray-500 dark:text-gray-400">
				<strong>Syntax:</strong> <code>*</code> wildcard, <code>|</code> OR, <code>..</code> range,
				<code>&lt;&gt;=</code> comparison
			</p>
		</div>
	</Card>
</div>

<style>
	.filter-pane {
		@apply w-80 flex-shrink-0;
	}

	.filter-header {
		@apply flex items-center justify-between;
	}

	.filter-title {
		@apply text-lg font-semibold text-nav-blue dark:text-blue-400;
	}

	.close-btn {
		@apply p-1 rounded hover:bg-gray-100 dark:hover:bg-gray-700;
		@apply text-gray-600 dark:text-gray-400;
		@apply transition-colors;
	}

	/* Views Section */
	.views-section {
		@apply mb-6 pb-4 border-b border-gray-200 dark:border-gray-700;
	}

	.section-title {
		@apply text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wide mb-2;
	}

	.view-item-container {
		@apply relative flex items-center gap-1;
	}

	.view-item {
		@apply w-full flex items-center gap-2 px-3 py-2 text-left rounded;
		@apply text-sm text-blue-600 dark:text-blue-400;
		@apply hover:bg-blue-50 dark:hover:bg-blue-900/20;
		@apply transition-colors;
	}

	.view-item.active {
		@apply bg-blue-100 dark:bg-blue-900/30 font-medium;
	}

	.view-item.new-view {
		@apply text-gray-600 dark:text-gray-400;
		@apply border border-dashed border-gray-300 dark:border-gray-600;
		@apply hover:border-blue-400 hover:text-blue-600;
	}

	.view-name {
		@apply flex-1;
	}

	.view-item-editing {
		@apply px-3 py-2;
	}

	.rename-input {
		@apply w-full px-2 py-1 text-sm;
		@apply border border-blue-500 rounded;
		@apply bg-white dark:bg-gray-800;
		@apply text-gray-900 dark:text-gray-100;
		@apply focus:outline-none focus:ring-2 focus:ring-blue-500;
	}

	.preset-menu-btn {
		@apply p-1 rounded opacity-0 group-hover:opacity-100;
		@apply hover:bg-gray-200 dark:hover:bg-gray-600;
		@apply text-gray-500 dark:text-gray-400;
		@apply transition-opacity;
	}

	.view-item-container:hover .preset-menu-btn {
		@apply opacity-100;
	}

	.preset-menu {
		@apply absolute right-0 top-full mt-1 z-50;
		@apply bg-white dark:bg-gray-800 rounded-md shadow-lg;
		@apply border border-gray-200 dark:border-gray-700;
		@apply py-1 min-w-[150px];
	}

	.preset-menu-item {
		@apply w-full flex items-center gap-2 px-3 py-2 text-left;
		@apply text-sm text-gray-700 dark:text-gray-300;
		@apply hover:bg-gray-100 dark:hover:bg-gray-700;
		@apply transition-colors;
	}

	.preset-menu-item.delete {
		@apply text-red-600 dark:text-red-400;
		@apply hover:bg-red-50 dark:hover:bg-red-900/20;
	}

	/* Filter Section */
	.filter-section {
		@apply mb-4;
	}

	.active-filters {
		@apply space-y-2 mb-3;
	}

	.filter-row {
		@apply space-y-1;
	}

	.filter-field-tag {
		@apply flex items-center gap-2 text-sm;
		@apply text-gray-700 dark:text-gray-300;
	}

	.filter-field-name {
		@apply flex items-center gap-1;
	}

	.filter-field-name::before {
		content: '×';
		@apply text-gray-400 dark:text-gray-500 text-lg;
	}

	.remove-filter-icon {
		@apply p-0.5 rounded hover:bg-gray-200 dark:hover:bg-gray-600;
		@apply text-gray-400 dark:text-gray-500;
		@apply opacity-0 hover:opacity-100;
	}

	.filter-row:hover .remove-filter-icon {
		@apply opacity-100;
	}

	.filter-expression-input {
		@apply w-full px-3 py-2 text-sm;
		@apply border border-gray-300 dark:border-gray-600 rounded;
		@apply bg-white dark:bg-gray-800;
		@apply text-gray-900 dark:text-gray-100;
		@apply focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent;
	}

	.add-filter-row {
		@apply flex flex-col gap-2;
	}

	.add-filter-btn {
		@apply flex items-center gap-2 px-3 py-2 text-left;
		@apply text-sm text-blue-600 dark:text-blue-400;
		@apply hover:bg-blue-50 dark:hover:bg-blue-900/20;
		@apply rounded transition-colors;
	}

	.field-selector-inline {
		@apply w-full px-3 py-2 text-sm;
		@apply border border-blue-500 rounded;
		@apply bg-white dark:bg-gray-800;
		@apply text-gray-900 dark:text-gray-100;
		@apply focus:outline-none focus:ring-2 focus:ring-blue-500;
	}

	/* Filter Help */
	.filter-help {
		@apply pt-3 border-t border-gray-200 dark:border-gray-700;
	}

	.filter-help code {
		@apply px-1 py-0.5 bg-gray-100 dark:bg-gray-700 rounded;
		@apply text-xs font-mono;
	}
</style>
