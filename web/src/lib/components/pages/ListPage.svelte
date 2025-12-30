<script lang="ts">
	import type { PageDefinition, Field } from '$lib/types/pages';
	import type { TableFilter } from '$lib/types/api';
	import Button from '$lib/components/Button.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import ModalCardPage from './ModalCardPage.svelte';
	import CustomizeListPageModal from './CustomizeListPageModal.svelte';
	import FilterPane from './FilterPane.svelte';
	import { shortcuts } from '$lib/utils/shortcuts';
	import { cn } from '$lib/utils/cn';
	import { api } from '$lib/services/api';

	interface Props {
		page: PageDefinition;
		records?: Array<Record<string, any>>;
		captions?: Record<string, string>;
		currentFilters?: TableFilter[];
		onaction?: (actionName: string, record?: Record<string, any>) => void;
		onrowclick?: (record: Record<string, any>) => void;
		onsave?: (record: Record<string, any>, isNew: boolean) => Promise<void>;
		ondelete?: (record: Record<string, any>) => Promise<void>;
		onfilter?: (filters: TableFilter[]) => void;
	}

	let {
		page,
		records = [],
		captions = {},
		currentFilters = [],
		onaction,
		onrowclick,
		onsave,
		ondelete,
		onfilter
	}: Props = $props();

	// Column customization type
	interface ColumnCustomization {
		visible: boolean;
		order?: number;
	}

	// Customization state
	let customizeModalOpen = $state(false);
	let columnCustomizations = $state<Record<string, ColumnCustomization>>({});

	// Filter pane state
	let filterPaneOpen = $state(false);

	// Load customizations from localStorage on mount
	$effect(() => {
		const key = `page-customization-${page.page.id}`;
		const stored = localStorage.getItem(key);
		if (stored) {
			try {
				columnCustomizations = JSON.parse(stored);
			} catch (e) {
				console.error('Failed to load page customizations:', e);
			}
		}
	});

	// Track selected row index
	let selectedIndex = $state(-1);

	// Track table body element for scrolling
	let tableBodyElement: HTMLElement | null = null;

	// Auto-select first row when records load
	$effect(() => {
		if (records.length > 0 && selectedIndex === -1 && editingIndex === null) {
			selectedIndex = 0;
		}
	});

	// Auto-scroll selected row into view
	$effect(() => {
		if (selectedIndex >= 0 && tableBodyElement) {
			const rows = tableBodyElement.querySelectorAll('tr');
			const selectedRow = rows[selectedIndex];
			if (selectedRow) {
				selectedRow.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
			}
		}
	});

	// Track editing state
	let editingIndex = $state<number | null>(null);
	let editingRecord = $state<Record<string, any>>({});
	let isNewRecord = $state(false);

	// Modal card state
	let modalOpen = $state(false);
	let modalCardPage = $state<PageDefinition | null>(null);
	let modalRecord = $state<Record<string, any>>({});
	let modalCaptions = $state<Record<string, string>>({});

	// Get selected record
	const selectedRecord = $derived(
		selectedIndex >= 0 && selectedIndex < records.length ? records[selectedIndex] : null
	);

	// Handle action clicks
	function handleAction(actionName: string) {
		// Handle built-in actions for editable lists
		if (page.page.editable) {
			switch (actionName) {
				case 'New':
					handleNew();
					return;
				case 'Edit':
					handleEdit();
					return;
				case 'Delete':
					handleDelete();
					return;
			}
		}

		onaction?.(actionName, selectedRecord || undefined);
	}

	// Handle new record
	function handleNew() {
		editingRecord = {};
		editingIndex = records.length; // Add at the end
		isNewRecord = true;
	}

	// Handle edit record
	function handleEdit() {
		if (selectedRecord) {
			editingRecord = { ...selectedRecord };
			editingIndex = selectedIndex;
			isNewRecord = false;
		} else {
			alert('Please select a record first by clicking on it in the list.');
		}
	}

	// Handle delete record
	async function handleDelete() {
		if (selectedRecord && confirm('Delete this record?')) {
			await ondelete?.(selectedRecord);
		}
	}

	// Handle save record
	async function handleSave() {
		await onsave?.(editingRecord, isNewRecord);
		editingIndex = null;
		editingRecord = {};
		isNewRecord = false;
	}

	// Handle cancel editing
	function handleCancel() {
		editingIndex = null;
		editingRecord = {};
		isNewRecord = false;
	}

	// Handle field change
	function handleFieldChange(fieldSource: string, value: any) {
		editingRecord[fieldSource] = value;
	}

	// Handle row click
	async function handleRowClick(index: number) {
		selectedIndex = index;

		if (page.page.card_page_id) {
			if (page.page.modal_card) {
				// Open as modal
				await openModalCard(records[index]);
			} else {
				// Navigate to full page
				onrowclick?.(records[index]);
			}
		}
	}

	// Open modal card
	async function openModalCard(record: Record<string, any>) {
		try {
			// Fetch the card page definition
			const response = await fetch(`/api/pages/${page.page.card_page_id}`);
			if (!response.ok) {
				throw new Error(`Failed to load card page: ${response.statusText}`);
			}

			const result = await response.json();
			if (!result.success) {
				throw new Error(result.error || 'Failed to load card page');
			}

			modalCardPage = result.data;
			modalCaptions = result.captions?.fields || {};

			// Load the record data
			const recordId = record['no'] || record['code'] || record['id'];
			if (recordId) {
				modalRecord = await api.getRecord(page.page.source_table, recordId);
			} else {
				modalRecord = { ...record };
			}

			modalOpen = true;
		} catch (err) {
			console.error('Error opening modal card:', err);
			alert('Failed to open card');
		}
	}

	// Close modal
	function closeModal() {
		modalOpen = false;
		modalCardPage = null;
		modalRecord = {};
		modalCaptions = {};
	}

	// Handle save from modal
	async function handleModalSave(savedRecord: Record<string, any>) {
		if (!modalCardPage) return;

		try {
			const recordId = savedRecord['no'] || savedRecord['code'] || savedRecord['id'];
			await api.modifyRecord(page.page.source_table, recordId, savedRecord);

			// Update the record in the list without full refresh
			const index = records.findIndex(r =>
				(r['no'] && r['no'] === recordId) ||
				(r['code'] && r['code'] === recordId) ||
				(r['id'] && r['id'] === recordId)
			);
			if (index !== -1) {
				records[index] = { ...savedRecord };
			}

			// Don't close modal - keep it open like Business Central
		} catch (err) {
			console.error('Error saving modal record:', err);
			alert('Failed to save record');
		}
	}

	// Handle row double-click
	function handleRowDoubleClick(index: number) {
		if (page.page.card_page_id) {
			onrowclick?.(records[index]);
		}
	}

	// Build keyboard shortcut map from actions
	const shortcutMap = $derived(() => {
		const map: Record<string, () => void> = {};

		page.page.actions?.forEach((action) => {
			if (action.shortcut && action.enabled !== false) {
				map[action.shortcut] = () => handleAction(action.name);
			}
		});

		// Add navigation shortcuts
		map['ArrowDown'] = moveDown;
		map['ArrowUp'] = moveUp;
		map['Home'] = moveFirst;
		map['End'] = moveLast;
		map['Enter'] = openCard;

		return map;
	});

	// Navigation functions
	function moveDown() {
		if (selectedIndex < records.length - 1) {
			selectedIndex++;
		}
	}

	function moveUp() {
		if (selectedIndex > 0) {
			selectedIndex--;
		}
	}

	function moveFirst() {
		if (records.length > 0) {
			selectedIndex = 0;
		}
	}

	function moveLast() {
		if (records.length > 0) {
			selectedIndex = records.length - 1;
		}
	}

	async function openCard() {
		if (selectedRecord && page.page.card_page_id) {
			if (page.page.modal_card) {
				// Open as modal
				await openModalCard(selectedRecord);
			} else {
				// Navigate to full page
				onrowclick?.(selectedRecord);
			}
		}
	}

	// Get field caption
	function getFieldCaption(fieldSource: string, fieldCaption?: string): string {
		return captions[fieldSource] || fieldCaption || fieldSource;
	}

	// Format cell value
	function formatValue(val: any): string {
		if (val === null || val === undefined) {
			return '';
		}
		if (typeof val === 'boolean') {
			return val ? 'Yes' : 'No';
		}
		return String(val);
	}

	// Get field style classes
	function getFieldStyle(field: any) {
		const classes: string[] = [];

		switch (field.style) {
			case 'Strong':
				classes.push('font-bold text-nav-blue dark:text-blue-400');
				break;
			case 'Attention':
				classes.push('font-medium text-orange-600 dark:text-orange-400');
				break;
			case 'Favorable':
				classes.push('text-green-600 dark:text-green-400');
				break;
			case 'Unfavorable':
				classes.push('text-red-600 dark:text-red-400');
				break;
		}

		return classes.join(' ');
	}

	// Check if column should be visible based on customizations
	function isColumnVisible(field: Field): boolean {
		// If user has customized this column, use that preference
		if (field.source in columnCustomizations) {
			return columnCustomizations[field.source].visible;
		}
		// Otherwise use the field's visible property (default true)
		return field.visible !== false;
	}

	// Get visible columns (for rendering) with custom order applied
	const visibleColumns = $derived(() => {
		const fields = (page.page.layout.repeater?.fields || [])
			.map((field, index) => ({ field, index }))
			.filter(item => isColumnVisible(item.field));

		// Sort by custom order if available
		return fields
			.sort((a, b) => {
				const orderA = columnCustomizations[a.field.source]?.order ?? a.index;
				const orderB = columnCustomizations[b.field.source]?.order ?? b.index;
				return orderA - orderB;
			})
			.map(item => item.field);
	});

	// Open customize modal
	function handleCustomize() {
		customizeModalOpen = true;
	}

	// Save customizations
	function handleSaveCustomizations(customizations: Record<string, ColumnCustomization>) {
		columnCustomizations = customizations;
		const key = `page-customization-${page.page.id}`;
		localStorage.setItem(key, JSON.stringify(customizations));
	}

	// Toggle filter pane
	function handleToggleFilters() {
		filterPaneOpen = !filterPaneOpen;
	}

	// Apply filters
	function handleApplyFilters(filters: TableFilter[]) {
		onfilter?.(filters);
	}

	// Close filter pane
	function handleCloseFilterPane() {
		filterPaneOpen = false;
	}
