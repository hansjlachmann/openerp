<script lang="ts">
	import type { PageDefinition, Field } from '$lib/types/pages';
	import type { TableFilter, LookupData, DialogResult } from '$lib/types/api';
	import { goto } from '$app/navigation';
	import { toast } from '$lib/stores/toast';
	import { confirm } from '$lib/stores/confirm';
	import { t, MSG, ERR, DLG, BTN, LIST } from '$lib/services/i18n.svelte';
	import Button from '$lib/components/Button.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import ModalCardPage from './ModalCardPage.svelte';
	import LookupDropdown from './LookupDropdown.svelte';
	import OptionDropdown from './OptionDropdown.svelte';
	import CustomizeFieldsModal from './CustomizeFieldsModal.svelte';
	import FilterPane from './FilterPane.svelte';
	import ConfirmModal from '../ConfirmModal.svelte';
	import Modal from '../Modal.svelte';
	import ProgressModal from '../ProgressModal.svelte';
	import { startJob, type SyncJobResult } from '$lib/services/jobs';
	import PlusIcon from '$lib/components/icons/PlusIcon.svelte';
	import EditIcon from '$lib/components/icons/EditIcon.svelte';
	import TrashIcon from '$lib/components/icons/TrashIcon.svelte';
	import RefreshIcon from '$lib/components/icons/RefreshIcon.svelte';
	import { shortcuts, normalizeShortcut, getShortcutKey, commonShortcuts } from '$lib/utils/shortcuts';
	import { cn } from '$lib/utils/cn';
	import { api } from '$lib/services/api';
	import { currentUser } from '$lib/stores/user';
	import { companySwitchOpen } from '$lib/stores/companySwitch';
	import { get } from 'svelte/store';
	import { getFieldCaption, getFieldStyleClasses, formatValue, formatOptionValue, formatLookupValue, isItemVisible, isDateType, isDateTimeType, formatDate, formatDateTime, type ItemCustomization } from '$lib/utils/fieldHelpers';
	import { currentLanguage } from '$lib/stores/session';
	import { loadPageCustomizations, savePageCustomizations, loadColumnWidths, saveColumnWidths, loadRowNumbersPreference, saveRowNumbersPreference } from '$lib/utils/customizationStorage';
	import { tick } from 'svelte';
	import { needsShift, windowOffsetFor, windowSize, type ListWindowRequest } from '$lib/utils/listWindow';
	import { getRecordId, getRecordKey, getPrimaryKeyField, getPrimaryKeyFields, deepCopy, hasRecordChanged, hasUserEdits, sameFieldValue, shouldInsertNewRecord, stripInternalFields, findSelectedRecord } from '$lib/utils/recordHelpers';

	interface Props {
		page: PageDefinition;
		records?: Array<Record<string, any>>;
		captions?: Record<string, string>;
		fieldTypes?: Record<string, string>; // Field type metadata (e.g., "bool", "code", "text")
		options?: Record<string, Record<string, string>>; // Option field values (enum lookups)
		lookups?: Record<string, LookupData>; // Table relation lookup values
		currentFilters?: TableFilter[];
		// Windowed loading (PageRenderer): records is a window of the list starting at
		// windowOffset; total counts all matching records; onwindow loads another window
		total?: number;
		windowOffset?: number;
		onwindow?: (request: ListWindowRequest) => Promise<void>;
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
		fieldTypes = {},
		options = {},
		lookups = {},
		currentFilters = [],
		total = records.length,
		windowOffset = 0,
		onwindow,
		onaction,
		onrowclick,
		onsave,
		ondelete,
		onfilter
	}: Props = $props();

	// Get primary key field name from page definition
	const primaryKeyField = $derived(getPrimaryKeyField(page));
	// Get all primary key fields (supports composite keys for delayed insert)
	const primaryKeyFieldsList = $derived(getPrimaryKeyFields(page));

	// Customization state
	let customizeModalOpen = $state(false);
	let columnCustomizations = $state<Record<string, ItemCustomization>>({});

	// Filter pane state
	let filterPaneOpen = $state(false);

	// Quick search state
	let searchQuery = $state('');
	let searchInputElement: HTMLInputElement | null = null;

	// Sort state
	let sortField = $state<string | null>(null);
	let sortDirection = $state<'asc' | 'desc'>('asc');

	// Column resize state
	let isResizing = $state(false);
	let resizeField = $state<string | null>(null);
	let resizeStartX = $state(0);
	let resizeStartWidth = $state(0);
	let columnWidths = $state<Record<string, number>>({});

	// Row numbers state
	let showRowNumbers = $state(false);

	// 3-state cell model: navigation → cell-selected → cell-editing
	let cellState = $state<'navigation' | 'cell-selected' | 'cell-editing'>('navigation');
	let cellEditSnapshot = $state<any>(undefined); // value before editing, for Escape revert
	let editableActive = $state(false); // whether editableRecords is initialized

	// Editable copy of records for edit mode (to avoid mutating props)
	let editableRecords = $state<Array<Record<string, any>>>([]);

	// Derived state helpers
	const isNavigation = $derived(cellState === 'navigation');
	const isCellSelected = $derived(cellState === 'cell-selected');
	const isCellEditing = $derived(cellState === 'cell-editing');

	// Locale for date formatting (subscribe to currentLanguage store)
	let locale = $state('en-US');
	currentLanguage.subscribe((lang) => { locale = lang; });

	/** Format a cell value with date/datetime awareness */
	function formatCellValue(value: any, fieldSource: string): string {
		const ft = fieldTypes[fieldSource];
		if (isDateType(ft)) return formatDate(String(value ?? ''), locale);
		if (isDateTimeType(ft)) return formatDateTime(String(value ?? ''), locale);
		return formatValue(value);
	}

	/** Get the HTML input type for a cell-editing field */
	function getCellInputType(fieldSource: string): string {
		const ft = fieldTypes[fieldSource];
		if (isDateType(ft)) return 'date';
		if (isDateTimeType(ft)) return 'datetime-local';
		return 'text';
	}

	// Dialog state (for codeunit results)
	let dialogOpen = $state(false);
	let dialogData = $state<DialogResult | null>(null);

	// Progress modal state (for codeunits with progress)
	let progressModalOpen = $state(false);
	let progressTitle = $state(t(MSG.PROCESSING));
	let progressValue = $state(0);
	let progressMessage = $state('');
	let progressError = $state('');
	let progressConfirmMode = $state(false);
	let progressConfirmMessage = $state('');
	let confirmResponseCallback: ((response: boolean) => void) | undefined = $state(undefined);

	// Input dialog state (for codeunits requesting user input)
	let progressInputMode = $state(false);
	let progressInputFields = $state<Array<{ name: string; label: string; type: string; required?: boolean; default?: string }>>([]);
	let inputResponseCallback: ((values: Record<string, string> | null) => void) | undefined = $state(undefined);

	// Rows shown: the loaded window of the list. Search and sort run on the server
	// (PageRenderer), over all records; editableRecords is the window's editable copy.
	const displayRecords = $derived(editableActive ? editableRecords : records);

	// More records exist after the loaded window
	const moreBelow = $derived(windowOffset + records.length < total);

	// Window position `index` is the last row of the whole list (new rows the user is
	// entering count too): moving down from it creates a new row (BC)
	function isLastRow(index: number): boolean {
		return index >= displayRecords.length - 1 && !moreBelow;
	}

	// A user-edited new row that is not inserted yet (Record Entry): the window must not
	// be replaced while it exists, or the user's input would be lost
	function hasUncommittedNewRow(): boolean {
		return editableActive && editableRecords.some((r) => r._isNew && !isEmptyNewRecord(r));
	}

	// Window loads run one at a time; a caller arriving during a load waits for it
	let windowLoad: Promise<void> | null = null;
	async function loadWindow(request: ListWindowRequest) {
		while (windowLoad) await windowLoad;
		windowLoad = (async () => {
			await onwindow?.(request);
			await tick();
		})();
		try {
			await windowLoad;
		} finally {
			windowLoad = null;
		}
	}

	// Load the window around absolute row `absRow` when it is outside the loaded rows or
	// within a page of an edge with more rows beyond. Returns the row's window position,
	// clamped to the loaded rows.
	async function windowIndexOf(absRow: number): Promise<number> {
		while (windowLoad) await windowLoad;
		const target = Math.max(0, Math.min(absRow, Math.max(total, displayRecords.length) - 1));
		const perPage = rowsPerPage();
		if (onwindow && needsShift(target - windowOffset, perPage, records.length, windowOffset, total) && !hasUncommittedNewRow()) {
			if (editableActive) cleanupEmptyNewRows();
			await loadWindow({ offset: windowOffsetFor(target, perPage, total), limit: windowSize(perPage) });
			if (editableActive) editableRecords = toEditableRecords();
		}
		return Math.max(0, Math.min(target - windowOffset, displayRecords.length - 1));
	}

	// Mouse wheel / scrollbar (navigation mode): near the top or bottom of the loaded
	// window, load the window around the rows in view and keep them in place (BC
	// "loading more"). Keyboard moves load windows through windowIndexOf.
	let scrollTimer: ReturnType<typeof setTimeout> | undefined;
	function handleTableScroll(event: Event) {
		if (!onwindow || !isNavigation) return;
		const container = event.currentTarget as HTMLElement;
		clearTimeout(scrollTimer);
		scrollTimer = setTimeout(() => shiftWindowForScroll(container), 80);
	}

	async function shiftWindowForScroll(container: HTMLElement) {
		const row = tableBodyElement?.querySelector('tr') as HTMLElement | null;
		if (!onwindow || windowLoad || pendingTarget !== null || !row || row.offsetHeight === 0) return;
		const rowHeight = row.offsetHeight;
		const perPage = rowsPerPage();
		const firstVisible = Math.floor(container.scrollTop / rowHeight);
		const nearTop = windowOffset > 0 && firstVisible < perPage;
		const nearBottom = moreBelow && firstVisible + perPage >= records.length - perPage;
		if (!nearTop && !nearBottom) return;

		const absFirst = windowOffset + firstVisible;
		const newOffset = windowOffsetFor(absFirst, perPage, total);
		if (newOffset === windowOffset) return;
		const absSelected = windowOffset + selectedIndex;
		const withinRow = container.scrollTop - firstVisible * rowHeight;

		await loadWindow({ offset: newOffset, limit: windowSize(perPage) });
		container.scrollTop = (absFirst - windowOffset) * rowHeight + withinRow;
		// Keep the selected record selected if it is still loaded; don't scroll to it
		const index = Math.max(0, Math.min(absSelected - windowOffset, displayRecords.length - 1));
		if (index !== selectedIndex) {
			skipAutoScroll = true;
			selectedIndex = index;
		}
	}

	// Search box: runs on the server over all records, shortly after the user stops typing
	let searchTimer: ReturnType<typeof setTimeout> | undefined;
	function handleSearchInput() {
		clearTimeout(searchTimer);
		searchTimer = setTimeout(() => applySearchAndSort({ search: searchQuery }), 300);
	}

	// New search or sort: leave cell editing (the cell is saved, as when focus leaves the
	// table), then load the first window of the new result
	async function applySearchAndSort(request: ListWindowRequest) {
		if (!onwindow) return;
		if (editableActive && !isNavigation && currentCellRow >= 0) {
			const record = displayRecords[currentCellRow];
			const field = visibleColumns()[currentCellCol];
			if (record) await handleCellBlur(record, currentCellRow, field?.source, true);
		}
		if (editableActive) exitToNavigation(false); // keep the focus in the search box
		await onwindow({ ...request, offset: 0, limit: windowSize(rowsPerPage()) });
		selectedIndex = records.length > 0 ? 0 : -1;
	}

	// Track list page element for focus
	let listPageElement: HTMLDivElement | null = null;

	// Auto-focus the page on mount and when records load (but not when modal is open)
	$effect(() => {
		if (listPageElement && isNavigation && records.length > 0 && !modalOpen) {
			setTimeout(() => {
				// Only when nothing else has the focus: never take it from a cell, the search
				// box or another control the user moved to
				const active = document.activeElement;
				const unfocused = !active || active === document.body;
				if (isNavigation && !modalOpen && unfocused) listPageElement?.focus();
			}, 100);
		}
	});

	// Window-level keyboard shortcuts (to capture before browser handles them)
	$effect(() => {
		function handleGlobalKeydown(event: KeyboardEvent) {
			// Skip if a modal or the Switch Company dialog (Ctrl+O) is open
			if (modalOpen || get(companySwitchOpen)) return;

			// Handle Escape specially - works even in input fields (NAV/BC behavior)
			if (event.key === 'Escape') {
				event.preventDefault();
				event.stopPropagation();
				if (isCellEditing) {
					// Revert edit, back to cell-selected
					exitEditingToCellSelected(true);
				} else if (isCellSelected) {
					// Back to navigation mode
					exitToNavigation();
				} else {
					// Navigate back to main menu
					goto('/');
				}
				return;
			}

			// Skip other shortcuts if we're in an input field (but not cell-editing inputs in the table)
			if (event.target instanceof HTMLInputElement || event.target instanceof HTMLTextAreaElement) return;

			const shortcutKey = getShortcutKey(event);

			// Check if this matches any action shortcut (normalize the action shortcut for comparison)
			const action = page.page.actions?.find(a => a.shortcut && normalizeShortcut(a.shortcut) === shortcutKey);
			if (action) {
				event.preventDefault();
				event.stopPropagation();
				handleAction(action.name);
			}
		}

		window.addEventListener('keydown', handleGlobalKeydown, true); // capture phase
		return () => window.removeEventListener('keydown', handleGlobalKeydown, true);
	});

	// Note: Focus is handled explicitly in transition functions, handleCellKeyDown(), and insertNewRow()
	// No auto-focus effect needed - it interferes with user clicks

	// Load customizations from localStorage on mount
	$effect(() => {
		const userId = $currentUser?.user_id || 'anonymous';
		columnCustomizations = loadPageCustomizations<Record<string, ItemCustomization>>(
			userId,
			page.page.id
		);
		columnWidths = loadColumnWidths(userId, page.page.id);
		showRowNumbers = loadRowNumbersPreference(userId, page.page.id);
	});

	// Track selected row index
	let selectedIndex = $state(-1);

	// Track table body element for scrolling
	let tableBodyElement: HTMLElement | null = null;

	// Auto-select first row when records load
	$effect(() => {
		if (records.length > 0 && selectedIndex === -1) {
			selectedIndex = 0;
		}
	});

	// Keep the selection on a loaded row when the window gets shorter (search, delete)
	$effect(() => {
		if (selectedIndex >= displayRecords.length) {
			selectedIndex = displayRecords.length > 0 ? displayRecords.length - 1 : -1;
		}
	});

	// Auto-scroll selected row into view (not after a window shift caused by the user
	// scrolling: that would scroll the list back to the selection)
	let skipAutoScroll = false;
	$effect(() => {
		if (selectedIndex >= 0 && tableBodyElement) {
			if (skipAutoScroll) {
				skipAutoScroll = false;
				return;
			}
			scrollRowIntoView(selectedIndex);
		}
	});

	// Scroll the list so the given displayed row is fully visible. The column header is
	// sticky, so the browser's scrollIntoView would park a row moving up *behind* the header
	// (e.g. the first record after PageUp); scroll the list container ourselves instead,
	// keeping rows below the header.
	function scrollRowIntoView(rowIndex: number) {
		const row = tableBodyElement?.querySelectorAll('tr')[rowIndex] as HTMLElement | undefined;
		if (!row) return;
		const container = row.closest('.table-container') as HTMLElement | null;
		if (!container) {
			row.scrollIntoView({ block: 'nearest' });
			return;
		}
		const headerHeight = (container.querySelector('thead') as HTMLElement | null)?.offsetHeight ?? 0;
		const box = container.getBoundingClientRect();
		const rowBox = row.getBoundingClientRect();
		const visibleTop = box.top + headerHeight;
		const visibleBottom = box.top + container.clientHeight; // excludes horizontal scrollbar
		if (rowBox.top < visibleTop) {
			container.scrollTop -= visibleTop - rowBox.top;
		} else if (rowBox.bottom > visibleBottom) {
			container.scrollTop += rowBox.bottom - visibleBottom;
		}
	}

	// Edit List mode state (BC-style full list editing)
	let currentCellRow = $state<number>(-1);
	let currentCellCol = $state<number>(-1);

	// Modal card state
	let modalOpen = $state(false);
	let modalCardPage = $state<PageDefinition | null>(null);
	let modalRecord = $state<Record<string, any>>({});
	let modalOriginalRecord = $state<Record<string, any>>({}); // Track original for change detection
	let modalIsNewRecord = $state(false);
	let modalCaptions = $state<Record<string, string>>({});
	let modalFieldTypes = $state<Record<string, string>>({}); // Field type metadata for modal card
	let modalOptions = $state<Record<string, Record<string, string>>>({}); // Option field values
	let modalLookups = $state<Record<string, LookupData>>({}); // Table relation lookup values
	let modalOptionsLoaded = $state(false); // Track if options have been loaded (prevent re-render during editing)
	let modalSaving = $state(false);
	let skipNextAutoSave = $state(false);
	let lastSaveToastTime = 0; // Debounce for save toast
	let modalHadChanges = $state(false); // Track if modal made any changes
	let modalInitialEditMode = $state(false); // Start modal in edit mode
	let modalRecordDeleted = $state(false); // Prevent saves after delete
	let modalSaveBlocked = $state(false); // Block editing due to save error
	let modalSaveBlockedMessage = $state(''); // Error message for blocked state

	// Get selected record. selectedIndex indexes the rows as displayed (after search and
	// column sort), so look the row up in displayRecords — never records[selectedIndex], which
	// is a different record whenever the list is searched or sorted. While cells are being
	// edited the displayed rows are editable copies: return the saved record with the same
	// persisted key. An uncommitted new row has no saved record, so it is no selection.
	const selectedRecord = $derived(
		findSelectedRecord(displayRecords, selectedIndex, records, editableActive, primaryKeyField, primaryKeyFieldsList)
	);

	// Handle running a codeunit
	// The backend decides whether to use progress based on codeunit.UsesProgress()
	async function handleRunObject(runObject: string) {
		let codeunitId: number;

		// Check if it's a field reference (format: "field:fieldname")
		if (runObject.startsWith('field:')) {
			const fieldName = runObject.substring(6); // Remove "field:" prefix
			if (!selectedRecord) {
				toast.error(t(ERR.NO_RECORD_SELECTED));
				return;
			}
			const fieldValue = selectedRecord[fieldName];
			if (fieldValue === undefined || fieldValue === null || fieldValue === 0) {
				toast.error(`No codeunit specified in ${fieldName}`);
				return;
			}
			codeunitId = typeof fieldValue === 'number' ? fieldValue : parseInt(String(fieldValue), 10);
		} else {
			// Parse the run_object string (format: "codeunit:ID")
			const [objectType, objectId] = runObject.split(':');
			if (objectType !== 'codeunit') {
				toast.error(t(ERR.UNKNOWN_OBJECT_TYPE, objectType));
				return;
			}
			codeunitId = parseInt(objectId, 10);
		}

		if (isNaN(codeunitId) || codeunitId <= 0) {
			toast.error(t(ERR.INVALID_CODEUNIT_ID));
			return;
		}

		// Show progress modal initially - backend decides if it's actually used
		progressModalOpen = true;
		progressTitle = t(MSG.PROCESSING);
		progressValue = 0;
		progressMessage = '';
		progressError = '';
		progressConfirmMode = false;
		progressConfirmMessage = '';
		confirmResponseCallback = undefined;
		progressInputMode = false;
		progressInputFields = [];
		inputResponseCallback = undefined;

		try {
			const result = await startJob(codeunitId, selectedRecord || {}, {
				onProgress: (event) => {
					// When receiving progress, exit confirm/input mode
					progressConfirmMode = false;
					progressInputMode = false;
					progressValue = event.value;
					if (event.message) {
						progressMessage = event.message;
					}
				},
				onComplete: (event) => {
					progressConfirmMode = false;
					progressInputMode = false;
					progressValue = 100;
					if (event.message) {
						progressMessage = event.message;
					}
				},
				onError: (err) => {
					progressConfirmMode = false;
					progressInputMode = false;
					progressError = err;
				},
				onConfirm: (event, respond) => {
					// Show confirm dialog
					progressConfirmMode = true;
					progressConfirmMessage = event.message;
					confirmResponseCallback = (response: boolean) => {
						// Reset confirm mode before sending response
						progressConfirmMode = false;
						progressConfirmMessage = '';
						respond(response);
					};
				},
				onInputRequest: (event, respond) => {
					// Show input dialog
					progressInputMode = true;
					progressInputFields = (event.data?.fields as typeof progressInputFields) || [];
					if (event.message) progressTitle = event.message;
					inputResponseCallback = (values: Record<string, string> | null) => {
						progressInputMode = false;
						progressInputFields = [];
						respond(values);
					};
				},
				onSyncResult: (syncResult) => {
					// Sync result - close progress modal immediately and show dialog/toast
					progressModalOpen = false;
					if (syncResult.success) {
						if (syncResult.dialog) {
							dialogData = syncResult.dialog;
							dialogOpen = true;
						} else {
							toast.success(syncResult.message || t(MSG.CODEUNIT_SUCCESS));
						}
					} else {
						toast.error(syncResult.message || t(ERR.CODEUNIT_FAILED));
					}
				}
			});

			// For async jobs, keep modal open briefly to show 100%
			if (result && 'job_id' in result) {
				await new Promise((resolve) => setTimeout(resolve, 500));
				progressModalOpen = false;
				if (!progressError) {
					toast.success(t(MSG.JOB_COMPLETED));
				}
				// Refresh list data to reflect changes (e.g. new entries, status updates)
				onaction?.('Refresh');
			}
		} catch (err) {
			const errMsg = err instanceof Error ? err.message : 'Unknown error';
			progressModalOpen = false;
			toast.error(errMsg);
		}
	}

	// Handle action clicks
	async function handleAction(actionName: string) {
		// Check if the action has a run_object
		const action = page.page.actions?.find(a => a.name === actionName);
		if (action?.run_object) {
			await handleRunObject(action.run_object);
			return;
		}

		// Handle Edit action - open card page in edit mode
		if (actionName === 'Edit') {
			if (page.page.card_page_id && selectedRecord) {
				if (page.page.modal_card) {
					// Open as modal card in edit mode
					await openModalCard(selectedRecord, true);
				} else {
					// Navigate to card page
					onaction?.(actionName, selectedRecord);
				}
				return;
			} else if (page.page.editable) {
				// Toggle inline edit mode
				if (isNavigation) {
					if (selectedIndex >= 0) enterCellSelected(selectedIndex, 0);
				} else {
					exitToNavigation();
				}
				return;
			}
		}

		// Handle "New" action - prioritize opening card page if available
		if (actionName === 'New') {
			if (page.page.card_page_id) {
				if (page.page.modal_card) {
					// Open as modal card
					await openModalCard({});
				} else {
					// Navigate to card page
					onaction?.(actionName, undefined);
				}
				return;
			} else if (page.page.editable) {
				// Inline editing (no card page available)
				handleNew();
				return;
			}
		}

		// Handle Delete action
		if (actionName === 'Delete') {
			if (page.page.editable) {
				handleDelete();
				return;
			}
		}

		onaction?.(actionName, selectedRecord || undefined);
	}

	// Editable copy of the records. _key holds each record's persisted primary key, so a
	// record whose key field the user edits is still addressed by its stored key
	// _saved holds the values last saved, so a row the user did not change is never written.
	function toEditableRecords(): Array<Record<string, any>> {
		return records.map(r => ({ ...r, _key: getRecordId(r, primaryKeyField, primaryKeyFieldsList), _saved: deepCopy(r) }));
	}

	// Ctrl+Insert (new line) or Alt+N (New) inserts a new row (BC). Not Ctrl+N: browsers
	// reserve it for a new window and never pass it to the page.
	function isNewRowShortcut(event: KeyboardEvent): boolean {
		const shortcutKey = getShortcutKey(event);
		return shortcutKey === 'Ctrl+Insert' || shortcutKey === commonShortcuts.NEW;
	}

	// Row index convention: every row index in the editing code (currentCellRow, selectedIndex,
	// rowIndex, prevRow, ...) is a position in displayRecords — the rows as shown, after search
	// and column sort. editableRecords holds the same row objects in their underlying order:
	// insert and remove rows there by identity, never by a displayed index, then map back with
	// displayIndexOf().
	function displayIndexOf(row: Record<string, any>): number {
		return displayRecords.findIndex((r) => r === row || (!!row._tempId && r._tempId === row._tempId));
	}

	// Insert a row into editableRecords right after the given displayed row (at the end if
	// there is none) and return its displayed index
	function insertRowAfter(newRow: Record<string, any>, anchor: Record<string, any> | undefined): number {
		const anchorPos = anchor ? editableRecords.indexOf(anchor) : -1;
		const insertPos = anchorPos >= 0 ? anchorPos + 1 : editableRecords.length;
		editableRecords = [...editableRecords.slice(0, insertPos), newRow, ...editableRecords.slice(insertPos)];
		return displayIndexOf(newRow);
	}

	// Handle new record - insert blank row below current selection
	function handleNew() {
		// Initialize editable records if not already active
		if (!editableActive) {
			editableRecords = toEditableRecords();
			editableActive = true;
		}

		// If an empty new row already exists, just focus it instead of creating another
		const existingNewRowIndex = displayRecords.findIndex(r => isEmptyNewRecord(r));
		if (existingNewRowIndex >= 0) {
			selectedIndex = existingNewRowIndex;
			currentCellRow = existingNewRowIndex;
			currentCellCol = 0;
			cellState = 'cell-selected';
			focusCellSelectedElement(currentCellRow, currentCellCol);
			return;
		}

		const newRecord = createNewRecord();

		// Insert below the currently selected row (or at end if none selected)
		const insertIndex = insertRowAfter(newRecord, selectedIndex >= 0 ? displayRecords[selectedIndex] : undefined);

		// Update selection to the new row
		selectedIndex = insertIndex;

		// Focus the first cell of the new row in cell-selected mode
		currentCellRow = insertIndex;
		currentCellCol = 0;
		cellState = 'cell-selected';
		focusCellSelectedElement(currentCellRow, currentCellCol);
	}

	// --- 3-State Transition Functions ---

	// Enter cell-selected state at the given cell
	function enterCellSelected(row: number, col: number) {
		const cols = visibleColumns();
		if (row < 0 || col < 0 || col >= cols.length) return;

		// Initialize editable records if not active
		if (!editableActive) {
			editableRecords = toEditableRecords();
			editableActive = true;
		}

		if (row >= displayRecords.length) return;

		currentCellRow = row;
		currentCellCol = col;
		selectedIndex = row;
		cellState = 'cell-selected';
		cellEditSnapshot = undefined;

		// Focus the cell-selected div
		focusCellSelectedElement(row, col);
	}

	// Enter cell-editing state from cell-selected
	function enterCellEditing(clearContent: boolean = false, typedChar?: string) {
		if (cellState !== 'cell-selected') return;

		const cols = visibleColumns();
		const field = cols[currentCellCol];
		const record = displayRecords[currentCellRow];
		if (!field || !record) return;

		// Snapshot current value for Escape revert
		cellEditSnapshot = record[field.source];

		if (clearContent) {
			record[field.source] = typedChar ?? '';
			editableRecords = [...editableRecords];
		}

		cellState = 'cell-editing';
		// Cursor at the end of the text, never select-all (NAV/BC): F2/F8/double-click keep the
		// content to be amended (PAY-TERM01 -> Backspace -> PAY-TERM02); a typed character
		// starts the new content with the cursor after it
		focusCell(currentCellRow, currentCellCol, false);
	}

	// Exit cell-editing back to cell-selected
	function exitEditingToCellSelected(revert: boolean) {
		if (cellState !== 'cell-editing') return;

		if (revert && cellEditSnapshot !== undefined) {
			const cols = visibleColumns();
			const field = cols[currentCellCol];
			if (field && displayRecords[currentCellRow]) {
				displayRecords[currentCellRow][field.source] = cellEditSnapshot;
				editableRecords = [...editableRecords];
			}
		}

		cellState = 'cell-selected';
		cellEditSnapshot = undefined;
		focusCellSelectedElement(currentCellRow, currentCellCol);
	}

	// Exit to navigation mode. focusPage = false when the user moved the focus elsewhere
	// (e.g. clicked the search box): keep it there.
	function exitToNavigation(focusPage: boolean = true) {
		// Clean up empty new rows
		cleanupEmptyNewRows();

		cellState = 'navigation';
		currentCellRow = -1;
		currentCellCol = -1;
		cellEditSnapshot = undefined;
		editableRecords = [];
		editableActive = false;

		if (focusPage) listPageElement?.focus();
	}

	// Confirm current cell value and move to target cell (enters cell-selected at target)
	async function confirmAndMoveTo(targetRow: number, targetCol: number) {
		const cols = visibleColumns();
		const prevRow = currentCellRow;
		const prevCol = currentCellCol;
		const record = displayRecords[prevRow];
		// Remember the target row itself: saving can reorder the displayed rows (column sort)
		const targetRecord = displayRecords[targetRow];

		// Immediately transition state to prevent blur handler from interfering
		cellState = 'cell-selected';
		cellEditSnapshot = undefined;

		// Apply Code uppercase and save the previous cell
		if (record) {
			const field = cols[prevCol];
			if (field && fieldTypes[field.source] === 'code' && typeof record[field.source] === 'string') {
				record[field.source] = record[field.source].toUpperCase();
			}
			await handleCellBlur(record, prevRow, field?.source, targetRecord !== record);
		}

		// Clean up empty new row if leaving it
		if (record && isEmptyNewRecord(record) && targetRecord !== record) {
			editableRecords = editableRecords.filter((r) => r !== record);
		}

		// Find the target row again (by identity), then clamp to the valid range
		let adjustedTargetRow = targetRecord ? displayIndexOf(targetRecord) : targetRow;
		if (adjustedTargetRow < 0) adjustedTargetRow = targetRow;
		// Outside the loaded window (or near its edge): load the window around it first
		if (onwindow && needsShift(adjustedTargetRow, rowsPerPage(), records.length, windowOffset, total)) {
			const index = await windowIndexOf(windowOffset + adjustedTargetRow);
			if (index >= 0) adjustedTargetRow = index;
		}
		adjustedTargetRow = Math.max(0, Math.min(adjustedTargetRow, displayRecords.length - 1));
		const adjustedTargetCol = Math.max(0, Math.min(targetCol, cols.length - 1));

		// Enter cell-selected at target
		enterCellSelected(adjustedTargetRow, adjustedTargetCol);
	}


	// Track saving state to prevent concurrent saves
	let isSaving = $state(false);
	// Saves confirmed while another save runs, in order. A queue, not a single slot: with
	// one slot a third quick edit overwrote the waiting one and that edit was never saved.
	let pendingSaves: Array<{ record: Record<string, any>; rowIndex: number; fieldName?: string; leavingRow?: boolean }> = [];

	// Header save indicator ("Saving..." / "✓ Saved"), the only feedback that a row committed
	let saveState = $state<'idle' | 'saving' | 'saved'>('idle');
	let savedTimeout: ReturnType<typeof setTimeout> | undefined;

	// Auto-save when a cell value is confirmed (BC/NAV record entry):
	// - Existing record: MODIFY.
	// - New record: a field the user changed is first validated server-side together with the
	//   whole in-progress record, so its OnValidate trigger can fill sibling fields. The record
	//   is then INSERTed as soon as it has a user edit and values for its required primary key
	//   fields — while the cursor is still on the row, not when the row is left. Until then it
	//   stays uncommitted; an untouched new row is discarded when the user leaves it.
	// - Pages with delayed_insert (BC DelayedInsert) insert only when leavingRow is true: the
	//   user moved to another row, past the last row, or out of the table.
	async function handleCellBlur(record: Record<string, any>, rowIndex: number, fieldName?: string, leavingRow: boolean = false) {
		if (!page || !editableActive) return;

		// An existing record the user did not change since it was last saved: nothing to
		// validate or save (BC/NAV modifies only changed records). Moving through the cells
		// of a list used to MODIFY every row passed — a write plus a list reload per key.
		if (record._isNew !== true && record._saved && !hasUserEdits(record, record._saved)) return;

		// If already saving, queue this save for later (must check before async validation)
		if (isSaving) {
			pendingSaves.push({ record, rowIndex, fieldName, leavingRow });
			return;
		}

		isSaving = true;

		const isNew = record._isNew === true;

		if (fieldName && isNew) {
			// Validate the user's change with the full record; merge sibling fields the trigger set
			if (!sameFieldValue(record[fieldName], record._pristine?.[fieldName])) {
				try {
					const result = await api.validateField(page.page.source_table, fieldName, record[fieldName], stripInternalFields(record));
					if (!result.valid) {
						toast.error(result.error || `Invalid value for ${fieldName}`);
						record[fieldName] = record._pristine?.[fieldName] ?? '';
						isSaving = false;
						return;
					}
					if (result.record) {
						Object.assign(record, result.record);
					}
				} catch {
					// Validation endpoint failed — skip validation, don't block
				}
			}
		} else if (fieldName) {
			// Validate table_relation fields: check if the value exists in the related table
			// Skip validation for fields using LookupDropdown (it validates internally)
			const fieldDef = page.page.layout.repeater?.fields?.find(f => f.source === fieldName);
			const hasAdvancedLookup = lookups[fieldName]?.columns && lookups[fieldName]?.rows?.length;
			const value = record[fieldName];
			if (fieldDef?.table_relation && !hasAdvancedLookup && value && value !== '') {
				try {
					const result = await api.validateField(page.page.source_table, fieldName, value);
					if (!result.valid) {
						toast.error(result.error || `Invalid value for ${fieldName}`);
						record[fieldName] = '';
						isSaving = false;
						return;
					}
				} catch {
					// Validation endpoint failed — skip validation, don't block
				}
			}
		}
		try {
			// Address an existing record by its persisted key: the user may have edited a
			// primary key field, which the backend then renames (BC/NAV Rename)
			const recordId = record._key ?? getRecordId(record, primaryKeyField, primaryKeyFieldsList);

			if (isNew) {
				const pkFields = primaryKeyFieldsList.map(pk => ({
					source: pk,
					required: page.page.layout.repeater?.fields?.find(f => f.source === pk)?.required
				}));
				// DelayedInsert pages: keep the row uncommitted until the user leaves it
				if (page.page.delayed_insert && !leavingRow) {
					return;
				}
				if (!shouldInsertNewRecord(record, record._pristine ?? {}, pkFields)) {
					return;
				}

				const tempId = record._tempId;
				saveState = 'saving';
				const savedRecord = await api.insertRecord(page.page.source_table, stripInternalFields(record));
				// Update the row in place, keeping _tempId so Svelte's keyed each stays stable.
				// Find it by _tempId: an await can race with rows being removed or with
				// exitToNavigation() clearing editableRecords.
				const savedIndex = editableRecords.findIndex(r => r._tempId === tempId);
				if (savedRecord && savedIndex >= 0) {
					Object.assign(editableRecords[savedIndex], savedRecord);
					delete editableRecords[savedIndex]._isNew;
					delete editableRecords[savedIndex]._pristine;
					editableRecords[savedIndex]._key = getRecordId(savedRecord, primaryKeyField, primaryKeyFieldsList);
					editableRecords[savedIndex]._saved = deepCopy(stripInternalFields(editableRecords[savedIndex]));
				}
				markSaved();
				// Trigger parent update if callback exists
				if (onsave) {
					await onsave(savedRecord, true);
				}
			} else if (recordId !== undefined) {
				// Existing record - update it (recordId may be "" for a blank-PK setup record)
				const _tempId = record._tempId;
				saveState = 'saving';
				const savedRecord = await api.modifyRecord(page.page.source_table, recordId, stripInternalFields(record));
				// Update the row object itself (not by index: the displayed order can differ from
				// editableRecords, and an await can race with exitToNavigation() clearing it;
				// updating a detached row is harmless). Keep any _tempId.
				if (savedRecord) {
					Object.assign(record, savedRecord);
					if (_tempId) record._tempId = _tempId;
					record._key = getRecordId(savedRecord, primaryKeyField, primaryKeyFieldsList);
					record._saved = deepCopy(stripInternalFields(record));
				}
				markSaved();
				// Trigger parent update if callback exists
				if (onsave) {
					await onsave(savedRecord, false);
				}
			}
		} catch (err) {
			console.error('Error saving cell:', err);
			saveState = 'idle';
			const message = err instanceof Error ? err.message : t(ERR.FAILED_SAVE_RECORD);
			toast.error(message);
			// Revert the cell to its last saved values
			const originalRecord = record._saved ?? records.find(r => getRecordId(r, primaryKeyField, primaryKeyFieldsList) === (record._key ?? getRecordId(record, primaryKeyField, primaryKeyFieldsList)));
			if (!isNew && originalRecord) {
				// Existing record - revert the row to its saved values (temp flags are kept)
				Object.assign(record, deepCopy(originalRecord));
			}
			// A new record that failed to insert stays uncommitted and editable;
			// the insert is retried on the next confirmed cell
		} finally {
			isSaving = false;
			// Process the next queued save
			const next = pendingSaves.shift();
			if (next) {
				// Use setTimeout to avoid stack overflow
				setTimeout(() => handleCellBlur(next.record, next.rowIndex, next.fieldName, next.leavingRow), 0);
			}
		}
	}

	// Show the "Saved" indicator briefly after an insert/modify completes
	function markSaved() {
		saveState = 'saved';
		if (savedTimeout) clearTimeout(savedTimeout);
		savedTimeout = setTimeout(() => {
			saveState = 'idle';
		}, 1500);
	}

	// Create a new, not yet inserted row (BC/NAV OnNewRecord). It starts blank and is filled
	// with the table's defaults from the init endpoint when they arrive; if that fails it stays
	// blank so data entry is never blocked. The initial values are kept in _pristine: only
	// changes away from them count as user edits, so defaults never trigger an insert.
	function createNewRecord(): Record<string, any> {
		const tempId = `new-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
		// All repeater fields start as empty strings so composite PK fields are always defined
		const blank: Record<string, any> = {};
		for (const f of page.page.layout.repeater?.fields ?? []) {
			blank[f.source] = '';
		}
		void applyNewRecordDefaults(tempId, blank);
		return { ...blank, _isNew: true, _tempId: tempId, _pristine: { ...blank } };
	}

	async function applyNewRecordDefaults(tempId: string, blank: Record<string, any>) {
		let defaults: Record<string, any>;
		try {
			defaults = await api.initRecord(page.page.source_table);
		} catch (err) {
			console.error('Failed to initialize new record, keeping it blank:', err);
			return;
		}
		const row = editableRecords.find(r => r._tempId === tempId);
		if (!row || row._isNew !== true || !defaults) return;
		for (const [key, value] of Object.entries(defaults)) {
			// Never overwrite what the user already typed while the defaults were loading
			if (sameFieldValue(row[key], blank[key])) {
				row[key] = value;
			}
		}
		row._pristine = { ...blank, ...defaults };
		editableRecords = [...editableRecords];
	}

	// Check if a record is an untouched new record (marked as new, no user edits beyond its defaults)
	function isEmptyNewRecord(record: Record<string, any>): boolean {
		return record._isNew === true && !hasUserEdits(record, record._pristine ?? {});
	}

	// Remove empty new rows from editableRecords
	function cleanupEmptyNewRows() {
		if (!editableRecords.some((r) => isEmptyNewRecord(r))) return;
		// Keep the current row's identity so its displayed index can be found again
		const current = displayRecords[currentCellRow];
		editableRecords = editableRecords.filter((r) => !isEmptyNewRecord(r));
		if (current) {
			const index = displayIndexOf(current);
			currentCellRow = index >= 0 ? index : Math.min(currentCellRow, displayRecords.length - 1);
		}
	}

	// Confirm the cell on the last row and open a new blank row below it — unless the row
	// is itself an untouched new row (no chain of blank rows)
	function confirmAndAddRow(rowIndex: number, colIndex: number) {
		const record = displayRecords[rowIndex];
		if (!record || isEmptyNewRecord(record)) return;
		const field = visibleColumns()[colIndex];
		if (field && fieldTypes[field.source] === 'code' && typeof record[field.source] === 'string') {
			record[field.source] = record[field.source].toUpperCase();
		}
		handleCellBlur(record, rowIndex, field?.source, true);
		insertNewRow(true);
	}

	// Insert a new row at cursor position
	function insertNewRow(atEnd: boolean = false) {
		if (!editableActive) return;

		// Don't create a new row if we're already on an empty new row
		const currentRecord = currentCellRow >= 0 ? displayRecords[currentCellRow] : undefined;
		if (currentRecord && isEmptyNewRecord(currentRecord)) {
			// Already on an empty new row, just focus it
			currentCellCol = 0;
			cellState = 'cell-selected';
			focusCellSelectedElement(currentCellRow, currentCellCol);
			return;
		}

		// Clean up any other empty new rows first
		cleanupEmptyNewRows();

		const newRecord = createNewRecord();

		// Insert at the end, or above the current row (by identity in editableRecords)
		const current = currentCellRow >= 0 ? displayRecords[currentCellRow] : undefined;
		const currentPos = !atEnd && current ? editableRecords.indexOf(current) : -1;
		const insertPos = currentPos >= 0 ? currentPos : editableRecords.length;
		editableRecords = [...editableRecords.slice(0, insertPos), newRecord, ...editableRecords.slice(insertPos)];

		// Focus the first cell of the new row in cell-selected mode
		currentCellRow = displayIndexOf(newRecord);
		selectedIndex = currentCellRow;
		currentCellCol = 0;
		cellState = 'cell-selected';
		focusCellSelectedElement(currentCellRow, currentCellCol);
	}

	// Handle keyboard in cell-selected mode
	function handleCellSelectedKeyDown(event: KeyboardEvent, rowIndex: number, colIndex: number) {
		const cols = visibleColumns();
		const record = displayRecords[rowIndex];
		if (!record) return;

		const field = cols[colIndex];
		const isBoolean = typeof record[field.source] === 'boolean' || fieldTypes[field.source] === 'bool';
		const hasLookup = lookups[field.source]?.columns || lookups[field.source]?.simple;
		const hasOptions = !!options[field.source];

		// Alt+ArrowDown opens the lookup/option dropdown
		if (event.key === 'ArrowDown' && event.altKey && (hasLookup || hasOptions)) {
			event.preventDefault();
			enterCellEditing(false);
			return;
		}

		// Ctrl+Insert or Alt+N to insert new row
		if (isNewRowShortcut(event)) {
			event.preventDefault();
			insertNewRow();
			return;
		}

		// Ctrl+C: Copy cell value to clipboard
		if (event.key === 'c' && event.ctrlKey && !event.shiftKey && !event.altKey) {
			event.preventDefault();
			const value = record[field.source];
			const text = formatCellValue(value, field.source);
			navigator.clipboard.writeText(text);
			return;
		}

		// Ctrl+V: Paste from clipboard into cell
		if (event.key === 'v' && event.ctrlKey && !event.shiftKey && !event.altKey) {
			event.preventDefault();
			if (field.editable === false) return;
			if (isBoolean) return;
			navigator.clipboard.readText().then((text) => {
				record[field.source] = text;
				editableRecords = [...editableRecords];
				enterCellEditing(false);
			});
			return;
		}

		switch (event.key) {
			case 'ArrowUp':
				event.preventDefault();
				if (rowIndex > 0 || windowOffset > 0) {
					confirmAndMoveTo(rowIndex - 1, colIndex);
				}
				break;
			case 'ArrowDown':
				event.preventDefault();
				if (!isLastRow(rowIndex)) {
					confirmAndMoveTo(rowIndex + 1, colIndex);
				} else if (!isEmptyNewRecord(record)) {
					// Past the last row: save current cell, then create a new row (BC behavior)
					handleCellBlur(record, rowIndex, field.source, true);
					insertNewRow(true);
				}
				break;
			case 'PageDown':
				// Confirm value + move a page down in the same column
				event.preventDefault();
				confirmAndMoveTo(rowIndex + rowsPerPage(), colIndex);
				break;
			case 'PageUp':
				event.preventDefault();
				confirmAndMoveTo(rowIndex - rowsPerPage(), colIndex);
				break;
			case 'ArrowLeft':
				event.preventDefault();
				if (colIndex > 0) {
					// No save, just move selection
					enterCellSelected(rowIndex, colIndex - 1);
				}
				break;
			case 'ArrowRight':
				event.preventDefault();
				if (colIndex < cols.length - 1) {
					// No save, just move selection
					enterCellSelected(rowIndex, colIndex + 1);
				}
				break;
			case 'Tab':
				event.preventDefault();
				if (event.shiftKey) {
					// Move left, wrap to previous row
					if (colIndex > 0) {
						confirmAndMoveTo(rowIndex, colIndex - 1);
					} else if (rowIndex > 0 || windowOffset > 0) {
						confirmAndMoveTo(rowIndex - 1, cols.length - 1);
					}
				} else {
					// Move right, wrap to next row
					if (colIndex < cols.length - 1) {
						confirmAndMoveTo(rowIndex, colIndex + 1);
					} else if (!isLastRow(rowIndex)) {
						confirmAndMoveTo(rowIndex + 1, 0);
					} else {
						// Last column of the last row: save and open a new blank row (BC)
						confirmAndAddRow(rowIndex, colIndex);
					}
				}
				break;
			case 'Enter':
				event.preventDefault();
				if (isBoolean) {
					// Toggle checkbox + move down
					record[field.source] = !record[field.source];
					editableRecords = [...editableRecords];
				}
				if (!isLastRow(rowIndex)) {
					confirmAndMoveTo(rowIndex + 1, colIndex);
				} else if (!isEmptyNewRecord(record)) {
					// Save current cell, then create new row at end
					handleCellBlur(record, rowIndex, field.source, true);
					insertNewRow(true);
				}
				break;
			case 'F2':
				event.preventDefault();
				if (!isBoolean) {
					enterCellEditing(false); // cursor at end, content preserved
				}
				break;
			case 'Delete':
				event.preventDefault();
				if (!isBoolean) {
					record[field.source] = '';
					editableRecords = [...editableRecords];
				}
				break;
			case 'Backspace':
				event.preventDefault();
				if (!isBoolean) {
					enterCellEditing(true); // clear + edit
				}
				break;
			case ' ':
				// Space on boolean: toggle checkbox
				if (isBoolean) {
					event.preventDefault();
					record[field.source] = !record[field.source];
					editableRecords = [...editableRecords];
					handleCellBlur(record, rowIndex, field.source);
				}
				break;
			case 'F8':
				// Copy from cell above, then edit it with the cursor at the end (NAV/BC)
				event.preventDefault();
				if (rowIndex > 0 && field.editable !== false) {
					const aboveRecord = displayRecords[rowIndex - 1];
					const valueBefore = record[field.source];
					record[field.source] = aboveRecord[field.source];
					editableRecords = [...editableRecords];
					if (!isBoolean) {
						enterCellEditing(false); // keeps content, cursor at end
						cellEditSnapshot = valueBefore; // Escape reverts to the value before F8
					}
				}
				break;
			default:
				// Printable character: clear + enter editing with typed char
				if (!isBoolean && event.key.length === 1 && !event.ctrlKey && !event.altKey && !event.metaKey) {
					event.preventDefault();
					enterCellEditing(true, event.key);
				}
				break;
		}
	}

	// Handle keyboard navigation in cell-editing mode
	function handleCellKeyDown(event: KeyboardEvent, rowIndex: number, colIndex: number) {
		const cols = visibleColumns();

		// Ctrl+Insert or Alt+N to insert new row
		if (isNewRowShortcut(event)) {
			event.preventDefault();
			insertNewRow();
			return;
		}

		// Skip arrow/key navigation for <select> elements — let browser handle option cycling
		const isSelectElement = event.target instanceof HTMLSelectElement;

		switch (event.key) {
			case 'ArrowUp':
				{
					if (isSelectElement) break; // Let <select> handle its own arrow navigation
					const input = event.target as HTMLInputElement;
					const isTextInput = input.type === 'text' || input.type === 'number';
					let shouldNavigate = !isTextInput;

					if (isTextInput) {
						const textLength = input.value?.length || 0;
						const allSelected = input.selectionStart === 0 && input.selectionEnd === textLength && textLength > 0;
						const atStart = input.selectionStart === 0;
						shouldNavigate = allSelected || !!atStart;
					}

					if (shouldNavigate && (rowIndex > 0 || windowOffset > 0)) {
						event.preventDefault();
						confirmAndMoveTo(rowIndex - 1, colIndex);
					}
				}
				break;
			case 'ArrowDown':
				{
					if (isSelectElement) break; // Let <select> handle its own arrow navigation
					const input = event.target as HTMLInputElement;
					const isTextInput = input.type === 'text' || input.type === 'number';
					let shouldNavigate = !isTextInput;

					if (isTextInput) {
						const textLength = input.value?.length || 0;
						const allSelected = input.selectionStart === 0 && input.selectionEnd === textLength && textLength > 0;
						const atEnd = input.selectionStart === textLength;
						shouldNavigate = allSelected || !!atEnd;
					}

					if (shouldNavigate) {
						event.preventDefault();
						if (!isLastRow(rowIndex)) {
							confirmAndMoveTo(rowIndex + 1, colIndex);
						} else {
							const currentRecord = displayRecords[rowIndex];
							if (!isEmptyNewRecord(currentRecord)) {
								const field = cols[colIndex];
								if (field && fieldTypes[field.source] === 'code' && typeof currentRecord[field.source] === 'string') {
									currentRecord[field.source] = currentRecord[field.source].toUpperCase();
								}
								handleCellBlur(currentRecord, rowIndex, field?.source, true);
								insertNewRow(true);
							}
						}
					}
				}
				break;
			case 'ArrowLeft':
				{
					if (isSelectElement) break; // Let <select> handle its own navigation
					const input = event.target as HTMLInputElement;
					const isTextInput = input.type === 'text' || input.type === 'number';
					let shouldNavigate = !isTextInput;

					if (isTextInput) {
						const textLength = input.value?.length || 0;
						const allSelected = input.selectionStart === 0 && input.selectionEnd === textLength && textLength > 0;
						const atStart = input.selectionStart === 0 && input.selectionEnd === 0;
						shouldNavigate = allSelected || atStart;
					}

					if (shouldNavigate && colIndex > 0) {
						event.preventDefault();
						confirmAndMoveTo(rowIndex, colIndex - 1);
					}
				}
				break;
			case 'ArrowRight':
				{
					if (isSelectElement) break; // Let <select> handle its own navigation
					const input = event.target as HTMLInputElement;
					const isTextInput = input.type === 'text' || input.type === 'number';
					let shouldNavigate = !isTextInput;

					if (isTextInput) {
						const textLength = input.value?.length || 0;
						const allSelected = input.selectionStart === 0 && input.selectionEnd === textLength && textLength > 0;
						const atEnd = input.selectionStart === textLength && input.selectionEnd === textLength;
						shouldNavigate = allSelected || atEnd;
					}

					if (shouldNavigate && colIndex < cols.length - 1) {
						event.preventDefault();
						confirmAndMoveTo(rowIndex, colIndex + 1);
					}
				}
				break;
			case 'Tab':
				event.preventDefault();
				if (event.shiftKey) {
					if (colIndex > 0) {
						confirmAndMoveTo(rowIndex, colIndex - 1);
					} else if (rowIndex > 0 || windowOffset > 0) {
						confirmAndMoveTo(rowIndex - 1, cols.length - 1);
					}
				} else {
					if (colIndex < cols.length - 1) {
						confirmAndMoveTo(rowIndex, colIndex + 1);
					} else if (!isLastRow(rowIndex)) {
						confirmAndMoveTo(rowIndex + 1, 0);
					} else {
						// Last column of the last row: save and open a new blank row (BC)
						confirmAndAddRow(rowIndex, colIndex);
					}
				}
				break;
			case 'PageDown':
				if (isSelectElement) break;
				event.preventDefault();
				confirmAndMoveTo(rowIndex + rowsPerPage(), colIndex);
				break;
			case 'PageUp':
				if (isSelectElement) break;
				event.preventDefault();
				confirmAndMoveTo(rowIndex - rowsPerPage(), colIndex);
				break;
			case 'F2':
				// Exit cell-editing → return to cell-selected (keep current value)
				event.preventDefault();
				exitEditingToCellSelected(false);
				break;
			case 'F8':
				// F8 copies value from the cell above (NAV/BC behavior)
				{
					event.preventDefault();
					const field = cols[colIndex];
					if (rowIndex > 0 && field && field.editable !== false) {
						const aboveRecord = displayRecords[rowIndex - 1];
						const currentRecord = displayRecords[rowIndex];
						const valueToCopy = aboveRecord[field.source];

						// Copy the value
						currentRecord[field.source] = valueToCopy;

						// Update the input element directly for immediate visual feedback
						const input = event.target as HTMLInputElement;
						if (input.type === 'checkbox') {
							input.checked = !!valueToCopy;
						} else {
							input.value = valueToCopy ?? '';
						}

						// Trigger reactivity, then put the cursor at the end of the copied text
						editableRecords = [...editableRecords];
						if (input.type !== 'checkbox') focusCell(rowIndex, colIndex, false);
					}
				}
				break;
			case 'Enter':
				event.preventDefault();
				if (!isLastRow(rowIndex)) {
					confirmAndMoveTo(rowIndex + 1, colIndex);
				} else {
					const currentRecord = displayRecords[rowIndex];
					if (!isEmptyNewRecord(currentRecord)) {
						// Save current cell, then create new row
						const field = cols[colIndex];
						if (field && fieldTypes[field.source] === 'code' && typeof currentRecord[field.source] === 'string') {
							currentRecord[field.source] = currentRecord[field.source].toUpperCase();
						}
						handleCellBlur(currentRecord, rowIndex, field?.source, true);
						insertNewRow(true);
					}
				}
				break;
		}
	}

	// Handle keyboard on LookupDropdown wrapper div (bubbles up from LookupDropdown input)
	// Only intercepts Tab/Enter for cell navigation — LookupDropdown handles Arrow/Escape/F4 internally
	function handleLookupCellKeyDown(event: KeyboardEvent, rowIndex: number, colIndex: number) {
		// Skip keys already handled by LookupDropdown (Arrow keys, Escape with open dropdown)
		if (event.defaultPrevented) return;

		const cols = visibleColumns();

		switch (event.key) {
			case 'Tab':
				event.preventDefault();
				if (event.shiftKey) {
					if (colIndex > 0) {
						confirmAndMoveTo(rowIndex, colIndex - 1);
					} else if (rowIndex > 0 || windowOffset > 0) {
						confirmAndMoveTo(rowIndex - 1, cols.length - 1);
					}
				} else {
					if (colIndex < cols.length - 1) {
						confirmAndMoveTo(rowIndex, colIndex + 1);
					} else if (!isLastRow(rowIndex)) {
						confirmAndMoveTo(rowIndex + 1, 0);
					} else {
						// Last column of the last row: save and open a new blank row (BC)
						confirmAndAddRow(rowIndex, colIndex);
					}
				}
				break;
			case 'Enter':
				// Only handle Enter when dropdown is closed (LookupDropdown preventDefault's Enter when open)
				event.preventDefault();
				if (!isLastRow(rowIndex)) {
					confirmAndMoveTo(rowIndex + 1, colIndex);
				} else {
					const currentRecord = displayRecords[rowIndex];
					if (!isEmptyNewRecord(currentRecord)) {
						const field = cols[colIndex];
						if (field && fieldTypes[field.source] === 'code' && typeof currentRecord[field.source] === 'string') {
							currentRecord[field.source] = currentRecord[field.source].toUpperCase();
						}
						handleCellBlur(currentRecord, rowIndex, field?.source, true);
						insertNewRow(true);
					}
				}
				break;
			case 'Escape':
				// LookupDropdown handles Escape when dropdown is open (preventDefault).
				// When dropdown is closed, Escape bubbles here — revert and exit editing.
				exitEditingToCellSelected(true);
				break;
			case 'F2':
				event.preventDefault();
				exitEditingToCellSelected(false);
				break;
		}
	}

	// Focus a specific cell input (for cell-editing mode)
	// Put the text cursor at the end of the input's value. Date/time inputs have no text
	// selection (setSelectionRange throws on them), so they are just left focused.
	function placeCursorAtEnd(input: HTMLInputElement) {
		const len = input.value?.length || 0;
		try {
			input.setSelectionRange(len, len);
		} catch {
			// input type without selection support (date, datetime-local, checkbox, ...)
		}
	}

	// Scroll the list so a cell is visible: its row vertically (below the sticky header), and
	// its column horizontally. Used with focus({ preventScroll: true }), which turns off the
	// browser's own scrolling (that one hides rows behind the sticky header).
	function scrollCellIntoView(rowIndex: number, cellElement: HTMLElement) {
		scrollRowIntoView(rowIndex);
		const cell = (cellElement.closest('td') as HTMLElement | null) ?? cellElement;
		const container = cell.closest('.table-container') as HTMLElement | null;
		if (!container) return;
		const box = container.getBoundingClientRect();
		const cellBox = cell.getBoundingClientRect();
		const visibleRight = box.left + container.clientWidth; // excludes vertical scrollbar
		if (cellBox.left < box.left) {
			container.scrollLeft -= box.left - cellBox.left;
		} else if (cellBox.right > visibleRight) {
			container.scrollLeft += cellBox.right - visibleRight;
		}
	}

	function focusCell(rowIndex: number, colIndex: number, selectAll: boolean = true) {
		// Focus as soon as Svelte has updated the DOM: keys typed before the input has the
		// focus are lost (a fixed 50 ms delay dropped characters typed quickly after the first)
		afterRender(() => {
			// Try direct input/select first (regular inputs and simple lookups)
			const input = document.querySelector(
				`input[data-row="${rowIndex}"][data-col="${colIndex}"], select[data-row="${rowIndex}"][data-col="${colIndex}"]`
			) as HTMLInputElement | HTMLSelectElement | null;
			if (input) {
				input.focus({ preventScroll: true });
				scrollCellIntoView(rowIndex, input);
				if (input instanceof HTMLInputElement) {
					if (selectAll) {
						input.select();
					} else {
						placeCursorAtEnd(input);
					}
				}
				return true;
			}
			// Try container div (LookupDropdown/OptionDropdown wrapper) and focus the input inside
			const container = document.querySelector(
				`div[data-row="${rowIndex}"][data-col="${colIndex}"]`
			);
			if (container) {
				scrollCellIntoView(rowIndex, container as HTMLElement);
				const innerInput = container.querySelector('input') as HTMLInputElement | null;
				if (innerInput) {
					innerInput.focus({ preventScroll: true });
					if (selectAll) {
						innerInput.select();
					} else {
						placeCursorAtEnd(innerInput);
					}
				} else {
					// OptionDropdown uses a focusable div[role="combobox"] trigger
					const combobox = container.querySelector('[role="combobox"]') as HTMLElement | null;
					if (combobox) {
						combobox.focus({ preventScroll: true });
					}
				}
				return true;
			}
			return false;
		});
	}

	// Run a DOM step (focus) once Svelte has applied pending state; if its element is not
	// there yet (a component still mounting), try once more a little later
	function afterRender(step: () => boolean) {
		tick().then(() => {
			if (!step()) setTimeout(step, 50);
		});
	}

	// Focus a cell-selected element (div with data-cell-row/data-cell-col)
	function focusCellSelectedElement(row: number, col: number) {
		afterRender(() => {
			const el = document.querySelector(
				`[data-cell-row="${row}"][data-cell-col="${col}"]`
			) as HTMLElement | null;
			if (!el) return false;
			el.focus({ preventScroll: true });
			scrollCellIntoView(row, el);
			return true;
		});
	}


	// Handle blur from cell-editing inputs
	function handleEditingInputBlur() {
		const blurredRow = currentCellRow;
		const blurredCol = currentCellCol;

		setTimeout(() => {
			// Only a blur while still editing means the user left the table. A key or click
			// that moves to another cell first switches to cell-selected and removes this
			// input (which blurs it): that handler saves the cell itself. Treating that blur as
			// "left the table" saved the cell twice and went to navigation mode, whose delayed
			// page focus then took the focus away from the next cell.
			if (cellState !== 'cell-editing') return;

			// If we've moved to a different cell, the transition was already handled
			if (currentCellRow !== blurredRow || currentCellCol !== blurredCol) return;

			// If focus left the table (to another page, or the list's own search box or
			// toolbar), save and exit. Lookup/option dropdowns render inside the cell.
			const table = tableBodyElement?.closest('.table-container');
			if (!table?.contains(document.activeElement)) {
				const cols = visibleColumns();
				const field = cols[blurredCol];
				const record = displayRecords[blurredRow];
				if (record && field) {
					if (fieldTypes[field.source] === 'code' && typeof record[field.source] === 'string') {
						record[field.source] = record[field.source].toUpperCase();
					}
					handleCellBlur(record, blurredRow, field.source, true);
				}
				exitToNavigation(false);
			}
		}, 10);
	}

	// Handle click on a cell in read-only display
	function handleCellClick(rowIndex: number, colIndex: number) {
		if (isNavigation) {
			if (page.page.editable) {
				selectedIndex = rowIndex;
				enterCellSelected(rowIndex, colIndex);
			} else {
				handleRowClick(rowIndex);
			}
		} else {
			// Already in editable state, move to clicked cell
			confirmAndMoveTo(rowIndex, colIndex);
		}
	}

	// Handle click on the trailing blank row: start a new record at the end (BC/NAV)
	async function handlePlaceholderRowClick() {
		if (isNavigation) {
			if (!editableActive) {
				editableRecords = toEditableRecords();
				editableActive = true;
			}
		} else if (currentCellRow >= 0 && currentCellCol >= 0) {
			// Confirm the cell being left, as a click on any other cell does
			await confirmAndMoveTo(currentCellRow, currentCellCol);
		}
		insertNewRow(true);
		selectedIndex = currentCellRow;
	}

	// Handle delete record
	async function handleDelete() {
		if (selectedRecord) {
			confirm.show(
				t(DLG.DELETE_RECORD_TITLE),
				t(DLG.DELETE_RECORD_CONFIRM),
				async () => {
					await ondelete?.(selectedRecord);
				}
			);
		}
	}

	// Handle row click - just select the row
	function handleRowClick(index: number) {
		selectedIndex = index;
		// Give focus to the page so keyboard shortcuts work
		listPageElement?.focus();
	}

	// Open modal card
	async function openModalCard(record: Record<string, any>, editMode: boolean = false) {
		try {
			// Fetch the card page definition
			const response = await fetch(`/api/pages/${page.page.card_page_id}`);
			if (!response.ok) {
				throw new Error(t(ERR.FAILED_LOAD_CARD_PAGE));
			}

			const result = await response.json();
			if (!result.success) {
				throw new Error(result.error || t(ERR.FAILED_LOAD_CARD_PAGE));
			}

			// Deep clone ALL API response data to avoid Svelte reactivity issues
			const pageData = deepCopy(result.data);
			const pageCaptions = result.captions?.fields ? deepCopy(result.captions.fields) : {};
			const pageFieldTypes = result.captions?.field_types ? deepCopy(result.captions.field_types) : {};

			// Use the card page's source table (more reliable)
			const sourceTable = pageData?.page?.source_table || page.page.source_table;

			// Load the record data with options and lookups (for enum/lookup dropdowns)
			const recordId = getRecordId(record, primaryKeyField, primaryKeyFieldsList);

			let opts: Record<string, Record<string, string>> = {};
			let lkps: Record<string, LookupData> = {};
			let recData = { ...record };
			let origRecord = {};
			let isNew = false;

			if (recordId) {
				// Existing record - load it with options/lookups
				const recordResult = await api.getRecordWithCaptions(sourceTable, recordId);
				recData = deepCopy(recordResult.data);
				opts = recordResult.captions?.options ? deepCopy(recordResult.captions.options) : {};
				lkps = recordResult.captions?.lookups ? deepCopy(recordResult.captions.lookups) : {};
				origRecord = deepCopy(recData);
				isNew = false;
			} else {
				// New record - load options/lookups for dropdowns
				isNew = true;
				origRecord = {};
				try {
					const optLkp = await api.getTableOptionsAndLookups(sourceTable);
					opts = optLkp.options ? deepCopy(optLkp.options) : {};
					lkps = optLkp.lookups ? deepCopy(optLkp.lookups) : {};
				} catch (err) {
					console.error('Failed to load options/lookups:', err);
					opts = {};
					lkps = {};
				}
			}

			// Set all state at once to minimize re-renders
			modalCardPage = pageData;
			modalCaptions = pageCaptions;
			modalFieldTypes = pageFieldTypes;
			modalRecord = recData;
			modalOriginalRecord = origRecord;
			modalIsNewRecord = isNew;
			modalOptions = opts;
			modalLookups = lkps;
			modalOptionsLoaded = true;
			modalInitialEditMode = editMode || isNew;
			modalHadChanges = false;
			modalRecordDeleted = false;

			// Open the modal
			modalOpen = true;
		} catch (err) {
			console.error('Error opening modal card:', err);
			toast.error(t(ERR.FAILED_OPEN_CARD));
		}
	}

	// Close modal
	function closeModal() {
		const hadChanges = modalHadChanges;

		// Close the modal
		modalOpen = false;
		modalCardPage = null;
		modalRecord = {};
		modalIsNewRecord = false;
		skipNextAutoSave = false;
		modalCaptions = {};
		modalFieldTypes = {};
		modalOptions = {};
		modalLookups = {};
		modalOptionsLoaded = false;
		modalHadChanges = false;
		modalRecordDeleted = false;
		modalSaveBlocked = false;
		modalSaveBlockedMessage = '';

		// Refresh the list if changes were made
		if (hadChanges) {
			onaction?.('Refresh');
		}
	}

	// Clear error and reset the form for a fresh new record
	function handleClearError() {
		modalRecord = {};
		modalSaveBlocked = false;
		modalSaveBlockedMessage = '';
		modalIsNewRecord = true;
	}

	// Show save toast with debounce to prevent duplicates
	function showSaveToast() {
		const now = Date.now();
		if (now - lastSaveToastTime > 500) {
			toast.success(t(MSG.RECORD_SAVED));
			lastSaveToastTime = now;
		}
	}

	// Handle save from modal - returns true if save happened, false otherwise
	async function handleModalSave(savedRecord: Record<string, any>): Promise<boolean> {
		if (!modalCardPage || !modalOpen || modalSaving || modalRecordDeleted) {
			return false; // Prevent saves when modal closed, concurrent saves, or after delete
		}

		// Skip if this is a reactive trigger from programmatic update
		if (skipNextAutoSave) {
			skipNextAutoSave = false;
			return false;
		}

		// For existing records, skip save if nothing changed
		if (!modalIsNewRecord && !hasRecordChanged(savedRecord, modalOriginalRecord)) {
			return false;
		}

		// Save currently focused element before any state changes
		const activeElement = document.activeElement;
		const activeElementId = activeElement instanceof HTMLElement ? activeElement.id : null;

		modalSaving = true;
		try {
			const recordId = getRecordId(savedRecord, primaryKeyField, primaryKeyFieldsList);

			if (modalIsNewRecord) {
				// Insert new record
				const responseData = await api.insertRecord(page.page.source_table, savedRecord);
				// Add the new record to the list
				records = [...records, responseData];
				// After first save, it's no longer a new record
				modalIsNewRecord = false;
				// Update original to current for future change detection
				modalOriginalRecord = deepCopy(responseData);
				modalHadChanges = true;
				// Don't show toast for auto-save - it's disruptive during data entry
				// The CardPage has a subtle "Saved" indicator in the header
			} else {
				// Update existing record
				const responseData = await api.modifyRecord(page.page.source_table, recordId!, savedRecord);

				// Update the record in the list without full refresh
				const index = records.findIndex(r => getRecordId(r, primaryKeyField, primaryKeyFieldsList) === recordId);
				if (index !== -1) {
					records[index] = responseData;
				}
				// Update original for future change detection
				modalOriginalRecord = deepCopy(responseData);
				modalHadChanges = true;
				// No toast for modifications - too noisy with auto-save
			}
			// Note: We intentionally don't update modalRecord to avoid losing focus
			// The user's edits are preserved and the save was successful

			// Don't close modal - keep it open like Business Central
			// Restore focus if it was lost during state updates
			if (activeElementId) {
				setTimeout(() => {
					const element = document.getElementById(activeElementId);
					if (element && document.activeElement !== element) {
						element.focus();
					}
				}, 0);
			}
			return true; // Save happened
		} catch (err) {
			console.error('Error saving modal record:', err);
			const message = err instanceof Error ? err.message : t(ERR.FAILED_SAVE_RECORD);
			toast.error(message);

			// Block further edits if this was a new record that failed to save
			// (likely because the record already exists)
			// Use setTimeout to avoid Svelte prop update issues during error handling
			if (modalIsNewRecord) {
				setTimeout(() => {
					if (modalOpen) { // Only update if modal is still open
						modalSaveBlocked = true;
						modalSaveBlockedMessage = message;
					}
				}, 0);
			}

			return false; // Save failed
		} finally {
			modalSaving = false;
		}
	}

	// Handle actions from modal card
	async function handleModalAction(actionName: string) {
		if (!modalCardPage) return;

		switch (actionName) {
			case 'Back to List':
				// Close modal and return to list (triggered by Esc key or Back to List button)
				closeModal();
				break;
			case 'Delete':
				const deleteRecordId = getRecordId(modalRecord, primaryKeyField, primaryKeyFieldsList);
				if (deleteRecordId && window.confirm(`Delete this ${modalCardPage.page.caption}?`)) {
					// Mark as deleted BEFORE API call to prevent any pending auto-saves
					modalRecordDeleted = true;

					try {
						await api.deleteRecord(page.page.source_table, deleteRecordId);

						// Remove the record from the list
						records = records.filter(r => getRecordId(r, primaryKeyField, primaryKeyFieldsList) !== deleteRecordId);

						// Close the modal
						closeModal();

						toast.success(t(MSG.RECORD_DELETED_SUCCESS));
					} catch (err) {
						console.error('Delete error:', err);
						toast.error(t(ERR.FAILED_DELETE));
						// Reset flag if delete failed
						modalRecordDeleted = false;
					}
				}
				break;
			case 'Refresh':
				// Reload the modal record with options
				const refreshRecordId = getRecordId(modalRecord, primaryKeyField, primaryKeyFieldsList);
				if (refreshRecordId) {
					try {
						const refreshResult = await api.getRecordWithCaptions(page.page.source_table, refreshRecordId);
						modalRecord = refreshResult.data;
						modalOptions = refreshResult.captions?.options || {};
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
		// index is a displayed row (search/sort applied), not an index into records
		const record = displayRecords[index];
		if (record && page.page.card_page_id) {
			if (page.page.modal_card) {
				// Open as modal
				await openModalCard(record);
			} else {
				// Navigate to full page
				onrowclick?.(record);
			}
		}
	}

	// Build keyboard shortcut map from actions
	const shortcutMap = $derived(() => {
		const map: Record<string, () => void> = {};

		page.page.actions?.forEach((action) => {
			if (action.shortcut && action.enabled !== false) {
				// Normalize shortcut (e.g., "Esc" -> "Escape") to match keyboard event key names
				const normalizedShortcut = normalizeShortcut(action.shortcut);
				map[normalizedShortcut] = () => handleAction(action.name);
			}
		});

		// Add navigation shortcuts only when in navigation mode
		if (isNavigation) {
			map['ArrowDown'] = moveDown;
			map['ArrowUp'] = moveUp;
			map['Home'] = moveFirst;
			map['End'] = moveLast;
			map['Ctrl+Home'] = moveFirst;
			map['Ctrl+End'] = moveLast;
			map['PageDown'] = movePageDown;
			map['PageUp'] = movePageUp;
			map['Enter'] = () => {
				if (page.page.card_page_id) {
					openCard();
				} else if (page.page.editable && selectedIndex >= 0) {
					enterCellSelected(selectedIndex, 0);
				}
			};
			if (page.page.editable) {
				map['F2'] = () => {
					if (selectedIndex >= 0) enterCellSelected(selectedIndex, 0);
				};
				map['Ctrl+E'] = () => {
					if (selectedIndex >= 0) enterCellSelected(selectedIndex, 0);
				};
			}
			map['F5'] = () => handleAction('Refresh');
			map['Ctrl+D'] = () => {
				if (selectedRecord) handleDelete();
			};
		}

		return map;
	});

	// Navigation functions. Positions are absolute (offset + window position) so the
	// selection can move past the loaded window; windowIndexOf loads the window around it
	// Keys pressed while a window loads are not lost: they move pendingTarget, and the
	// running move goes on to the latest target once the load is done
	let pendingTarget: number | null = null;

	// Absolute position the next relative move starts from
	function currentRow(): number {
		return pendingTarget ?? windowOffset + Math.max(selectedIndex, 0);
	}

	async function moveToRow(absRow: number) {
		if (displayRecords.length === 0) return;
		const running = pendingTarget !== null;
		pendingTarget = Math.max(0, Math.min(absRow, total - 1));
		if (running) return;
		while (pendingTarget !== null) {
			const target: number = pendingTarget;
			const index = await windowIndexOf(target);
			if (pendingTarget === target) pendingTarget = null;
			selectedIndex = index;
		}
	}

	function moveDown() {
		moveToRow(currentRow() + 1);
	}

	function moveUp() {
		moveToRow(currentRow() - 1);
	}

	// Rows per page for PageUp/PageDown: as many rows as fit in the visible list area
	function rowsPerPage(): number {
		const container = tableBodyElement?.closest('.table-container') as HTMLElement | null;
		const row = tableBodyElement?.querySelector('tr') as HTMLElement | null;
		if (!container || !row || row.offsetHeight === 0) return 10;
		const headerHeight = (container.querySelector('thead') as HTMLElement | null)?.offsetHeight ?? 0;
		return Math.max(1, Math.floor((container.clientHeight - headerHeight) / row.offsetHeight) - 1);
	}

	function movePageDown() {
		moveToRow(currentRow() + rowsPerPage());
	}

	function movePageUp() {
		moveToRow(currentRow() - rowsPerPage());
	}

	// Home / Ctrl+Home: first record of the whole list
	function moveFirst() {
		moveToRow(0);
	}

	// End / Ctrl+End: last record of the whole list
	function moveLast() {
		moveToRow(total - 1);
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


	// Get visible columns (for rendering) with custom order applied
	const visibleColumns = $derived(() => {
		const fields = (page.page.layout.repeater?.fields || [])
			.map((field, index) => ({ field, index }))
			.filter(item => isItemVisible(item.field, columnCustomizations));

		// Sort by custom order if available
		return fields
			.sort((a, b) => {
				const orderA = columnCustomizations[a.field.source]?.order ?? a.index;
				const orderB = columnCustomizations[b.field.source]?.order ?? b.index;
				return orderA - orderB;
			})
			.map(item => item.field);
	});

	// FlowFields are computed, not stored: the server can not sort on them
	function isSortable(fieldSource: string): boolean {
		return !(page.page.flow_fields ?? []).includes(fieldSource);
	}

	// Toggle sort on a column (on the server, over all records)
	function handleSort(fieldSource: string) {
		if (!isSortable(fieldSource)) return;
		if (sortField === fieldSource) {
			// Toggle direction if same field
			sortDirection = sortDirection === 'asc' ? 'desc' : 'asc';
		} else {
			// New field, start with ascending
			sortField = fieldSource;
			sortDirection = 'asc';
		}
		applySearchAndSort({ sort: { field: sortField, direction: sortDirection } });
	}

	// Column resize handlers
	function handleResizeStart(e: MouseEvent, fieldSource: string, currentWidth: number) {
		e.preventDefault();
		isResizing = true;
		resizeField = fieldSource;
		resizeStartX = e.clientX;
		resizeStartWidth = currentWidth;

		// Prevent text selection while dragging
		document.body.style.cursor = 'col-resize';
		document.body.style.userSelect = 'none';

		// Add document-level listeners for drag
		document.addEventListener('mousemove', handleResizeMove);
		document.addEventListener('mouseup', handleResizeEnd);
	}

	function handleResizeMove(e: MouseEvent) {
		if (!isResizing || !resizeField) return;

		const delta = e.clientX - resizeStartX;
		const newWidth = Math.max(50, resizeStartWidth + delta); // Minimum 50px width

		columnWidths = {
			...columnWidths,
			[resizeField]: newWidth
		};
	}

	function handleResizeEnd() {
		if (isResizing && resizeField) {
			// Save to localStorage
			const userId = $currentUser?.user_id || 'anonymous';
			saveColumnWidths(userId, page.page.id, columnWidths);
		}

		isResizing = false;
		resizeField = null;

		// Reset cursor and selection
		document.body.style.cursor = '';
		document.body.style.userSelect = '';

		// Remove document-level listeners
		document.removeEventListener('mousemove', handleResizeMove);
		document.removeEventListener('mouseup', handleResizeEnd);
	}

	// Get column width (custom or default from field definition)
	function getColumnWidth(field: Field): number {
		return columnWidths[field.source] ?? field.width ?? 150;
	}

	// Open customize modal
	function handleCustomize() {
		customizeModalOpen = true;
	}

	// Save customizations
	function handleSaveCustomizations(customizations: Record<string, ItemCustomization>) {
		columnCustomizations = customizations;
		const userId = $currentUser?.user_id || 'anonymous';
		savePageCustomizations(userId, page.page.id, customizations);
	}

	// Toggle row numbers
	function handleToggleRowNumbers() {
		showRowNumbers = !showRowNumbers;
		const userId = $currentUser?.user_id || 'anonymous';
		saveRowNumbersPreference(userId, page.page.id, showRowNumbers);
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

	// Clear search
	function clearSearch() {
		searchQuery = '';
		clearTimeout(searchTimer);
		applySearchAndSort({ search: '' });
		searchInputElement?.focus();
	}

	// Focus search on Ctrl+F
	function handleSearchShortcut(e: KeyboardEvent) {
		if (e.ctrlKey && e.key === 'f') {
			e.preventDefault();
			searchInputElement?.focus();
			searchInputElement?.select();
		}
	}
</script>

<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
<div class="list-page" use:shortcuts={shortcutMap()} tabindex="0" bind:this={listPageElement} onkeydown={handleSearchShortcut} role="application" aria-label={page.page.caption}>
	<PageHeader title={page.page.caption}>
		{#snippet leftActions()}
			{#if page.page.editable}
				<!-- Save state indicator - fixed width container to prevent layout shift -->
				<div class="save-state-container">
					<div class="saving-indicator" class:visible={saveState === 'saving'}>
						<svg class="animate-spin h-4 w-4 text-blue-600" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
						</svg>
						<span class="text-sm text-gray-600 dark:text-gray-400">{t(MSG.SAVING)}</span>
					</div>
					<div class="saved-indicator" class:visible={saveState === 'saved'}>
						<svg class="h-4 w-4 text-green-600" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
						</svg>
						<span class="text-sm text-green-600 dark:text-green-400 font-medium">{t(MSG.SAVED)}</span>
					</div>
				</div>
			{/if}

			{#each page.page.actions?.filter((a) => a.promoted) || [] as action}
					{@const isDisabled = (() => {
						// New and Refresh are always enabled
						if (action.name === 'New' || action.name === 'Refresh') return false;

						// Edit is disabled if page is not editable
						if (action.name === 'Edit') return page.page.editable !== true;

						// Delete requires a selected record
						if (action.name === 'Delete') return !selectedRecord;

						// Other buttons require selection
						return !selectedRecord;
					})()}
					{@const variant = 'secondary'}
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

				<!-- Quick Search -->
				<div class="search-container">
					<svg
						class="search-icon"
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
						/>
					</svg>
					<input
						type="text"
						class="search-input"
						placeholder={t(LIST.SEARCH_PLACEHOLDER)}
						bind:value={searchQuery}
						bind:this={searchInputElement}
						oninput={handleSearchInput}
					/>
					{#if searchQuery}
						<button
							type="button"
							class="clear-search-btn"
							onclick={clearSearch}
							title={t(LIST.CLEAR_SEARCH)}
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
									d="M6 18L18 6M6 6l12 12"
								/>
							</svg>
						</button>
					{/if}
				</div>
		{/snippet}

		{#snippet rightActions()}
			<!-- Row Numbers toggle button -->
			<Button
					variant={showRowNumbers ? 'primary' : 'secondary'}
					size="sm"
					onclick={handleToggleRowNumbers}
					title={t(LIST.TOGGLE_ROW_NUMBERS)}
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
							d="M7 20l4-16m2 16l4-16M6 9h14M4 15h14"
						/>
					</svg>
					<span class="ml-1">#</span>
				</Button>

				<!-- Customize button -->
				<Button variant="secondary" size="sm" onclick={handleCustomize} title={t(LIST.CUSTOMIZE_COLUMNS)}>
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
					<span class="ml-1">{t(LIST.CUSTOMIZE)}</span>
				</Button>

				<!-- Filter button -->
				<Button
					variant={filterPaneOpen ? 'primary' : 'secondary'}
					size="sm"
					onclick={handleToggleFilters}
					title={t(LIST.TOGGLE_FILTERS)}
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
					<span class="ml-1">{t(LIST.FILTER)}</span>
					{#if currentFilters.length > 0}
						<span class="ml-1 px-1.5 py-0.5 text-xs bg-blue-600 text-white rounded-full">
							{currentFilters.length}
						</span>
					{/if}
				</Button>
		{/snippet}
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

		<div class="table-container" onscroll={handleTableScroll}>
		<table class="table">
			<thead>
				<tr>
					{#if showRowNumbers}
						<th class="row-number-header">#</th>
					{/if}
					{#each visibleColumns() as field}
						<th style="width: {getColumnWidth(field)}px">
							<div class="th-content">
								<span class="th-label">{getFieldCaption(field.source, captions, field.caption)}</span>
								{#if isSortable(field.source)}
								<button
									type="button"
									class="sort-btn"
									onclick={(e) => {
										e.stopPropagation();
										handleSort(field.source);
									}}
									title={sortField === field.source
										? `Sort ${sortDirection === 'asc' ? 'descending' : 'ascending'}`
										: 'Sort ascending'}
								>
									{#if sortField === field.source}
										{#if sortDirection === 'asc'}
											<svg class="sort-icon" viewBox="0 0 24 24" fill="currentColor">
												<path d="M7 14l5-5 5 5H7z"/>
											</svg>
										{:else}
											<svg class="sort-icon" viewBox="0 0 24 24" fill="currentColor">
												<path d="M7 10l5 5 5-5H7z"/>
											</svg>
										{/if}
									{:else}
										<svg class="sort-icon sort-icon-inactive" viewBox="0 0 24 24" fill="currentColor">
											<path d="M7 10l5 5 5-5H7z"/>
										</svg>
									{/if}
								</button>
								{/if}
							</div>
							<!-- Resize handle -->
							<button
								type="button"
								class="resize-handle"
								aria-label="Resize column"
								tabindex="-1"
								onmousedown={(e) => handleResizeStart(e, field.source, getColumnWidth(field))}
							></button>
						</th>
					{/each}
				</tr>
			</thead>
			<tbody bind:this={tableBodyElement}>
				{#each displayRecords as record, index (getRecordKey(record, primaryKeyField, primaryKeyFieldsList))}
					<tr
						class={cn(
							isNavigation ? 'cursor-pointer' : '',
							isNavigation && selectedIndex === index && 'selected',
							record._isNew && 'new-row',
							(windowOffset + index) % 2 === 1 ? 'row-even' : 'row-odd'
						)}
					>
						{#if showRowNumbers}
							<td class="row-number-cell">{windowOffset + index + 1}</td>
						{/if}
						{#each visibleColumns() as field, colIndex}
							<td class="p-0 border-r border-b border-gray-300 dark:border-gray-600">
								{#if isCellEditing && currentCellRow === index && currentCellCol === colIndex}
									<!-- Cell-Editing Mode - Active input for this specific cell -->
									{#if typeof record[field.source] === 'boolean' || fieldTypes[field.source] === 'bool'}
										<div class="edit-cell-input flex items-center">
											<input
												type="checkbox"
												data-row={index}
												data-col={colIndex}
												bind:checked={record[field.source]}
												onfocus={() => {
													currentCellRow = index;
													currentCellCol = colIndex;
												}}
												onchange={async () => {
													await handleCellBlur(record, index);
												}}
												onkeydown={(e) => handleCellKeyDown(e, index, colIndex)}
												onblur={handleEditingInputBlur}
											/>
										</div>
									{:else if lookups[field.source]?.columns && lookups[field.source]?.rows?.length}
										<!-- Advanced lookup with columns - LookupDropdown -->
										<!-- svelte-ignore a11y_no_static_element_interactions -->
										<div data-row={index} data-col={colIndex} class="lookup-cell-wrapper"
											onkeydown={(e) => handleLookupCellKeyDown(e, index, colIndex)}>
											<LookupDropdown
												columns={lookups[field.source].columns ?? []}
												rows={lookups[field.source].rows ?? []}
												value={record[field.source] || ''}
												fieldName={getFieldCaption(field.source, captions, field.caption)}
												captions={captions}
												compact={true}
												onselect={(key) => {
													record[field.source] = key;
													currentCellRow = index;
													currentCellCol = colIndex;
												}}
												onblur={() => {
													handleEditingInputBlur();
												}}
											/>
										</div>
									{:else if lookups[field.source]?.simple}
										<!-- Simple lookup - select dropdown -->
										<select
											data-row={index}
											data-col={colIndex}
											class="edit-cell-input"
											value={record[field.source] || ''}
											onfocus={() => {
												currentCellRow = index;
												currentCellCol = colIndex;
											}}
											onchange={(e) => {
												record[field.source] = (e.target as HTMLSelectElement).value;
												handleCellBlur(record, index, field.source);
											}}
											onkeydown={(e) => handleCellKeyDown(e, index, colIndex)}
											onblur={handleEditingInputBlur}
										>
											<option value="">—</option>
											{#each Object.entries(lookups[field.source].simple ?? {}) as [key, label]}
												<option value={key}>{label}</option>
											{/each}
										</select>
									{:else if options[field.source]}
										<!-- Option field - OptionDropdown -->
										<!-- svelte-ignore a11y_no_static_element_interactions -->
										<div data-row={index} data-col={colIndex} class="lookup-cell-wrapper"
											onkeydown={(e) => handleLookupCellKeyDown(e, index, colIndex)}>
											<OptionDropdown
												options={options[field.source]}
												value={record[field.source]}
												compact={true}
												onselect={(newValue) => {
													record[field.source] = newValue;
													currentCellRow = index;
													currentCellCol = colIndex;
												}}
												onblur={() => {
													handleEditingInputBlur();
												}}
											/>
										</div>
									{:else}
										<input
											type={getCellInputType(field.source)}
											data-row={index}
											data-col={colIndex}
											class="edit-cell-input"
											bind:value={record[field.source]}
											onfocus={() => {
												currentCellRow = index;
												currentCellCol = colIndex;
											}}
											onblur={handleEditingInputBlur}
											onkeydown={(e) => handleCellKeyDown(e, index, colIndex)}
										/>
									{/if}
								{:else if (isCellSelected || isCellEditing) && currentCellRow === index && currentCellCol === colIndex}
									<!-- Cell-Selected Mode - Blue border, no cursor, keyboard-driven -->
									<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
									<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
									<!-- svelte-ignore a11y_no_static_element_interactions -->
									{#if typeof record[field.source] === 'boolean' || fieldTypes[field.source] === 'bool'}
										<div
											class="cell-selected-content cell-selected-active"
											tabindex="0"
											data-cell-row={index}
											data-cell-col={colIndex}
											onkeydown={(e) => handleCellSelectedKeyDown(e, index, colIndex)}
										>
											<input type="checkbox" checked={record[field.source]}
												onclick={() => {
													record[field.source] = !record[field.source];
													handleCellBlur(record, index);
												}}
											/>
										</div>
									{:else if lookups[field.source]?.columns || lookups[field.source]?.simple}
										<!-- Cell-selected with lookup: show value + dropdown arrow -->
										<div
											class="cell-selected-content cell-selected-active cell-selected-lookup"
											tabindex="0"
											data-cell-row={index}
											data-cell-col={colIndex}
											ondblclick={() => enterCellEditing(false)}
											onkeydown={(e) => handleCellSelectedKeyDown(e, index, colIndex)}
										>
											<span class="cell-selected-lookup-value"><span class="cell-selected-text">{#if lookups[field.source]?.rows?.length}{formatLookupValue(record[field.source], lookups[field.source])}{:else}{formatCellValue(record[field.source], field.source)}{/if}</span></span>
											<!-- svelte-ignore a11y_click_events_have_key_events -->
											<span
												class="cell-selected-lookup-arrow"
												onclick={(e) => {
													e.stopPropagation();
													enterCellEditing(false);
												}}
												role="button"
												tabindex="-1"
												aria-label="Open lookup"
											>▼</span>
										</div>
									{:else if options[field.source]}
										<!-- Cell-selected with option: show value + dropdown arrow -->
										<div
											class="cell-selected-content cell-selected-active cell-selected-lookup"
											tabindex="0"
											data-cell-row={index}
											data-cell-col={colIndex}
											ondblclick={() => enterCellEditing(false)}
											onkeydown={(e) => handleCellSelectedKeyDown(e, index, colIndex)}
										>
											<span class="cell-selected-lookup-value"><span class="cell-selected-text">{formatOptionValue(record[field.source], options[field.source])}</span></span>
											<!-- svelte-ignore a11y_click_events_have_key_events -->
											<span
												class="cell-selected-lookup-arrow"
												onclick={(e) => {
													e.stopPropagation();
													enterCellEditing(false);
												}}
												role="button"
												tabindex="-1"
												aria-label="Open options"
											>▼</span>
										</div>
									{:else}
										<div
											class="cell-selected-content cell-selected-active"
											tabindex="0"
											data-cell-row={index}
											data-cell-col={colIndex}
											ondblclick={() => enterCellEditing(false)}
											onkeydown={(e) => handleCellSelectedKeyDown(e, index, colIndex)}
										>
											<span class="cell-selected-text">{formatCellValue(record[field.source], field.source)}</span>
										</div>
									{/if}
								{:else}
									<!-- Read-only display -->
									<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
									<!-- svelte-ignore a11y_no_static_element_interactions -->
									<!-- svelte-ignore a11y_click_events_have_key_events -->
									<div
										class={cn('read-cell-content', getFieldStyleClasses(field))}
										onclick={() => handleCellClick(index, colIndex)}
									>
										{#if typeof record[field.source] === 'boolean' || fieldTypes[field.source] === 'bool'}
											{#if page.page.editable}
												<input type="checkbox" checked={record[field.source]}
													onclick={async (e) => {
														e.stopPropagation();
														record[field.source] = !record[field.source];
														const recordId = getRecordId(record, primaryKeyField, primaryKeyFieldsList);
														if (recordId !== undefined) {
															try {
																await api.modifyRecord(page.page.source_table, recordId, { [field.source]: record[field.source] });
															} catch (err) {
																record[field.source] = !record[field.source];
															}
														}
													}}
												/>
											{:else}
												<input type="checkbox" checked={record[field.source]} disabled class="cursor-not-allowed" />
											{/if}
										{:else if field.primary_key && page.page.card_page_id && isNavigation}
											<button
												type="button"
												class="primary-key-link"
												onclick={(e) => {
													e.stopPropagation();
													handlePrimaryKeyClick(index);
												}}
											>
												{formatCellValue(record[field.source], field.source)}
											</button>
										{:else if field.drilldown && record[field.source]}
											<button
												type="button"
												class="primary-key-link"
												onclick={(e) => {
													e.stopPropagation();
													const filterValue = record[field.drilldown_filter_value || ''] ?? '';
													window.location.href = `/pages/${field.drilldown}?filter=${field.drilldown_filter_field}=${filterValue}`;
												}}
											>
												{formatCellValue(record[field.source], field.source)}
											</button>
											{:else if options[field.source]}
												{formatOptionValue(record[field.source], options[field.source])}
										{:else if lookups[field.source]?.rows?.length}
											{formatLookupValue(record[field.source], lookups[field.source])}
										{:else}
											{formatCellValue(record[field.source], field.source)}
										{/if}
									</div>
								{/if}
							</td>
						{/each}
					</tr>
				{/each}
				{#if page.page.editable && !searchQuery.trim() && !moreBelow && !(displayRecords.length > 0 && isEmptyNewRecord(displayRecords[displayRecords.length - 1]))}
					<!-- BC-style trailing blank row: click it to start a new record -->
					<tr class="placeholder-row" onclick={handlePlaceholderRowClick}>
						{#if showRowNumbers}
							<td class="row-number-cell"></td>
						{/if}
						{#each visibleColumns() as _field}
							<td class="p-0 border-r border-b border-gray-300 dark:border-gray-600">
								<div class="read-cell-content"></div>
							</td>
						{/each}
					</tr>
				{/if}
		</tbody>
		</table>
		</div>
	</div>

	<div class="status-bar">
		<span class="text-sm text-gray-600 dark:text-gray-400">
			{total} record{total !== 1 ? 's' : ''}{searchQuery ? ' (filtered)' : ''}
			{#if isNavigation && selectedIndex >= 0 && selectedIndex < displayRecords.length}
				• Row {windowOffset + selectedIndex + 1} selected
			{:else if !isNavigation && currentCellRow >= 0 && currentCellCol >= 0}
				• Cell [{windowOffset + currentCellRow + 1}, {currentCellCol + 1}] {isCellEditing ? '(editing)' : '(selected)'}
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
		fieldTypes={modalFieldTypes}
		options={modalOptions}
		lookups={modalLookups}
		initialEditMode={modalInitialEditMode}
		saveBlocked={modalSaveBlocked}
		saveBlockedMessage={modalSaveBlockedMessage}
		onclose={closeModal}
		onaction={handleModalAction}
		onsave={handleModalSave}
		onclearerror={handleClearError}
	/>
{/if}

<!-- Customize Columns Modal -->
{#if customizeModalOpen}
	<CustomizeFieldsModal
		open={customizeModalOpen}
		{page}
		customizations={columnCustomizations}
		mode="list"
		onclose={() => customizeModalOpen = false}
		onsave={handleSaveCustomizations}
	/>
{/if}

<!-- Confirm Modal -->
<ConfirmModal
	open={$confirm.open}
	title={$confirm.title}
	message={$confirm.message}
	confirmText={t(BTN.DELETE)}
	variant="danger"
	onconfirm={confirm.confirm}
	oncancel={confirm.cancel}
/>

<!-- Progress Modal (for codeunits with progress) -->
<ProgressModal
	open={progressModalOpen}
	title={progressTitle}
	message={progressMessage}
	progress={progressValue}
	error={progressError}
	confirmMode={progressConfirmMode}
	confirmMessage={progressConfirmMessage}
	onConfirmResponse={confirmResponseCallback}
	inputMode={progressInputMode}
	inputFields={progressInputFields}
	onInputResponse={inputResponseCallback}
/>

<!-- Codeunit Dialog Modal -->
{#if dialogOpen && dialogData}
	<Modal open={dialogOpen} onclose={() => { dialogOpen = false; dialogData = null; }}>
		<div class="p-6">
			<div class="flex items-start gap-4">
				{#if dialogData.type === 'info'}
					<div class="flex-shrink-0 w-10 h-10 rounded-full bg-blue-100 dark:bg-blue-900/30 flex items-center justify-center">
						<svg class="w-6 h-6 text-blue-600 dark:text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
						</svg>
					</div>
				{:else if dialogData.type === 'success'}
					<div class="flex-shrink-0 w-10 h-10 rounded-full bg-green-100 dark:bg-green-900/30 flex items-center justify-center">
						<svg class="w-6 h-6 text-green-600 dark:text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
						</svg>
					</div>
				{:else if dialogData.type === 'warning'}
					<div class="flex-shrink-0 w-10 h-10 rounded-full bg-yellow-100 dark:bg-yellow-900/30 flex items-center justify-center">
						<svg class="w-6 h-6 text-yellow-600 dark:text-yellow-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
						</svg>
					</div>
				{:else if dialogData.type === 'error'}
					<div class="flex-shrink-0 w-10 h-10 rounded-full bg-red-100 dark:bg-red-900/30 flex items-center justify-center">
						<svg class="w-6 h-6 text-red-600 dark:text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
						</svg>
					</div>
				{/if}
				<div class="flex-1">
					<h3 class="text-lg font-semibold text-gray-900 dark:text-white">{dialogData.title}</h3>
					<p class="mt-2 text-gray-600 dark:text-gray-300">{dialogData.message}</p>
				</div>
			</div>
			<div class="mt-6 flex justify-end">
				<button
					class="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 dark:focus:ring-offset-gray-800"
					onclick={() => { dialogOpen = false; dialogData = null; }}
				>
					{t(BTN.OK)}
				</button>
			</div>
		</div>
	</Modal>
{/if}

<style>
	.list-page {
		@apply flex flex-col gap-4;
		height: calc(100vh - 180px); /* Account for menu, breadcrumb, padding */
	}

	.list-content {
		@apply flex flex-1 gap-4;
		min-height: 0;
		overflow: hidden;
	}

	.table-container {
		overflow: auto;
		max-height: 100%;
		width: 100%;
	}

	.table {
		@apply w-full border border-gray-200 rounded-lg;
		@apply dark:border-gray-700;
		border-collapse: separate;
		border-spacing: 0;
		background: transparent;
		table-layout: fixed;
	}

	.table thead {
		@apply z-10;
	}

	.table tbody {
		background: transparent;
	}

	/* BC column header: light background with bold dark text (dark mode: dark header) */
	.table th {
		@apply px-4 py-3 text-left text-sm font-bold;
		background-color: #ffffff;
		color: #212121;
		border-right: 1px solid #e5e7e9;
		border-bottom: 1px solid #d3d6da;
		position: sticky;
		top: 0;
		z-index: 10;
	}

	:global(.dark) .table th {
		background-color: #1e1e1e;
		color: #f7f7f7;
		border-right-color: rgba(255, 255, 255, 0.1);
		border-bottom-color: rgba(255, 255, 255, 0.2);
	}

	.table th:last-child {
		border-right: none;
	}

	/* Row number column styles */
	.row-number-header {
		width: 50px !important;
		min-width: 50px;
		max-width: 50px;
		text-align: center;
	}

	.row-number-cell {
		width: 50px;
		min-width: 50px;
		max-width: 50px;
		text-align: center;
		font-size: 0.75rem;
		color: #737d8a;
		border-right: 1px solid #d3d6da;
		border-bottom: 1px solid #d3d6da;
	}

	:global(.dark) .row-number-cell {
		color: white;
		background-color: rgb(30 30 30); /* gray-800 - matches normal columns */
		border-color: #505c6d; /* gray-600 */
	}

	.resize-handle {
		position: absolute;
		right: -3px;
		top: 0;
		bottom: 0;
		width: 8px;
		cursor: col-resize;
		background: transparent;
		z-index: 20;
		border: none;
		padding: 0;
		margin: 0;
		outline: none;
	}

	.resize-handle:hover {
		background: rgba(0, 131, 143, 0.5);
	}

	.resize-handle:active {
		background: rgba(0, 131, 143, 0.8);
	}

	.th-content {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 4px;
	}

	.th-label {
		flex: 1;
		color: inherit;
	}

	.sort-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		background: none;
		border: none;
		padding: 2px;
		cursor: pointer;
		border-radius: 2px;
		opacity: 0.5;
		transition: opacity 0.15s;
	}

	.sort-btn:hover {
		opacity: 1;
		background: rgba(0, 0, 0, 0.06);
	}

	:global(.dark) .sort-btn:hover {
		background: rgba(255, 255, 255, 0.1);
	}

	.sort-icon {
		width: 16px;
		height: 16px;
	}

	.sort-icon-inactive {
		opacity: 0.4;
	}

	.table tbody tr {
		@apply border-b border-gray-200 hover:bg-blue-50 transition-colors;
		@apply dark:border-gray-700 dark:hover:bg-gray-700;
	}

	/* Zebra striping - alternating row colors, by position in the whole list (the
	   loaded window starts at any offset) */
	.table tbody tr.row-even {
		@apply bg-gray-50;
		@apply dark:bg-gray-800/50;
	}

	.table tbody tr.row-odd {
		@apply bg-white;
		@apply dark:bg-gray-900;
	}

	.table tbody tr.selected {
		background-color: #cce8ea !important; /* bg-blue-100 */
	}

	.table tbody tr.selected:hover {
		background-color: #cce8ea !important;
	}

	:global(.dark) .table tbody tr.selected {
		background-color: #00585c !important; /* bg-blue-800 */
		color: white;
	}

	:global(.dark) .table tbody tr.selected:hover {
		background-color: #00585c !important;
	}

	:global(.dark) .table tbody tr.selected td,
	:global(.dark) .table tbody tr.selected .read-cell-content,
	:global(.dark) .table tbody tr.selected .primary-key-link {
		color: white !important;
	}

	.table tbody tr.new-row {
		background-color: #e6f3f4 !important;
	}

	:global(.dark) .table tbody tr.new-row {
		background-color: #003a3e !important; /* Dark teal background for new rows */
	}

	.table td {
		padding: 2px 6px;
		font-size: 0.875rem;
		line-height: 1.3;
		vertical-align: bottom;
	}

	.status-bar {
		@apply px-4 py-2 bg-gray-50 border-t border-gray-200 rounded-b;
	}

	:global(.dark) .status-bar {
		background-color: #1e1e1e; /* gray-800 */
		border-color: #303032; /* gray-700 */
		color: #d3d6da; /* gray-300 */
	}

	.lookup-cell-wrapper {
		width: 100%;
	}

	.edit-cell-input {
		display: block !important;
		width: 100%;
		height: 1.3em !important;
		min-height: 0 !important;
		max-height: 1.3em !important;
		padding: 2px 6px !important;
		line-height: 1.3 !important;
		font-size: 0.875rem;
		background: transparent !important;
		border: 0 !important;
		outline: 0 !important;
		box-shadow: none !important;
		-webkit-appearance: none !important;
		-moz-appearance: none !important;
		appearance: none !important;
		margin: 0 !important;
		box-sizing: content-box !important;
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
		background: #121212; /* BC dark mode: list rows on the page background */
	}

	/* Selected rows - make td background transparent to show row highlight */
	tbody tr.selected td.p-0 {
		background: transparent;
	}

	/* Normal mode cell content - match edit mode input exactly */
	.read-cell-content {
		display: block;
		width: 100%;
		height: 1.3em;
		min-height: 1.3em;
		max-height: 1.3em;
		padding: 2px 6px;
		line-height: 1.3;
		font-size: 0.875rem;
		margin: 0;
		box-sizing: content-box;
		overflow: hidden;
	}

	/* Cell-selected state - same dimensions as read-cell-content with blue border */
	.cell-selected-content {
		display: block;
		width: auto; /* fill the cell; 100% plus padding would overflow into the next cell */
		height: 1.3em;
		min-height: 1.3em;
		max-height: 1.3em;
		padding: 2px 6px;
		line-height: 1.3;
		font-size: 0.875rem;
		margin: 0;
		box-sizing: content-box;
		overflow: hidden;
		outline: none;
	}

	.cell-selected-active {
		outline: 1px solid rgba(0, 132, 137, 0.45); /* subtle blue-600 frame */
		outline-offset: -1px; /* inset so it doesn't shift layout */
		background: rgba(230, 243, 244, 0.6); /* faint blue-50 highlight */
	}

	:global(.dark) .cell-selected-active {
		outline-color: rgba(0, 131, 143, 0.5);
		background: rgba(0, 131, 143, 0.06);
		color: white;
	}

	.cell-selected-lookup {
		display: flex;
		align-items: center;
		padding-right: 0;
	}

	.cell-selected-lookup-value {
		flex: 1;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.cell-selected-lookup-arrow {
		flex-shrink: 0;
		padding: 0 4px;
		font-size: 0.5rem;
		color: #737d8a;
		cursor: pointer;
		line-height: 1;
	}

	.cell-selected-lookup-arrow:hover {
		color: #008489;
	}

	:global(.dark) .cell-selected-lookup-arrow {
		color: #a4b0c4;
	}

	:global(.dark) .cell-selected-lookup-arrow:hover {
		color: #37a1a5;
	}

	/* Primary key link - looks like a hyperlink */
	.primary-key-link {
		color: #008489;
		text-decoration: underline;
		background: none;
		border: none;
		padding: 0;
		font: inherit;
		cursor: pointer;
		text-align: inherit;
	}

	.primary-key-link:hover {
		color: #006e72;
	}

	:global(.dark) .primary-key-link {
		color: #37a1a5;
	}

	:global(.dark) .primary-key-link:hover {
		color: #66b9bf;
	}

	/* Quick Search Styles */
	.search-container {
		@apply relative flex items-center;
		margin-left: 1rem;
	}

	.search-icon {
		@apply absolute left-3 w-4 h-4 text-gray-400 pointer-events-none;
	}

	:global(.dark) .search-icon {
		color: #a4b0c4;
	}

	.search-input {
		@apply pl-9 pr-8 py-1.5 text-sm rounded-md border border-gray-300;
		@apply bg-white text-gray-900;
		@apply focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500;
		width: 200px;
		transition: width 0.2s ease;
	}

	.search-input:focus {
		width: 280px;
	}

	.search-input::placeholder {
		@apply text-gray-400;
	}

	:global(.dark) .search-input {
		background-color: #303032;
		border-color: #505c6d;
		color: white;
	}

	:global(.dark) .search-input::placeholder {
		color: #a4b0c4;
	}

	:global(.dark) .search-input:focus {
		border-color: #00838f;
		box-shadow: 0 0 0 2px rgba(0, 131, 143, 0.3);
	}

	.clear-search-btn {
		@apply absolute right-2 p-0.5 rounded text-gray-400 hover:text-gray-600;
		@apply hover:bg-gray-100 transition-colors;
	}

	:global(.dark) .clear-search-btn {
		color: #a4b0c4;
	}

	:global(.dark) .clear-search-btn:hover {
		color: #d3d6da;
		background-color: #505c6d;
	}

	/* Header save indicator (same as CardPage) */
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

	/* Trailing blank row (BC-style new-record affordance) */
	.placeholder-row {
		cursor: pointer;
	}

	/* Value in the selected cell looks like selected text (BC): blue in light mode, grey in
	   dark mode. Typing replaces it; F2 puts the cursor at the end. */
	.cell-selected-text {
		background-color: #0078d4;
		color: #ffffff;
	}

	:global(.dark) .cell-selected-text {
		background-color: #505c6d;
		color: #f7f7f7;
	}

	/* Text selected inside a cell being edited: same colors */
	.edit-cell-input::selection {
		background-color: #0078d4;
		color: #ffffff;
	}

	:global(.dark) .edit-cell-input::selection {
		background-color: #505c6d;
		color: #f7f7f7;
	}
</style>
