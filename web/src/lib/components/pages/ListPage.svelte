<script lang="ts">
	import type { PageDefinition } from '$lib/types/pages';
	import Button from '$lib/components/Button.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import ModalCardPage from './ModalCardPage.svelte';
	import { shortcuts } from '$lib/utils/shortcuts';
	import { cn } from '$lib/utils/cn';
	import { api } from '$lib/services/api';

	interface Props {
		page: PageDefinition;
		records?: Array<Record<string, any>>;
		captions?: Record<string, string>;
		onaction?: (actionName: string, record?: Record<string, any>) => void;
		onrowclick?: (record: Record<string, any>) => void;
		onsave?: (record: Record<string, any>, isNew: boolean) => Promise<void>;
		ondelete?: (record: Record<string, any>) => Promise<void>;
	}

	let { page, records = [], captions = {}, onaction, onrowclick, onsave, ondelete }: Props = $props();

	// Track selected row index
	let selectedIndex = $state(-1);

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
		console.log('🔷 Row clicked, index:', index);
		console.log('🔷 card_page_id:', page.page.card_page_id, 'modal_card:', page.page.modal_card);

		selectedIndex = index;

		if (page.page.card_page_id) {
			if (page.page.modal_card) {
				console.log('🟢 Opening as modal');
				// Open as modal
				await openModalCard(records[index]);
			} else {
				console.log('🟡 Navigating to full page');
				// Navigate to full page
				onrowclick?.(records[index]);
			}
		} else {
			console.log('⚠️ No card_page_id configured');
		}
	}

	// Open modal card
	async function openModalCard(record: Record<string, any>) {
		console.log('🔵 openModalCard called with record:', record);
		try {
			// Fetch the card page definition
			console.log('📡 Fetching card page definition:', page.page.card_page_id);
			const response = await fetch(`/api/pages/${page.page.card_page_id}`);
			if (!response.ok) {
				throw new Error(`Failed to load card page: ${response.statusText}`);
			}

			const result = await response.json();
			console.log('📦 Card page result:', result);
			if (!result.success) {
				throw new Error(result.error || 'Failed to load card page');
			}

			modalCardPage = result.data;
			modalCaptions = result.captions?.fields || {};

			// Load the record data
			const recordId = record['no'] || record['code'] || record['id'];
			console.log('🔑 Loading record with ID:', recordId);
			if (recordId) {
				modalRecord = await api.getRecord(page.page.source_table, recordId);
			} else {
				modalRecord = { ...record };
			}

			console.log('✅ Opening modal with record:', modalRecord);
			modalOpen = true;
		} catch (err) {
			console.error('❌ Error opening modal card:', err);
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

			// Refresh the list
			if (onsave) {
				await onsave(savedRecord, false);
			}

			closeModal();
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

	function openCard() {
		if (selectedRecord && page.page.card_page_id) {
			onrowclick?.(selectedRecord);
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
</script>

<div class="list-page" use:shortcuts={shortcutMap()}>
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

	<div class="table-container">
		<table class="table">
			<thead>
				<tr>
					{#each page.page.layout.repeater?.fields || [] as field}
						<th style={field.width ? `width: ${field.width}px` : ''}>
							{getFieldCaption(field.source, field.caption)}
						</th>
					{/each}
				</tr>
			</thead>
			<tbody>
				{#each records as record, index}
					<tr
						class={cn('cursor-pointer', selectedIndex === index && 'selected')}
						onclick={() => handleRowClick(index)}
						ondblclick={() => handleRowDoubleClick(index)}
					>
						{#each page.page.layout.repeater?.fields || [] as field}
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
						{#each page.page.layout.repeater?.fields || [] as field}
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

<style>
	.list-page {
		@apply flex flex-col gap-4 h-full;
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
