<script lang="ts">
	import { onMount } from 'svelte';
	import type { PageDefinition } from '$lib/types/pages';
	import type { LookupData } from '$lib/types/api';
	import { fetchPage } from '$lib/services/pages';
	import { api } from '$lib/services/api';
	import { currentUser } from '$lib/stores/user';
	import { toast } from '$lib/stores/toast';
	import { confirm } from '$lib/stores/confirm';
	import { breadcrumb } from '$lib/stores/breadcrumb';
	import { t, MSG, ERR, DLG } from '$lib/services/i18n.svelte';
	import CardPage from './CardPage.svelte';
	import ListPage from './ListPage.svelte';
	import ListPageSkeleton from './ListPageSkeleton.svelte';
	import CardPageSkeleton from './CardPageSkeleton.svelte';
	import ConfirmModal from '../ConfirmModal.svelte';
	import { getRecordId, getRecordLabel, getPrimaryKeyField, getPrimaryKeyFields } from '$lib/utils/recordHelpers';
	import { createNavigationActions } from '$lib/utils/navigationHelpers';
	import { getJson } from '$lib/utils/storage';
	import { estimateRowsPerPage, windowSize, windowOffsetFor, type ListWindowRequest } from '$lib/utils/listWindow';
	import { apiFlowFilters } from '$lib/utils/flowFilter';
	import { currentLanguage } from '$lib/stores/session';
	import { get } from 'svelte/store';

	interface Props {
		pageid: number;
		recordid?: string;
		initialFilter?: string;
		returnUrl?: string; // opened from a drilldown/card action: Esc and close go back here
		initialSelect?: string; // record key to select (returning from a drilldown)
	}

	let { pageid, recordid, initialFilter, returnUrl, initialSelect }: Props = $props();

	// State
	let page: PageDefinition | null = $state(null);
	let captions: Record<string, string> = $state({});
	let fieldTypes: Record<string, string> = $state({}); // Field type metadata (e.g., "bool", "code", "text")
	let options: Record<string, Record<string, string>> = $state({}); // Option field values (enum lookups)
	let lookups: Record<string, LookupData> = $state({}); // Table relation lookup values
	let navigation: Record<string, string> = $state({}); // Navigation translations
	let pageLoading = $state(true);  // Loading page definition
	let dataLoading = $state(false); // Loading data after page definition is known
	let error = $state<string | null>(null);

	// Data for the page
	let record: Record<string, any> = $state({});
	let records: Array<Record<string, any>> = $state([]);

	// List pages load a window of rows (the visible page plus two pages each way, see
	// listWindow.ts); search and sort run on the server over all records
	let listTotal = $state(0);
	let windowOffset = $state(0);
	let windowLimit = $state(windowSize(estimateRowsPerPage(typeof window !== 'undefined' ? window.innerHeight : 800)));
	let listSearch = $state('');
	let listSort = $state<ListWindowRequest['sort']>(null);
	let listLoadSeq = 0; // only the latest request may set the rows (fast typing in search)

	// Track if record was successfully loaded (for distinguishing new vs existing)
	let isExistingRecord = $state(false);
	// Track the current record ID (may differ from URL recordid after insert)
	let currentRecordId = $state<string | undefined>(undefined);

	// Filters for list pages - parse initialFilter if provided (format: "field=expression")
	let currentFilters: import('$lib/types/api').TableFilter[] = $state(parseInitialFilter(initialFilter));
	// FlowFilters as typed in the filter pane (e.g. Date Filter "01.01.26..31.03.26"); sent as ISO
	let currentFlowFilters: import('$lib/types/api').TableFilter[] = $state([]);

	function parseInitialFilter(filter: string | undefined): import('$lib/types/api').TableFilter[] {
		if (!filter) return [];
		const eqIndex = filter.indexOf('=');
		if (eqIndex <= 0) return [];
		return [{ field: filter.substring(0, eqIndex), expression: filter.substring(eqIndex + 1) }];
	}

	// Navigation data for card pages
	let recordIds: string[] = $state([]);
	let currentRecordIndex = $state(-1);

	// Get primary key field name from page definition
	const primaryKeyField = $derived(getPrimaryKeyField(page));
	const primaryKeyFieldsList = $derived(getPrimaryKeyFields(page));

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
			fieldTypes = result.captions?.field_types || {};
			navigation = result.navigation || {};
			pageLoading = false;

			if (!page) {
				throw new Error('Page data is null');
			}

			// Now show skeleton while loading data
			dataLoading = true;

			// Load data based on page type
			if (page.page.type === 'Card') {
				await loadCardData();
				// Set breadcrumb for card page
				await setBreadcrumbForCardPage();
			} else if (page.page.type === 'List') {
				await loadListData(await initialListOffset());
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
				// New record - fetch options/lookups before rendering
				record = {};
				isExistingRecord = false;
				currentRecordId = undefined;

				// Fetch options and lookups (await to ensure they're loaded before render)
				try {
					const result = await api.getTableOptionsAndLookups(page.page.source_table);
					options = result.options;
					lookups = result.lookups;
				} catch (err) {
					console.error('Failed to load options/lookups:', err);
					options = {};
					lookups = {};
				}
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

	// Load a window of the list (offset/limit, filters, search, sort). windowOffset and
	// records change together when the response arrives, so they always match.
	async function loadListData(offset: number = windowOffset) {
		if (!page) return;
		const seq = ++listLoadSeq;

		try {
			// Determine which fields are visible based on customizations
			const visibleFields = getVisibleFields();
			const flowFields = new Set(page.page.flow_fields ?? []);

			// Load records with only visible fields to avoid expensive FlowField calculations
			const listOptions: import('$lib/types/api').ListOptions = {
				offset,
				limit: windowLimit
			};
			if (visibleFields.length > 0) {
				listOptions.fields = visibleFields;
			}
			if (currentFilters.length > 0) {
				listOptions.filters = currentFilters;
			}
			const flowFilters = apiFlowFilters(currentFlowFilters, page.page.flow_filter_fields, get(currentLanguage));
			if (flowFilters.length > 0) {
				listOptions.flow_filters = flowFilters;
			}
			if (listSearch.trim()) {
				// FlowFields are computed, not stored: the server can not search them
				listOptions.search = listSearch.trim();
				listOptions.search_fields = visibleFields.filter((f) => !flowFields.has(f));
			}
			if (listSort) {
				listOptions.sort_by = listSort.field;
				listOptions.sort_order = listSort.direction;
			}

			// Fetch records, options, and lookups in a single API call
			const response = await api.listRecordsWithOptions(page.page.source_table, listOptions);
			if (seq !== listLoadSeq) return;
			const rows = response.list.records || [];
			const total = response.list.total ?? rows.length;
			// Past the end (rows deleted, search narrowed): load the last window instead
			if (rows.length === 0 && total > 0 && offset > 0) {
				return loadListData(Math.max(0, total - windowLimit));
			}
			records = rows;
			listTotal = total;
			windowOffset = offset;
			options = response.options;
			lookups = response.lookups || {};
		} catch (err) {
			if (seq !== listLoadSeq) return;
			console.error('Error loading list data:', err);
			records = [];
			listTotal = 0;
		}
	}

	// Returning from a drilldown to an unfiltered list: start with the window around the
	// record to select (its position in primary key order), not at the top
	async function initialListOffset(): Promise<number> {
		if (!page || !initialSelect || currentFilters.length > 0) return 0;
		try {
			const ids = await api.getRecordIDs(page.page.source_table);
			const position = ids.indexOf(initialSelect);
			if (position < 0) return 0;
			const perPage = estimateRowsPerPage(typeof window !== 'undefined' ? window.innerHeight : 800);
			return windowOffsetFor(position, perPage, ids.length);
		} catch {
			return 0;
		}
	}

	// ListPage asks for another window: moved past the loaded rows, searched or sorted
	async function handleListWindow(request: ListWindowRequest) {
		if (request.limit !== undefined) windowLimit = request.limit;
		if (request.search !== undefined) listSearch = request.search;
		if (request.sort !== undefined) listSort = request.sort;
		await loadListData(request.offset !== undefined ? Math.max(0, request.offset) : windowOffset);
	}

	// Get visible fields from page definition and user customizations
	function getVisibleFields(): string[] {
		if (!page || !page.page.layout.repeater?.fields) return [];

		// Load user customizations from storage (user-specific)
		let userId = 'anonymous';
		currentUser.subscribe(user => {
			userId = user?.user_id || 'anonymous';
		})();

		const key = `page-customization-${userId}-${page.page.id}`;
		const customizations = getJson<Record<string, { visible: boolean }>>(key, {});

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
				const deleteId = getRecordId(record, primaryKeyField, primaryKeyFieldsList) || recordid;
				if (deleteId) {
					confirm.show(
						t(DLG.DELETE_RECORD_TITLE),
						t(DLG.DELETE_CONFIRM, page.page.caption),
						async () => {
							try {
								await api.deleteRecord(page!.page.source_table, deleteId);
								toast.success(t(MSG.RECORD_DELETED_SUCCESS));
								// Navigate back to the list page if available
								if (page!.page.type === 'Card') {
									// Try to find the associated list page by convention
									// Customer Card (21) -> Customer List (22)
									const listPageId = page!.page.id + 1;
									window.location.href = `/pages/${listPageId}`;
								}
							} catch (err) {
								console.error('Delete error:', err);
								toast.error(err instanceof Error && err.message ? err.message : t(ERR.FAILED_DELETE));
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
				toast.success(t(MSG.RECORD_UPDATED));
			} else {
				// Insert new record
				const insertedRecord = await api.insertRecord(page.page.source_table, savedRecord);
				// After successful insert, mark as existing for future saves
				const newRecordId = getRecordId(insertedRecord, primaryKeyField, primaryKeyFieldsList);
				isExistingRecord = true;
				currentRecordId = newRecordId;
				toast.success(t(MSG.RECORD_CREATED));
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
					const editRecordId = getRecordId(selectedRecord, primaryKeyField, primaryKeyFieldsList);
					window.location.href = `/pages/${page.page.card_page_id}/${editRecordId}`;
				}
				break;
			case 'Delete':
				if (selectedRecord) {
					const deleteRecordId = getRecordId(selectedRecord, primaryKeyField, primaryKeyFieldsList);
					confirm.show(
						t(DLG.DELETE_RECORD_TITLE),
						t(DLG.DELETE_RECORD_CONFIRM),
						async () => {
							try {
								await api.deleteRecord(page!.page.source_table, deleteRecordId!);
								await loadListData();
								toast.success(t(MSG.RECORD_DELETED));
							} catch (err) {
								toast.error(err instanceof Error && err.message ? err.message : t(ERR.FAILED_DELETE));
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

		const recordId = getRecordId(clickedRecord, primaryKeyField, primaryKeyFieldsList);
		window.location.href = `/pages/${page.page.card_page_id}/${recordId}`;
	}

	// Handle save notification from list page (inline editing)
	// Note: ListPage already saves the record internally via handleCellBlur,
	// this callback is just for notification - no need to save again
	async function handleListSave(savedRecord: Record<string, any>, isNew: boolean) {
		if (!page) return;
		// Record is already saved by ListPage, just refresh the underlying data. Not awaited:
		// ListPage awaits onsave inside its save, and the next cell's save must not wait for
		// a list reload
		void loadListData();
	}

	// Handle delete from list page
	async function handleListDelete(deletedRecord: Record<string, any>) {
		if (!page) return;

		try {
			const recordId = getRecordId(deletedRecord, primaryKeyField, primaryKeyFieldsList);
			if (!recordId) {
				console.error('Cannot delete record: no record ID found', deletedRecord);
				toast.error('Cannot delete: record has no ID');
				return;
			}
			await api.deleteRecord(page.page.source_table, recordId);
			await loadListData();
			toast.success(t(MSG.RECORD_DELETED));
		} catch (err) {
			toast.error(err instanceof Error && err.message ? err.message : t(ERR.FAILED_DELETE));
			console.error('Delete error:', err);
			throw err;
		}
	}

	// Handle filter change from list page
	async function handleFilterChange(filters: import('$lib/types/api').TableFilter[], flowFilters: import('$lib/types/api').TableFilter[] = []) {
		currentFilters = filters;
		currentFlowFilters = flowFilters;
		await loadListData(0);
	}

	// Navigation functions for card pages
	function navigateToRecord(targetRecordId: string) {
		if (!page) return;
		window.location.href = `/pages/${page.page.id}/${targetRecordId}`;
	}

	// Create navigation actions using shared helper
	const navigationActions = createNavigationActions(
		() => ({ recordIds, currentRecordIndex }),
		navigateToRecord
	);

	// Handle closing the card page
	function handleCardClose() {
		if (page?.page.list_page_id) {
			window.location.href = `/pages/${page.page.list_page_id}`;
		} else {
			// Fallback: go back in history
			window.history.back();
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
			{fieldTypes}
			{options}
			{lookups}
			onaction={handleCardAction}
			onsave={handleCardSave}
			onclose={handleCardClose}
			navigationEnabled={page.page.enable_navigation || false}
			canNavigateFirst={currentRecordIndex > 0}
			canNavigatePrevious={currentRecordIndex > 0}
			canNavigateNext={currentRecordIndex >= 0 && currentRecordIndex < recordIds.length - 1}
			canNavigateLast={currentRecordIndex >= 0 && currentRecordIndex < recordIds.length - 1}
			onNavigateFirst={navigationActions.navigateFirst}
			onNavigatePrevious={navigationActions.navigatePrevious}
			onNavigateNext={navigationActions.navigateNext}
			onNavigateLast={navigationActions.navigateLast}
		/>
	{:else if page.page.type === 'List'}
		<ListPage
			{page}
			{records}
			{captions}
			{fieldTypes}
			{options}
			{lookups}
			{currentFilters}
			{currentFlowFilters}
			total={listTotal}
			{windowOffset}
			{returnUrl}
			initialSelectKey={initialSelect}
			onwindow={handleListWindow}
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
