<script lang="ts">
	import { onMount } from 'svelte';
	import type { PageDefinition } from '$lib/types/pages';
	import { fetchPage } from '$lib/services/pages';
	import { api } from '$lib/services/api';
	import CardPage from './CardPage.svelte';
	import ListPage from './ListPage.svelte';

	interface Props {
		pageid: number;
		recordid?: string;
	}

	let { pageid, recordid }: Props = $props();

	// State
	let page: PageDefinition | null = $state(null);
	let captions: Record<string, string> = $state({});
	let loading = $state(true);
	let error = $state<string | null>(null);

	// Data for the page
	let record: Record<string, any> = $state({});
	let records: Array<Record<string, any>> = $state([]);

	// Filters for list pages
	let currentFilters: import('$lib/types/api').TableFilter[] = $state([]);

	// Navigation data for card pages
	let recordIds: string[] = $state([]);
	let currentRecordIndex = $state(-1);

	// Load page definition and data
	onMount(async () => {
		try {
			loading = true;
			error = null;

			// Fetch page definition
			const response = await fetch(`/api/pages/${pageid}`);
			if (!response.ok) {
				throw new Error(`Failed to load page: ${response.statusText}`);
			}

			const result = await response.json();
			if (!result.success) {
				throw new Error(result.error || 'Failed to load page');
			}

			page = result.data;
			captions = result.captions?.fields || {};

			// Load data based on page type
			if (page.page.type === 'Card') {
				await loadCardData();
			} else if (page.page.type === 'List') {
				await loadListData();
			}
		} catch (err) {
			error = err instanceof Error ? err.message : 'Unknown error';
			console.error('Error loading page:', err);
		} finally {
			loading = false;
		}
	});

	// Load data for card page
	async function loadCardData() {
		if (!page) return;

		try {
			if (recordid) {
				// Load specific record
				record = await api.getRecord(page.page.source_table, recordid);
			} else {
				// New record
				record = {};
			}

			// Load record IDs for navigation if enabled
			if (page.page.enable_navigation && recordid) {
				// Use lightweight IDs-only endpoint
				recordIds = await api.getRecordIDs(page.page.source_table);

				// Find current record index
				currentRecordIndex = recordIds.indexOf(recordid);
			}
		} catch (err) {
			console.error('Error loading card data:', err);
			record = {};
		}
	}

	// Load data for list page
	async function loadListData() {
		if (!page) return;

		try {
			// Determine which fields are visible based on customizations
			const visibleFields = getVisibleFields();

			// Load records with only visible fields to avoid expensive FlowField calculations
			// Also apply current filters
			const options: import('$lib/types/api').ListOptions = {};
			if (visibleFields.length > 0) {
				options.fields = visibleFields;
			}
			if (currentFilters.length > 0) {
				options.filters = currentFilters;
			}

			const response = await api.listRecords(page.page.source_table, options);
			records = response.records || [];
		} catch (err) {
			console.error('Error loading list data:', err);
			records = [];
		}
	}

	// Get visible fields from page definition and user customizations
	function getVisibleFields(): string[] {
		if (!page || !page.page.layout.repeater?.fields) return [];

		// Load user customizations from localStorage
		const key = `page-customization-${page.page.id}`;
		const stored = localStorage.getItem(key);
		let customizations: Record<string, { visible: boolean }> = {};

		if (stored) {
			try {
				customizations = JSON.parse(stored);
			} catch (e) {
				console.error('Failed to load page customizations:', e);
			}
		}

		// Filter to visible fields only
		return page.page.layout.repeater.fields
			.filter(field => {
				// Check if user has customized this field
				if (field.source in customizations) {
					return customizations[field.source].visible;
				}
				// Otherwise use the field's visible property (default true)
				return field.visible !== false;
			})
			.map(field => field.source);
	}

	// Handle actions from card page
	async function handleCardAction(actionName: string) {
		if (!page) return;

		switch (actionName) {
			case 'New':
				record = {};
				break;
			case 'Delete':
				if (recordid) {
					try {
						await api.deleteRecord(page.page.source_table, recordid);
						// Navigate back or show message
						alert('Record deleted');
					} catch (err) {
						alert('Failed to delete record');
					}
				}
				break;
			case 'Refresh':
				await loadCardData();
				break;
		}
	}

	// Handle save from card page
	async function handleCardSave(savedRecord: Record<string, any>) {
		if (!page) return;

		try {
			if (recordid) {
				// Update existing record
				await api.modifyRecord(page.page.source_table, recordid, savedRecord);
				alert('Record updated');
			} else {
				// Insert new record
				const response = await api.insertRecord(page.page.source_table, savedRecord);
				if (response.success) {
					alert('Record created');
					// Could navigate to the new record
				}
			}
		} catch (err) {
			alert('Failed to save record');
			console.error('Save error:', err);
		}
	}

	// Handle actions from list page
	async function handleListAction(actionName: string, selectedRecord?: Record<string, any>) {
		if (!page) return;

		switch (actionName) {
			case 'New':
				// Navigate to card page in new mode
				if (page.page.card_page_id) {
					window.location.href = `/pages/${page.page.card_page_id}`;
				}
				break;
			case 'Edit':
				if (selectedRecord && page.page.card_page_id) {
					// Navigate to card page with record ID
					const recordId = selectedRecord['no'] || selectedRecord['code'] || selectedRecord['id'];
					window.location.href = `/pages/${page.page.card_page_id}/${recordId}`;
				}
				break;
			case 'Delete':
				if (selectedRecord) {
					const recordId = selectedRecord['no'] || selectedRecord['code'] || selectedRecord['id'];
					if (confirm('Delete this record?')) {
						try {
							await api.deleteRecord(page.page.source_table, recordId);
							await loadListData();
							alert('Record deleted');
						} catch (err) {
							alert('Failed to delete record');
						}
					}
				}
				break;
			case 'Refresh':
				await loadListData();
				break;
		}
	}

	// Handle row click in list page
	function handleRowClick(clickedRecord: Record<string, any>) {
		if (!page || !page.page.card_page_id) return;

		const recordId = clickedRecord['no'] || clickedRecord['code'] || clickedRecord['id'];
		window.location.href = `/pages/${page.page.card_page_id}/${recordId}`;
	}

	// Handle save from list page (inline editing)
	async function handleListSave(savedRecord: Record<string, any>, isNew: boolean) {
		if (!page) return;

		try {
			if (isNew) {
				// Insert new record
				await api.insertRecord(page.page.source_table, savedRecord);
				await loadListData();
			} else {
				// Update existing record
				const recordId = savedRecord['no'] || savedRecord['code'] || savedRecord['id'];
				await api.modifyRecord(page.page.source_table, recordId, savedRecord);
				await loadListData();
			}
		} catch (err) {
			alert('Failed to save record');
			console.error('Save error:', err);
			throw err;
		}
	}

	// Handle delete from list page
	async function handleListDelete(deletedRecord: Record<string, any>) {
		if (!page) return;

		try {
			const recordId = deletedRecord['no'] || deletedRecord['code'] || deletedRecord['id'];
			await api.deleteRecord(page.page.source_table, recordId);
			await loadListData();
		} catch (err) {
			alert('Failed to delete record');
			console.error('Delete error:', err);
			throw err;
		}
	}

	// Handle filter change from list page
	async function handleFilterChange(filters: import('$lib/types/api').TableFilter[]) {
		currentFilters = filters;
		await loadListData();
	}

	// Navigation functions for card pages
	function navigateToRecord(targetRecordId: string) {
		if (!page) return;
		window.location.href = `/pages/${page.page.id}/${targetRecordId}`;
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

{#if loading}
	<div class="flex items-center justify-center h-full">
		<div class="text-center">
			<div class="animate-spin rounded-full h-12 w-12 border-b-2 border-nav-blue dark:border-blue-400 mx-auto mb-4"></div>
			<p class="text-gray-600 dark:text-gray-400">Loading page...</p>
		</div>
	</div>
{:else if error}
	<div class="flex items-center justify-center h-full">
		<div class="text-center">
			<div class="text-red-600 dark:text-red-400 text-5xl mb-4">⚠</div>
			<h2 class="text-xl font-semibold text-gray-800 dark:text-gray-200 mb-2">Error Loading Page</h2>
			<p class="text-gray-600 dark:text-gray-400">{error}</p>
		</div>
	</div>
{:else if page}
	{#if page.page.type === 'Card'}
		<CardPage
			{page}
			bind:record
			{captions}
			onaction={handleCardAction}
			onsave={handleCardSave}
			navigationEnabled={page.page.enable_navigation || false}
			canNavigateFirst={currentRecordIndex > 0}
			canNavigatePrevious={currentRecordIndex > 0}
			canNavigateNext={currentRecordIndex >= 0 && currentRecordIndex < recordIds.length - 1}
			canNavigateLast={currentRecordIndex >= 0 && currentRecordIndex < recordIds.length - 1}
			onNavigateFirst={navigateFirst}
			onNavigatePrevious={navigatePrevious}
			onNavigateNext={navigateNext}
			onNavigateLast={navigateLast}
		/>
	{:else if page.page.type === 'List'}
		<ListPage
			{page}
			{records}
			{captions}
			{currentFilters}
			onaction={handleListAction}
			onrowclick={handleRowClick}
			onsave={handleListSave}
			ondelete={handleListDelete}
			onfilter={handleFilterChange}
		/>
	{:else}
		<div class="flex items-center justify-center h-full">
			<div class="text-center">
				<p class="text-gray-600 dark:text-gray-400">Page type "{page.page.type}" not yet supported</p>
			</div>
		</div>
	{/if}
{/if}
