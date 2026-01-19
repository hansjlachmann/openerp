<script lang="ts">
	import { onMount } from 'svelte';
	import type { PageDefinition } from '$lib/types/pages';
	import { fetchPage } from '$lib/services/pages';
	import { api } from '$lib/services/api';
	import { currentUser } from '$lib/stores/user';
	import { toast } from '$lib/stores/toast';
	import { confirm } from '$lib/stores/confirm';
	import { breadcrumb } from '$lib/stores/breadcrumb';
	import CardPage from './CardPage.svelte';
	import ListPage from './ListPage.svelte';
	import ListPageSkeleton from './ListPageSkeleton.svelte';
	import CardPageSkeleton from './CardPageSkeleton.svelte';
	import ConfirmModal from '../ConfirmModal.svelte';
	import { getRecordId, getRecordLabel } from '$lib/utils/recordHelpers';

	interface Props {
		pageid: number;
		recordid?: string;
	}

	let { pageid, recordid }: Props = $props();

	// State
	let page: PageDefinition | null = $state(null);
	let captions: Record<string, string> = $state({});
	let options: Record<string, Record<string, string>> = $state({}); // Option field values (enum lookups)
	let lookups: Record<string, Record<string, string>> = $state({}); // Table relation lookup values
	let navigation: Record<string, string> = $state({}); // Navigation translations
	let pageLoading = $state(true);  // Loading page definition
	let dataLoading = $state(false); // Loading data after page definition is known
	let error = $state<string | null>(null);

	// Data for the page
	let record: Record<string, any> = $state({});
	let records: Array<Record<string, any>> = $state([]);

	// Track if record was successfully loaded (for distinguishing new vs existing)
	let isExistingRecord = $state(false);
	// Track the current record ID (may differ from URL recordid after insert)
	let currentRecordId = $state<string | undefined>(undefined);

	// Filters for list pages
	let currentFilters: import('$lib/types/api').TableFilter[] = $state([]);

	// Navigation data for card pages
	let recordIds: string[] = $state([]);
	let currentRecordIndex = $state(-1);


	// Load page definition and data
	onMount(async () => {
		try {
			pageLoading = true;
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
			navigation = result.navigation || {};
			pageLoading = false;

			// Now show skeleton while loading data
			dataLoading = true;

			// Load data based on page type
			if (page.page.type === 'Card') {
				await loadCardData();
				// Set breadcrumb for card page
				await setBreadcrumbForCardPage();
			} else if (page.page.type === 'List') {
				await loadListData();
				// Set breadcrumb for list page
				breadcrumb.setListPage(page.page.id, page.page.caption, navigation.home);
			}
		} catch (err) {
			error = err instanceof Error ? err.message : 'Unknown error';
			console.error('Error loading page:', err);
		} finally {
			pageLoading = false;
			dataLoading = false;
		}
	});

	// Set breadcrumb for card page (needs to fetch list page caption if available)
	async function setBreadcrumbForCardPage() {
		if (!page) return;

		let listPageCaption: string | undefined;

		// If card page has a list_page_id, fetch the list page caption
		if (page.page.list_page_id) {
			try {
				const listPageResponse = await fetch(`/api/pages/${page.page.list_page_id}`);
				if (listPageResponse.ok) {
					const listPageResult = await listPageResponse.json();
					if (listPageResult.success) {
						listPageCaption = listPageResult.data.page.caption;
					}
				}
			} catch (err) {
				console.error('Error fetching list page for breadcrumb:', err);
			}
		}

		// Get record label (primary key value like customer number)
		const recordLabelValue = getRecordLabel(record) || recordid;

		breadcrumb.setCardPage(
			page.page.id,
			page.page.caption,
			recordid,
			recordLabelValue,
			page.page.list_page_id,
			listPageCaption,
			navigation.home
		);
	}

	// Load data for card page
	async function loadCardData() {
		if (!page) return;

		try {
			if (recordid) {
				// Load specific record with captions (includes option values and lookups)
				const result = await api.getRecordWithCaptions(page.page.source_table, recordid);
				record = result.data;
				// Merge options from the record response (contains enum values)
				if (result.captions?.options) {
					options = result.captions.options;
				}
				// Merge lookups from the record response (table relation values)
				if (result.captions?.lookups) {
					lookups = result.captions.lookups;
				}
				isExistingRecord = true; // Successfully loaded an existing record
				currentRecordId = recordid;
			} else {
				// New record - open immediately, fetch options/lookups in background
				record = {};
				isExistingRecord = false;
				currentRecordId = undefined;
				options = {}; // Start with empty, load in background
				lookups = {}; // Start with empty, load in background

				// Fetch options and lookups non-blocking
				api.getTableOptionsAndLookups(page.page.source_table).then(result => {
					options = result.options;
					lookups = result.lookups;
				}).catch(err => {
					console.error('Failed to load options/lookups:', err);
				});
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
			isExistingRecord = false; // Record didn't exist or failed to load - treat as new
			currentRecordId = undefined;
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
			const listOptions: import('$lib/types/api').ListOptions = {};
			if (visibleFields.length > 0) {
				listOptions.fields = visibleFields;
			}
			if (currentFilters.length > 0) {
				listOptions.filters = currentFilters;
			}

			// Fetch records, options, and lookups in a single API call
			const response = await api.listRecordsWithOptions(page.page.source_table, listOptions);
			records = response.list.records || [];
			options = response.options;
			lookups = response.lookups || {};
		} catch (err) {
			console.error('Error loading list data:', err);
			records = [];
		}
	}

	// Get visible fields from page definition and user customizations
	function getVisibleFields(): string[] {
		if (!page || !page.page.layout.repeater?.fields) return [];

		// Load user customizations from localStorage (user-specific)
		let userId: string;
		currentUser.subscribe(user => {
			userId = user?.user_id || 'anonymous';
		})();

		const key = `page-customization-${userId}-${page.page.id}`;
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
				isExistingRecord = false; // Reset for new record
				currentRecordId = undefined;
				// Update URL to remove the recordid
				window.history.replaceState({}, '', `/pages/${page.page.id}`);
				break;
			case 'Delete':
				// Get record ID from record object or recordid prop
				const deleteId = getRecordId(record) || recordid;
				if (deleteId) {
					confirm.show(
						'Delete Record',
						`Are you sure you want to delete this ${page.page.caption}?`,
						async () => {
							try {
								await api.deleteRecord(page.page.source_table, deleteId);
								toast.success('Record deleted successfully');
								// Navigate back to the list page if available
								if (page.page.type === 'Card') {
									// Try to find the associated list page by convention
									// Customer Card (21) -> Customer List (22)
									const listPageId = page.page.id + 1;
									window.location.href = `/pages/${listPageId}`;
								}
							} catch (err) {
								console.error('Delete error:', err);
								toast.error('Failed to delete record');
							}
						}
					);
				}
				break;
			case 'Refresh':
				await loadCardData();
				break;
		}
	}

	// Handle save from card page
	async function handleCardSave(savedRecord: Record<string, any>): Promise<boolean> {
		if (!page) return false;

		try {
			if (isExistingRecord && currentRecordId) {
				// Update existing record (only if we successfully loaded an existing record)
				await api.modifyRecord(page.page.source_table, currentRecordId, savedRecord);
				toast.success('Record updated');
			} else {
				// Insert new record
				const insertedRecord = await api.insertRecord(page.page.source_table, savedRecord);
				// After successful insert, mark as existing for future saves
				const newRecordId = insertedRecord.no || insertedRecord.code || insertedRecord.id;
				isExistingRecord = true;
				currentRecordId = newRecordId;
				toast.success('Record created');
				// Navigate to the proper URL with the new record ID
				if (newRecordId && page.page.id) {
					// Update URL without full page reload
					window.history.replaceState({}, '', `/pages/${page.page.id}/${newRecordId}`);
				}
			}
			return true;
		} catch (err) {
			// Re-throw so CardPage can catch and revert the field value
			throw err;
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
					const editRecordId = getRecordId(selectedRecord);
					window.location.href = `/pages/${page.page.card_page_id}/${editRecordId}`;
				}
				break;
			case 'Delete':
				if (selectedRecord) {
					const deleteRecordId = getRecordId(selectedRecord);
					confirm.show(
						'Delete Record',
						'Are you sure you want to delete this record?',
						async () => {
							try {
								await api.deleteRecord(page.page.source_table, deleteRecordId!);
								await loadListData();
								toast.success('Record deleted');
							} catch (err) {
								toast.error('Failed to delete record');
							}
						}
					);
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

	// Handle save notification from list page (inline editing)
	// Note: ListPage already saves the record internally via handleCellBlur,
	// this callback is just for notification - no need to save again
	async function handleListSave(savedRecord: Record<string, any>, isNew: boolean) {
		if (!page) return;
		// Record is already saved by ListPage, just refresh the underlying data
		await loadListData();
	}

	// Handle delete from list page
	async function handleListDelete(deletedRecord: Record<string, any>) {
		if (!page) return;

		try {
			const recordId = deletedRecord['no'] || deletedRecord['code'] || deletedRecord['id'];
			await api.deleteRecord(page.page.source_table, recordId);
			await loadListData();
			toast.success('Record deleted');
		} catch (err) {
			toast.error('Failed to delete record');
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

{#if pageLoading}
	<!-- Initial loading - show list skeleton as default -->
	<ListPageSkeleton />
{:else if dataLoading && page}
	<!-- Page definition loaded, show type-specific skeleton while data loads -->
	{#if page.page.type === 'Card'}
		<CardPageSkeleton />
	{:else}
		<ListPageSkeleton />
	{/if}
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
			{options}
			{lookups}
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
			{options}
			{lookups}
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

<!-- Confirm Modal -->
<ConfirmModal
	open={$confirm.open}
	title={$confirm.title}
	message={$confirm.message}
	confirmText="Delete"
	variant="danger"
	onconfirm={confirm.confirm}
	oncancel={confirm.cancel}
/>