</script>

<div class="list-page" use:shortcuts={shortcutMap()} tabindex="0" autofocus>
	<PageHeader title={page.page.caption}>
		<svelte:fragment slot="actions">
			{#if editingIndex !== null}
				<!-- Show Save/Cancel when editing -->
				<Button variant="primary" size="sm" onclick={handleSave}>
					Save
				</Button>
				<Button variant="secondary" size="sm" onclick={handleCancel}>
					Cancel
				</Button>
			{:else}
				<!-- Show normal actions when not editing -->
				<!-- Filter button -->
				<Button
					variant={filterPaneOpen ? 'primary' : 'secondary'}
					size="sm"
					onclick={handleToggleFilters}
					title="Toggle filter pane"
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
							d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2.586a1 1 0 01-.293.707l-6.414 6.414a1 1 0 00-.293.707V17l-4 4v-6.586a1 1 0 00-.293-.707L3.293 7.293A1 1 0 013 6.586V4z"
						/>
					</svg>
					<span class="ml-1">Filter</span>
					{#if currentFilters.length > 0}
						<span class="ml-1 px-1.5 py-0.5 text-xs bg-blue-600 text-white rounded-full">
							{currentFilters.length}
						</span>
					{/if}
				</Button>

				<!-- Customize button -->
				<Button variant="secondary" size="sm" onclick={handleCustomize} title="Customize columns">
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
							d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4"
						/>
					</svg>
					<span class="ml-1">Customize</span>
				</Button>

				{#each page.page.actions?.filter((a) => a.promoted) || [] as action}
					{@const isDisabled = action.enabled === false || (action.name !== 'New' && action.name !== 'Refresh' && !selectedRecord)}
					<Button
						variant={action.name === 'Delete' ? 'danger' : 'secondary'}
						size="sm"
						onclick={() => handleAction(action.name)}
						disabled={isDisabled}
					>
						{action.caption}
						{#if action.shortcut}
							<span class="ml-2 text-xs opacity-70">{action.shortcut}</span>
						{/if}
					</Button>
				{/each}
			{/if}
		</svelte:fragment>
	</PageHeader>

	<div class="list-content">
		{#if filterPaneOpen}
			<FilterPane
				{page}
				{captions}
				currentFilters={currentFilters}
				onApply={handleApplyFilters}
				onClose={handleCloseFilterPane}
			/>
		{/if}

		<div class="table-container">
		<table class="table">
			<thead>
				<tr>
					{#each visibleColumns() as field}
						<th style={field.width ? `width: ${field.width}px` : ''}>
							{getFieldCaption(field.source, field.caption)}
						</th>
					{/each}
				</tr>
			</thead>
			<tbody bind:this={tableBodyElement}>
				{#each records as record, index}
					<tr
						class={cn('cursor-pointer', selectedIndex === index && 'selected')}
						onclick={() => handleRowClick(index)}
						ondblclick={() => handleRowDoubleClick(index)}
					>
						{#each visibleColumns() as field}
							<td class={getFieldStyle(field)}>
								{#if editingIndex === index}
									<!-- Editable cell -->
									<input
										type="text"
										class="w-full px-2 py-1 border border-gray-300 rounded dark:bg-gray-700 dark:border-gray-600 dark:text-gray-100"
										value={editingRecord[field.source] ?? ''}
										oninput={(e) => handleFieldChange(field.source, e.currentTarget.value)}
									/>
								{:else}
									<!-- Read-only cell -->
									{formatValue(record[field.source])}
								{/if}
							</td>
						{/each}
					</tr>
				{/each}

				<!-- New record row if adding -->
				{#if isNewRecord && editingIndex === records.length}
					<tr class="bg-blue-50 dark:bg-blue-900/20">
						{#each visibleColumns() as field}
							<td>
								<input
									type="text"
									class="w-full px-2 py-1 border border-gray-300 rounded dark:bg-gray-700 dark:border-gray-600 dark:text-gray-100"
									value={editingRecord[field.source] ?? ''}
									oninput={(e) => handleFieldChange(field.source, e.currentTarget.value)}
									placeholder={getFieldCaption(field.source, field.caption)}
								/>
							</td>
						{/each}
					</tr>
				{/if}

				{#if records.length === 0 && !isNewRecord}
					<tr>
						<td colspan={page.page.layout.repeater?.fields?.length || 1} class="text-center py-8">
							<span class="text-gray-500 dark:text-gray-400">No records found</span>
						</td>
					</tr>
				{/if}
			</tbody>
		</table>
		</div>

		<div class="status-bar">
			<span class="text-sm text-gray-600 dark:text-gray-400">
				{records.length} record{records.length !== 1 ? 's' : ''}
				{#if selectedRecord}
					• Row {selectedIndex + 1} selected
				{/if}
			</span>
		</div>
	</div>
</div>

<!-- Modal Card -->
{#if modalOpen && modalCardPage}
	<ModalCardPage
		open={modalOpen}
		page={modalCardPage}
		bind:record={modalRecord}
		captions={modalCaptions}
		onclose={closeModal}
		onsave={handleModalSave}
	/>
{/if}

<!-- Customize Columns Modal -->
{#if customizeModalOpen}
	<CustomizeListPageModal
		open={customizeModalOpen}
		{page}
		customizations={columnCustomizations}
		onclose={() => customizeModalOpen = false}
		onsave={handleSaveCustomizations}
	/>
{/if}

<style>
	.list-page {
		@apply flex flex-col gap-4 h-full;
	}

	.list-content {
		@apply flex flex-1 gap-4 min-h-0;
	}

	.table-container {
		@apply flex-1 overflow-auto border border-gray-200 rounded-lg;
		@apply dark:border-gray-700;
	}

	.table {
		@apply w-full border-collapse;
	}

	.table thead {
		@apply sticky top-0 bg-nav-blue text-white z-10;
		@apply dark:bg-gray-800;
	}

	.table th {
		@apply px-4 py-3 text-left text-sm font-semibold;
		border-right: 1px solid rgba(255, 255, 255, 0.1);
	}

	.table th:last-child {
		border-right: none;
	}

	.table tbody tr {
		@apply border-b border-gray-200 hover:bg-blue-50 transition-colors;
		@apply dark:border-gray-700 dark:hover:bg-gray-700;
	}

	.table tbody tr.selected {
		@apply bg-blue-100 hover:bg-blue-100;
		@apply dark:bg-blue-900/30 dark:hover:bg-blue-900/30;
	}

	.table td {
		@apply px-4 py-2 text-sm;
	}

	.status-bar {
		@apply px-4 py-2 bg-gray-50 border-t border-gray-200 rounded-b;
		@apply dark:bg-gray-800 dark:border-gray-700 dark:text-gray-300;
	}
</style>
