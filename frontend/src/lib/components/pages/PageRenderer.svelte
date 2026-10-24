<script lang="ts">
	import { goto, replaceState } from '$app/navigation';
	import { withNavigationQuery, type NavigationQuery } from '$lib/utils/recordNavigation';
	import { onMount } from 'svelte';
	import type { PageDefinition } from '$lib/types/pages';
	import type { LookupData } from '$lib/types/api';
	import { fetchPage } from '$lib/services/pages';
	import { api } from '$lib/services/api';
	import { currentUser } from '$lib/stores/user';
	import { toast } from '$lib/stores/toast';
	import { confirm } from '$lib/stores/confirm';
	import { breadcrumb } from '$lib/stores/breadcrumb';
	import { cardTitle } from '$lib/utils/fieldHelpers';
	import { t, MSG, ERR, DLG, BTN, PAGE_ERR } from '$lib/services/i18n.svelte';
	import Button from '../Button.svelte';
	import CardPage from './CardPage.svelte';
	import ListPage from './ListPage.svelte';
	import ListPageSkeleton from './ListPageSkeleton.svelte';
	import CardPageSkeleton from './CardPageSkeleton.svelte';
	import ConfirmModal from '../ConfirmModal.svelte';
	import { getRecordId, getRecordLabel, getPrimaryKeyField, getPrimaryKeyFields, canInsertNewRecord, deepCopy, applyServerValues } from '$lib/utils/recordHelpers';
	import { createNavigationActions } from '$lib/utils/navigationHelpers';
	import { getJson } from '$lib/utils/storage';
	import { estimateRowsPerPage, windowSize, windowOffsetFor, type ListWindowRequest } from '$lib/utils/listWindow';
	import { apiFlowFilters } from '$lib/utils/flowFilter';
	import { currentLanguage } from '$lib/stores/session';
	import { get } from 'svelte/store';

	interface Props {
		pageid: number;
		recordid?: string;
		initialFilter?: string | string[]; // ?filter=field=expression (several: one per field)
		returnUrl?: string; // opened from a drilldown/card action: Esc and close go back here
		initialSelect?: string; // record key to select (returning from a drilldown)
		startNew?: boolean; // a list opened from a lookup's New: start a new record
		navigationQuery?: NavigationQuery; // card opened from a list: navigate through its records
		// Opened from a codeunit (RunModal) in a modal: no breadcrumb or navigation; the list is
		// filtered (initialFilters) and read-only, and onclose reports how it was closed — in
		// lookup mode Enter / double-click picks a record ('LookupOK'), Esc cancels
		embedded?: boolean;
		initialFilters?: import('$lib/types/api').TableFilter[];
		lookupMode?: boolean;
		onclose?: (action: 'OK' | 'LookupOK' | 'Cancel', key?: string) => void; // key: the picked record's id
		// A subpage in a card: a list page tied to the card's record — fixed filters (the link),
		// new records get the link values; Esc in navigation mode goes back to the card (onexit);
		// onlineschange after a line was saved or deleted (the card's totals)
		subpage?: boolean;
		fixedFilters?: import('$lib/types/api').FixedFilter[];
		newRecordDefaults?: Record<string, any>;
		onexit?: () => void;
		onlineschange?: () => void;
		// A subpage: when its card bumps this (the header was saved, a function asked for it),
		// the lines reload — with the user's filters, search and position
		refreshToken?: number;
	}

	let {
		pageid,
		recordid,
		initialFilter,
		returnUrl,
		initialSelect,
		startNew = false,
		navigationQuery,
		embedded = false,
		initialFilters,
		lookupMode = false,
		onclose,
		subpage = false,
		fixedFilters = [],
		newRecordDefaults = {},
		onexit,
		onlineschange,
		refreshToken = 0
	}: Props = $props();

	// A subpage reloads its lines when its card asks (refreshToken changes after the first load)
	let lastRefreshToken: number | undefined = undefined;
	$effect(() => {
		const token = refreshToken;
		if (lastRefreshToken !== undefined && token !== lastRefreshToken && page) void loadListData();
		lastRefreshToken = token;
	});

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
	let sessionExpired = $state(false); // the page request found no valid session (401)
	let errorStatus = $state(0); // HTTP status of a failed page request

	// A new login: full load of the login page (login starts the app fresh)
	function logInAgain() {
		currentUser.clear();
		window.location.href = '/login';
	}

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

	// RunModal lookup: the record the list's cursor is on (OK picks it)
	let lookupCurrent = $state<Record<string, any> | undefined>(undefined);

	// Filters for list pages - parse initialFilter if provided (format: "field=expression")
	// svelte-ignore state_referenced_locally - the initial filters only set the starting state
	let currentFilters: import('$lib/types/api').TableFilter[] = $state(initialFilters ?? parseInitialFilter(initialFilter));
	// FlowFilters as typed in the filter pane (e.g. Date Filter "01.01.26..31.03.26"); sent as ISO
	let currentFlowFilters: import('$lib/types/api').TableFilter[] = $state([]);

	function parseInitialFilter(filter: string | string[] | undefined): import('$lib/types/api').TableFilter[] {
		const filters: import('$lib/types/api').TableFilter[] = [];
		for (const f of Array.isArray(filter) ? filter : filter ? [filter] : []) {
			const eqIndex = f.indexOf('=');
			if (eqIndex > 0) filters.push({ field: f.substring(0, eqIndex), expression: f.substring(eqIndex + 1) });
		}
		return filters;
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
				// 401: no valid session any more (expired, or e.g. after an update) — offer a new login.
				// The texts are translated in the template: translations may still be loading here.
				sessionExpired = response.status === 401;
				errorStatus = response.status;
				let serverMessage = '';
				if (!sessionExpired) {
					try {
						serverMessage = (await response.json())?.error ?? '';
					} catch {
						// not JSON (e.g. a proxy error page while the server restarts)
					}
				}
				throw new Error(serverMessage);
			}

			const result = await response.json();
			if (!result.success) {
				throw new Error(result.error || 'Failed to load page');
			}

			page = result.data;
			// Opened from a codeunit: a read-only list (the codeunit waits for the user)
			if (embedded && page) page = { ...page, page: { ...page.page, editable: false } };
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
				// Set breadcrumb for list page (not for a page opened from a codeunit)
				if (!embedded && !subpage) breadcrumb.setListPage(page.page.id, page.page.caption, navigation.home);
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
			navigation.home,
			recordid ? cardTitle(page.page.caption, page.page.data_caption_fields, record, options) : undefined
		);
	}

	// Load data for card page
	async function loadCardData() {
		if (!page) return;

		try {
			// A setup card opens on the table's single record (blank key)
			const cardRecordId = recordid || (page.page.setup_table ? '' : undefined);
			if (cardRecordId !== undefined) {
				// Load specific record with captions (includes option values and lookups)
				const result = await api.getRecordWithCaptions(page.page.source_table, cardRecordId);
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
				currentRecordId = cardRecordId;
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
				recordIds = await api.getRecordIDs(page.page.source_table, navigationQuery);

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

	// The list's query: filters, FlowFilters, search over the visible stored fields, sort.
	// Shared by the windows of the list and "delete all selected", so both mean the same records.
	function listQuery(visibleFields = getVisibleFields(), flowFields = new Set(page?.page.flow_fields ?? [])) {
		const query: import('$lib/types/api').ListOptions = {};
		if (!page) return query;
		if (currentFilters.length > 0) {
			query.filters = currentFilters;
		}
		if (fixedFilters.length > 0) {
			query.fixed_filters = fixedFilters; // a subpage's link to its card
		}
		const flowFilters = apiFlowFilters(currentFlowFilters, page.page.flow_filter_fields, get(currentLanguage));
		if (flowFilters.length > 0) {
			query.flow_filters = flowFilters;
		}
		if (listSearch.trim()) {
			// FlowFields are computed, not stored: the server can not search them
			query.search = listSearch.trim();
			query.search_fields = visibleFields.filter((f) => !flowFields.has(f));
		}
		if (listSort) {
			query.sort_by = listSort.field;
			query.sort_order = listSort.direction;
		}
		return query;
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
				...listQuery(visibleFields, flowFields),
				offset,
				limit: windowLimit
			};
			if (visibleFields.length > 0) {
				listOptions.fields = visibleFields;
			}
			// The page's triggers fill its calculated fields (OnAfterGetRecord)
			if (page.page.has_record_triggers) listOptions.page_id = page.page.id;

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
		if (!page || !initialSelect || (currentFilters.length > 0 && !embedded)) return 0;
		try {
			const ids = await api.getRecordIDs(
				page.page.source_table,
				currentFilters.length > 0 ? { filters: currentFilters } : undefined
			);
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
				replaceState(`/pages/${page.page.id}`, {});
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
									goto(`/pages/${listPageId}`);
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

	// A subpage's lines changed: read the card's FlowFields (totals) again — only those, so
	// nothing the user is typing on the card is touched
	async function refreshCardTotals() {
		if (!page || currentRecordId === undefined || !record) return;
		try {
			const fresh = await api.getRecordWithCaptions(page.page.source_table, currentRecordId);
			for (const field of page.page.flow_fields ?? []) {
				if (fresh.data && field in fresh.data) record[field] = fresh.data[field];
			}
		} catch (err) {
			console.error('Failed to refresh the card totals:', err);
		}
	}

	// Handle save from card page
	async function handleCardSave(savedRecord: Record<string, any>, opts?: { leavingKey?: boolean }): Promise<boolean> {
		if (!page) return false;

		try {
			if (isExistingRecord && currentRecordId !== undefined) {
				// Update existing record (only if we successfully loaded an existing record)
				const sentRecord = deepCopy(savedRecord);
				const modified = await api.modifyRecord(page.page.source_table, currentRecordId, savedRecord);
				// Values a trigger changed (tidied, recalculated) show at once
				applyServerValues(savedRecord, sentRecord, modified);
				// Renamed (key changed, ): address the record by its new key from now on
				const newRecordId = getRecordId(modified, primaryKeyField, primaryKeyFieldsList);
				if (newRecordId && newRecordId !== currentRecordId) {
					recordIds = recordIds.map((id) => (id === currentRecordId ? newRecordId : id));
					currentRecordId = newRecordId;
					replaceState(withNavigationQuery(`/pages/${page.page.id}/${newRecordId}`, navigationQuery), {});
				}
				toast.success(t(MSG.RECORD_UPDATED));
			} else {
				// Insert a new record only once its primary key is filled in (Record Entry
				// rule); until then field changes stay on the form
				if (!canInsertNewRecord(savedRecord, {}, primaryKeyFieldsList, page.page.auto_key) && !(page.page.auto_key && opts?.leavingKey)) {
					return false;
				}
				const sentRecord = deepCopy(savedRecord);
				const insertedRecord = await api.insertRecord(page.page.source_table, savedRecord);
				applyServerValues(savedRecord, sentRecord, insertedRecord);
				// After successful insert, mark as existing for future saves
				const newRecordId = getRecordId(insertedRecord, primaryKeyField, primaryKeyFieldsList);
				isExistingRecord = true;
				currentRecordId = newRecordId;
				toast.success(t(MSG.RECORD_CREATED));
				// Navigate to the proper URL with the new record ID
				if (newRecordId && page.page.id) {
					// Update URL without full page reload
					replaceState(`/pages/${page.page.id}/${newRecordId}`, {});
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
					goto(`/pages/${page.page.card_page_id}`);
				}
				break;
			case 'Edit':
				if (selectedRecord && page.page.card_page_id) {
					// Navigate to card page with record ID
					const editRecordId = getRecordId(selectedRecord, primaryKeyField, primaryKeyFieldsList);
					goto(`/pages/${page.page.card_page_id}/${editRecordId}`);
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
				onlineschange?.(); // a subpage: the card's totals too
				break;
		}
	}

	// Handle row click in list page
	function handleRowClick(clickedRecord: Record<string, any>, navigation?: NavigationQuery) {
		if (embedded) {
			// Opened from a codeunit: Enter / double-click picks the record in lookup mode
			if (lookupMode) onclose?.('LookupOK', getRecordId(clickedRecord, primaryKeyField, primaryKeyFieldsList) ?? '');
			return;
		}
		if (!page || !page.page.card_page_id) return;

		const recordId = getRecordId(clickedRecord, primaryKeyField, primaryKeyFieldsList);
		goto(withNavigationQuery(`/pages/${page.page.card_page_id}/${recordId}`, navigation));
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
		onlineschange?.();
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
			onlineschange?.();
			toast.success(t(MSG.RECORD_DELETED));
		} catch (err) {
			toast.error(err instanceof Error && err.message ? err.message : t(ERR.FAILED_DELETE));
			console.error('Delete error:', err);
			throw err;
		}
	}

	// Delete the records selected on the list (multi-select): one transaction on the server;
	// when a record cannot be deleted nothing is, and the message names it
	async function handleListBulkDelete(request: { keys?: string[]; all?: boolean }, count: number) {
		if (!page) return;
		try {
			const deleted = await api.deleteRecords(page.page.source_table, request, request.all ? listQuery() : undefined);
			await loadListData();
			onlineschange?.();
			toast.success(t(MSG.RECORDS_DELETED, deleted.toLocaleString(get(currentLanguage))));
		} catch (err) {
			toast.error(err instanceof Error && err.message ? err.message : t(ERR.FAILED_DELETE));
			throw err;
		}
	}

	// Paste records copied on a list (Ctrl+V): one transaction on the server; when a record
	// cannot be inserted nothing is, and the message names it. Returns the inserted keys.
	async function handleListPaste(records: Record<string, unknown>[]): Promise<string[]> {
		if (!page) return [];
		try {
			// In a subpage the pasted lines belong to the card's record: its link values
			const rows = subpage ? records.map((r) => ({ ...r, ...newRecordDefaults })) : records;
			const keys = await api.insertRecords(page.page.source_table, rows);
			await loadListData();
			onlineschange?.();
			toast.success(t(MSG.RECORDS_PASTED, keys.length.toLocaleString(get(currentLanguage))));
			return keys;
		} catch (err) {
			toast.error(err instanceof Error && err.message ? err.message : t(ERR.FAILED_SAVE_RECORD));
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
		goto(withNavigationQuery(`/pages/${page.page.id}/${targetRecordId}`, navigationQuery));
	}

	// Create navigation actions using shared helper
	const navigationActions = createNavigationActions(
		() => ({ recordIds, currentRecordIndex }),
		navigateToRecord
	);

	// Handle closing the card page
	// The loading skeleton appears only when loading takes longer than this: a page that loads
	// quickly (the usual case) shows no flash of placeholder before its content
	const skeletonDelay = 1000;
	let showSkeleton = $state(false);
	// One loading period from the page definition through its data: the timer does not
	// restart between the two
	const loading = $derived(pageLoading || dataLoading);
	$effect(() => {
		if (!loading) {
			showSkeleton = false;
			return;
		}
		const timer = setTimeout(() => (showSkeleton = true), skeletonDelay);
		return () => clearTimeout(timer);
	});

	function handleCardClose() {
		if (page?.page.list_page_id) {
			goto(`/pages/${page.page.list_page_id}`);
		} else {
			// Fallback: go back in history
			window.history.back();
		}
	}
</script>

{#if pageLoading || (dataLoading && page)}
	<!-- Loading: the skeleton only after skeletonDelay — a quick load shows no placeholder -->
	{#if showSkeleton}
		{#if page?.page.type === 'Card'}
			<CardPageSkeleton />
		{:else}
			<ListPageSkeleton />
		{/if}
	{/if}
{:else if error !== null}
	<!-- error is "" for a failed page request: the screen translates its status -->
	<div class="flex items-center justify-center h-full">
		<div class="text-center">
			<div class="text-red-600 dark:text-red-400 text-5xl mb-4">⚠</div>
			<h2 class="text-xl font-semibold text-gray-800 dark:text-gray-200 mb-2">{t(PAGE_ERR.TITLE)}</h2>
			<p class="text-gray-600 dark:text-gray-400 max-w-md">
				{sessionExpired
					? t(PAGE_ERR.SESSION_EXPIRED)
					: error || t(PAGE_ERR.LOAD_FAILED, String(errorStatus))}
			</p>
			<div class="mt-6 flex justify-center gap-3">
				<Button variant="primary" onclick={logInAgain}>{t(PAGE_ERR.LOG_IN_AGAIN)}</Button>
				{#if !sessionExpired}
					<Button variant="secondary" onclick={() => window.location.reload()}>{t(PAGE_ERR.TRY_AGAIN)}</Button>
				{/if}
			</div>
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
			onrefresh={refreshCardTotals}
		/>
	{:else if page.page.type === 'List'}
		{#if embedded && lookupMode}
			<!-- Lookup mode (RunModal): OK picks the cursor's record, Cancel closes without one -->
			<div class="lookup-page">
				<div class="lookup-list">
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
					{startNew}
					onclose={embedded ? () => onclose?.(lookupMode ? 'Cancel' : 'OK') : subpage ? onexit : undefined}
					pickOnEnter={embedded}
					{subpage}
					{fixedFilters}
					{newRecordDefaults}
					oncurrent={embedded && lookupMode ? (r) => (lookupCurrent = r) : undefined}
					onwindow={handleListWindow}
					onaction={handleListAction}
					onrowclick={handleRowClick}
					onsave={handleListSave}
					ondelete={handleListDelete}
					onbulkdelete={handleListBulkDelete}
					onpaste={handleListPaste}
					onfilter={handleFilterChange}
				/>
				</div>
				<div class="lookup-buttons">
					<Button variant="secondary" onclick={() => onclose?.('Cancel')}>{t(BTN.CANCEL)}</Button>
					<Button
						variant="primary"
						disabled={!lookupCurrent}
						onclick={() => onclose?.('LookupOK', getRecordId(lookupCurrent, primaryKeyField, primaryKeyFieldsList) ?? '')}
					>{t(BTN.OK)}</Button>
				</div>
			</div>
		{:else}
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
				{startNew}
				onclose={embedded ? () => onclose?.(lookupMode ? 'Cancel' : 'OK') : subpage ? onexit : undefined}
				pickOnEnter={embedded}
				{subpage}
				{fixedFilters}
				{newRecordDefaults}
				oncurrent={embedded && lookupMode ? (r) => (lookupCurrent = r) : undefined}
				onwindow={handleListWindow}
				onaction={handleListAction}
				onrowclick={handleRowClick}
				onsave={handleListSave}
				ondelete={handleListDelete}
				onbulkdelete={handleListBulkDelete}
				onpaste={handleListPaste}
				onfilter={handleFilterChange}
			/>
		{/if}
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
	confirmText={$confirm.confirmText ?? t(BTN.DELETE)}
	variant={$confirm.variant ?? 'danger'}
	onconfirm={confirm.confirm}
	oncancel={confirm.cancel}
/>

<style>
	/* RunModal in lookup mode: the list with OK and Cancel below it */
	.lookup-page {
		@apply flex flex-col h-full gap-3;
	}

	.lookup-list {
		@apply flex-1 min-h-0;
	}

	.lookup-buttons {
		@apply flex justify-end gap-2;
	}
</style>
