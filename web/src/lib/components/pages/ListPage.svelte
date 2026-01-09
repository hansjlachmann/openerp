<script lang="ts">
	import type { PageDefinition, Field } from '$lib/types/pages';
	import type { TableFilter } from '$lib/types/api';
	import Button from '$lib/components/Button.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import ModalCardPage from './ModalCardPage.svelte';
	import CustomizeListPageModal from './CustomizeListPageModal.svelte';
	import FilterPane from './FilterPane.svelte';
	import PlusIcon from '$lib/components/icons/PlusIcon.svelte';
	import EditIcon from '$lib/components/icons/EditIcon.svelte';
	import TrashIcon from '$lib/components/icons/TrashIcon.svelte';
	import RefreshIcon from '$lib/components/icons/RefreshIcon.svelte';
	import { shortcuts } from '$lib/utils/shortcuts';
	import { cn } from '$lib/utils/cn';
	import { api } from '$lib/services/api';
	import { currentUser } from '$lib/stores/user';
	import { getFieldCaption, getFieldStyleClasses, formatValue } from '$lib/utils/fieldHelpers';
	import { loadPageCustomizations, savePageCustomizations } from '$lib/utils/customizationStorage';

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

	// Edit mode state
	let editMode = $state(false);
	// Editable copy of records for edit mode (to avoid mutating props)
	let editableRecords = $state<Array<Record<string, any>>>([]);
	// Prevent rapid toggling
	let isToggling = false;

	// Records to display - switches between editable and read-only
	const displayRecords = $derived(editMode ? editableRecords : records);

	// Track list page element for focus
	let listPageElement: HTMLDivElement | null = null;

	// Auto-focus the page on mount and when records load
	$effect(() => {
		if (listPageElement && !editMode && records.length > 0) {
			setTimeout(() => {
				listPageElement?.focus();
			}, 100);
		}
	});

	// Auto-focus first cell when entering edit mode
	$effect(() => {
		if (editMode && currentCellRow >= 0 && currentCellCol >= 0) {
			focusCell(currentCellRow, currentCellCol);
		}
	});

	// Load customizations from localStorage on mount
	$effect(() => {
		const userId = $currentUser?.user_id || 'anonymous';
		columnCustomizations = loadPageCustomizations<Record<string, ColumnCustomization>>(
			userId,
			page.page.id
		);
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

	// Track editing state (old inline editing - keep for compatibility)
	let editingIndex = $state<number | null>(null);
	let editingRecord = $state<Record<string, any>>({});
	let isNewRecord = $state(false);

	// Edit List mode state (BC-style full list editing)
	let currentCellRow = $state<number>(-1);
	let currentCellCol = $state<number>(-1);

	// Modal card state
	let modalOpen = $state(false);
	let modalCardPage = $state<PageDefinition | null>(null);
	let modalRecord = $state<Record<string, any>>({});
	let modalCaptions = $state<Record<string, string>>({});
	let modalSaving = $state(false);

	// Get selected record
	const selectedRecord = $derived(
		selectedIndex >= 0 && selectedIndex < records.length ? records[selectedIndex] : null
	);

	// Handle action clicks
	function handleAction(actionName: string) {
		// Handle Edit mode toggle
		if (actionName === 'Edit') {
			toggleEditMode();
			return;
		}

		// Handle built-in actions for editable lists
		if (page.page.editable) {
			switch (actionName) {
				case 'New':
					handleNew();
					return;
				case 'Delete':
					handleDelete();
					return;
			}
		}

		// Handle "New" action for modal cards
		if (actionName === 'New' && page.page.modal_card && page.page.card_page_id) {
			openModalCard({});
			return;
		}

		onaction?.(actionName, selectedRecord || undefined);
	}

	// Handle new record
	function handleNew() {
		editingRecord = {};
		editingIndex = records.length; // Add at the end
		isNewRecord = true;
	}

	// Toggle edit mode
	function toggleEditMode() {
		if (isToggling) {
			return;
		}

		isToggling = true;
		editMode = !editMode;

		if (editMode) {
			// Entering edit mode - create editable copies and focus at selected row
			editableRecords = records.map(r => ({ ...r }));
			if (editableRecords.length > 0) {
				// Start at the currently selected row, or first row if none selected
				currentCellRow = selectedIndex >= 0 ? selectedIndex : 0;
				currentCellCol = 0;
			}
		} else {
			// Exiting edit mode - reset state
			currentCellRow = -1;
			currentCellCol = -1;
			editableRecords = [];
		}

		// Reset the toggling flag after a short delay
		setTimeout(() => {
			isToggling = false;
		}, 100);
	}

	// Auto-save when leaving a cell
	async function handleCellBlur(record: Record<string, any>, rowIndex: number) {
		if (!page || !editMode) return;

		try {
			// Check if this is a new record
			const isNew = record._isNew === true;
			const recordId = record['no'] || record['code'] || record['id'];

			if (isNew && !recordId) {
				// New record - only save if user has entered some data
				const hasData = Object.keys(record).some(key => key !== '_isNew' && record[key] !== undefined && record[key] !== '');
				if (hasData) {
					// Remove the temporary flag before saving
					const { _isNew, ...recordToSave } = record;
					const savedRecord = await api.insertRecord(page.page.source_table, recordToSave);
					// Update with the saved record (which now has an ID)
					editableRecords[rowIndex] = savedRecord;
					// Trigger parent update if callback exists
					if (onsave) {
						await onsave(savedRecord, true);
					}
				}
			} else if (recordId) {
				// Existing record - update it
				const { _isNew, ...recordToSave } = record;
				const savedRecord = await api.modifyRecord(page.page.source_table, recordId, recordToSave);
				// Update the editable record with the response
				editableRecords[rowIndex] = savedRecord;
				// Trigger parent update if callback exists
				if (onsave) {
					await onsave(savedRecord, false);
				}
			}
		} catch (err) {
			console.error('Error saving cell:', err);
			// Silently fail - user can see the change didn't save if they refresh
		}
	}

	// Insert a new row at cursor position
	function insertNewRow() {
		if (!editMode) return;

		// Create a new empty record
		const newRecord: Record<string, any> = {};

		// Mark it as new with a temporary flag
		newRecord._isNew = true;

		// Insert at current cursor position (or at the end if no cursor)
		const insertIndex = currentCellRow >= 0 ? currentCellRow : editableRecords.length;
		editableRecords = [
			...editableRecords.slice(0, insertIndex),
			newRecord,
			...editableRecords.slice(insertIndex)
		];

		// Focus the first cell of the new row
		currentCellRow = insertIndex;
		currentCellCol = 0;

		// Focus will happen via the effect
	}

	// Handle keyboard navigation in edit list mode
	function handleCellKeyDown(event: KeyboardEvent, rowIndex: number, colIndex: number) {
		const cols = visibleColumns();
		const totalRows = editMode ? editableRecords.length : records.length;

		// Ctrl+Insert or Ctrl+N to insert new row
		if ((event.key === 'Insert' || event.key === 'n') && event.ctrlKey) {
			event.preventDefault();
			insertNewRow();
			return;
		}

		switch (event.key) {
			case 'ArrowUp':
				event.preventDefault();
				if (rowIndex > 0) {
					currentCellRow = rowIndex - 1;
					currentCellCol = colIndex;
					focusCell(currentCellRow, currentCellCol);
				}
				break;
			case 'ArrowDown':
				event.preventDefault();
				if (rowIndex < totalRows - 1) {
					currentCellRow = rowIndex + 1;
					currentCellCol = colIndex;
					focusCell(currentCellRow, currentCellCol);
				} else {
					// On last row, create new row below
					insertNewRow();
				}
				break;
			case 'ArrowLeft':
				event.preventDefault();
				if (colIndex > 0) {
					currentCellRow = rowIndex;
					currentCellCol = colIndex - 1;
					focusCell(currentCellRow, currentCellCol);
				}
				break;
			case 'ArrowRight':
			case 'Tab':
				event.preventDefault();
				if (colIndex < cols.length - 1) {
					currentCellRow = rowIndex;
					currentCellCol = colIndex + 1;
					focusCell(currentCellRow, currentCellCol);
				}
				break;
			case 'Enter':
				event.preventDefault();
				// Move to next row on Enter
				if (rowIndex < totalRows - 1) {
					currentCellRow = rowIndex + 1;
					currentCellCol = colIndex;
					focusCell(currentCellRow, currentCellCol);
				} else {
					// On last row, create new row below
					insertNewRow();
				}
				break;
		}
	}

	// Focus a specific cell
	function focusCell(rowIndex: number, colIndex: number) {
		setTimeout(() => {
			const input = document.querySelector(
				`input[data-row="${rowIndex}"][data-col="${colIndex}"]`
			) as HTMLInputElement;
			if (input) {
				input.focus();
				input.select();
			}
		}, 0);
	}

	// Handle edit record (only works in edit mode)
	function handleEdit() {
		if (!editMode) {
			alert('Please enable Edit mode first by clicking the Edit button.');
			return;
		}
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

	// Handle row click - just select the row
	function handleRowClick(index: number) {
		console.log('Row clicked:', index);
		selectedIndex = index;
		// Give focus to the page so keyboard shortcuts work
		listPageElement?.focus();
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
		if (!modalCardPage || modalSaving) {
			return; // Prevent concurrent saves
		}

		modalSaving = true;
		try {
			const recordId = savedRecord['no'] || savedRecord['code'] || savedRecord['id'];

			if (recordId) {
				// Update existing record
				const responseData = await api.modifyRecord(page.page.source_table, recordId, savedRecord);

				// Update the record in the list without full refresh
				const index = records.findIndex(r =>
					(r['no'] && r['no'] === recordId) ||
					(r['code'] && r['code'] === recordId) ||
					(r['id'] && r['id'] === recordId)
				);
				if (index !== -1) {
					records[index] = responseData;
				}
				// Update modal record with the response data
				modalRecord = responseData;
			} else {
				// Insert new record
				const responseData = await api.insertRecord(page.page.source_table, savedRecord);
				// Add the new record to the list
				records = [...records, responseData];
				// Update modal record with the saved data (including generated ID)
				modalRecord = responseData;
			}

			// Don't close modal - keep it open like Business Central
		} catch (err) {
			console.error('Error saving modal record:', err);
			alert('Failed to save record');
		} finally {
			modalSaving = false;
		}
	}

	// Handle actions from modal card
	async function handleModalAction(actionName: string) {
		if (!modalCardPage) return;

		switch (actionName) {
			case 'Delete':
				const recordId = modalRecord['no'] || modalRecord['code'] || modalRecord['id'];
				if (recordId && confirm(`Delete this ${modalCardPage.page.caption}?`)) {
					try {
						await api.deleteRecord(page.page.source_table, recordId);

						// Remove the record from the list
						records = records.filter(r => {
							const id = r['no'] || r['code'] || r['id'];
							return id !== recordId;
						});

						// Close the modal
						closeModal();

						alert('Record deleted successfully');
					} catch (err) {
						console.error('Delete error:', err);
						alert('Failed to delete record');
					}
				}
				break;
			case 'Refresh':
				// Reload the modal record
				const id = modalRecord['no'] || modalRecord['code'] || modalRecord['id'];
				if (id) {
					try {
						modalRecord = await api.getRecord(page.page.source_table, id);
					} catch (err) {
						console.error('Refresh error:', err);
					}
				}
				break;
		}
	}

	// Handle primary key click - open the card
	async function handlePrimaryKeyClick(index: number) {
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

	// Build keyboard shortcut map from actions
	const shortcutMap = $derived(() => {
		const map: Record<string, () => void> = {};

		page.page.actions?.forEach((action) => {
			if (action.shortcut && action.enabled !== false) {
				map[action.shortcut] = () => handleAction(action.name);
			}
		});

		// Add navigation shortcuts only when NOT in edit mode
		if (!editMode) {
			map['ArrowDown'] = moveDown;
			map['ArrowUp'] = moveUp;
			map['Home'] = moveFirst;
			map['End'] = moveLast;
			map['Enter'] = openCard;
		}

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
		const userId = $currentUser?.user_id || 'anonymous';
		savePageCustomizations(userId, page.page.id, customizations);
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

<div class="list-page" use:shortcuts={shortcutMap()} tabindex="0" bind:this={listPageElement}>
	<PageHeader title={page.page.caption}>
		<svelte:fragment slot="leftActions">
			{#if editingIndex !== null}
				<!-- Show Save/Cancel when editing -->
				<Button variant="primary" size="sm" onclick={handleSave}>
					Save
				</Button>
				<Button variant="secondary" size="sm" onclick={handleCancel}>
					Cancel
				</Button>
			{:else}
				{#each page.page.actions?.filter((a) => a.promoted) || [] as action}
					{@const isDisabled = action.enabled === false || (action.name !== 'New' && action.name !== 'Edit' && action.name !== 'Refresh' && !selectedRecord)}
					{@const variant = action.name === 'Delete' ? 'danger' : action.name === 'New' ? 'success' : 'secondary'}
					<Button
						variant={variant}
						size="sm"
						onclick={() => handleAction(action.name)}
						disabled={isDisabled}
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
			{/if}
		</svelte:fragment>

		<svelte:fragment slot="rightActions">
			{#if editingIndex === null}
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
							{getFieldCaption(field.source, captions, field.caption)}
						</th>
					{/each}
				</tr>
			</thead>
			<tbody bind:this={tableBodyElement}>
				{#each displayRecords as record, index (record.code || record.no || record.id || index)}
					<tr
						class={cn(
							editMode ? '' : 'cursor-pointer',
							selectedIndex === index && 'selected',
							record._isNew && 'new-row'
						)}
						onclick={() => !editMode && handleRowClick(index)}
					>
						{#each visibleColumns() as field, colIndex}
							<td class="p-0 border-r border-b border-gray-300 dark:border-gray-600">
								{#if editMode}
									<!-- Edit Mode - Editable inputs -->
									{#if typeof record[field.source] === 'boolean'}
										<div class="edit-cell-input flex items-center">
											<input
												type="checkbox"
												data-row={index}
												data-col={colIndex}
												bind:checked={record[field.source]}
												onchange={async () => {
													await handleCellBlur(record, index);
												}}
												onkeydown={(e) => handleCellKeyDown(e, index, colIndex)}
											/>
										</div>
									{:else}
										<input
											type="text"
											data-row={index}
											data-col={colIndex}
											class="edit-cell-input"
											bind:value={record[field.source]}
											onblur={async () => {
												await handleCellBlur(record, index);
											}}
											onkeydown={(e) => handleCellKeyDown(e, index, colIndex)}
										/>
									{/if}
								{:else}
									<!-- Normal Mode - Read-only -->
									<div class={cn('read-cell-content', getFieldStyleClasses(field))}>
										{#if typeof record[field.source] === 'boolean'}
											<input type="checkbox" checked={record[field.source]} disabled class="cursor-not-allowed" />
										{:else if field.primary_key && page.page.card_page_id}
											<button
												type="button"
												class="primary-key-link"
												onclick={(e) => {
													e.stopPropagation();
													handlePrimaryKeyClick(index);
												}}
											>
												{formatValue(record[field.source])}
											</button>
										{:else}
											{formatValue(record[field.source])}
										{/if}
									</div>
								{/if}
							</td>
						{/each}
					</tr>
				{/each}
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
		onaction={handleModalAction}
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

	.table tbody tr.new-row {
		background-color: #e0f2fe !important;
	}

	:global(.dark) .table tbody tr.new-row {
		background-color: rgba(56, 189, 248, 0.15) !important;
	}

	.table td {
		padding: 2px 6px;
		font-size: 0.875rem;
		line-height: 1.3;
		vertical-align: bottom;
	}

	.status-bar {
		@apply px-4 py-2 bg-gray-50 border-t border-gray-200 rounded-b;
		@apply dark:bg-gray-800 dark:border-gray-700 dark:text-gray-300;
	}

	.edit-cell-input {
		display: block !important;
		width: 100%;
		height: auto !important;
		min-height: 0 !important;
		padding: 2px 6px;
		line-height: 1.3;
		font-size: 0.875rem;
		background: transparent !important;
		border: 0 !important;
		outline: 0 !important;
		box-shadow: none !important;
		-webkit-appearance: none !important;
		-moz-appearance: none !important;
		appearance: none !important;
		margin: 0 !important;
	}

	.edit-cell-input:focus {
		outline: 0 !important;
		box-shadow: none !important;
		border: 0 !important;
		background: transparent !important;
	}

	:global(.dark) .edit-cell-input {
		background: transparent !important;
		color: white;
	}

	:global(.dark) .edit-cell-input:focus {
		background: transparent !important;
	}

	/* Set background on the td cells in edit mode and normal mode */
	tbody tr:not(.selected) td.p-0 {
		background: white;
	}

	:global(.dark) tbody tr:not(.selected) td.p-0 {
		background: rgb(31 41 55);
	}

	/* Selected rows - make td background transparent to show row highlight */
	tbody tr.selected td.p-0 {
		background: transparent;
	}

	/* Normal mode cell content - match edit mode input exactly */
	.read-cell-content {
		display: block;
		width: 100%;
		height: auto;
		padding: 2px 6px;
		line-height: 1.3;
		font-size: 0.875rem;
		margin: 0;
	}

	/* Primary key link - looks like a hyperlink */
	.primary-key-link {
		color: #2563eb;
		text-decoration: underline;
		background: none;
		border: none;
		padding: 0;
		font: inherit;
		cursor: pointer;
		text-align: inherit;
	}

	.primary-key-link:hover {
		color: #1d4ed8;
	}

	:global(.dark) .primary-key-link {
		color: #60a5fa;
	}

	:global(.dark) .primary-key-link:hover {
		color: #93c5fd;
	}
</style>
