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
	import { shortcuts, normalizeShortcut } from '$lib/utils/shortcuts';
	import { cn } from '$lib/utils/cn';
	import { api } from '$lib/services/api';
	import { currentUser } from '$lib/stores/user';
	import { getFieldCaption, getFieldStyleClasses, formatValue, formatOptionValue, formatLookupValue, isItemVisible, isDateType, isDateTimeType, formatDate, formatDateTime, type ItemCustomization } from '$lib/utils/fieldHelpers';
	import { currentLanguage } from '$lib/stores/session';
	import { loadPageCustomizations, savePageCustomizations, loadColumnWidths, saveColumnWidths, loadRowNumbersPreference, saveRowNumbersPreference } from '$lib/utils/customizationStorage';
	import { getRecordId, getRecordKey, getPrimaryKeyField, getPrimaryKeyFields, deepCopy, hasRecordChanged, isEmptyRecord, hasRecordData } from '$lib/utils/recordHelpers';

	interface Props {
		page: PageDefinition;
		records?: Array<Record<string, any>>;
		captions?: Record<string, string>;
		fieldTypes?: Record<string, string>; // Field type metadata (e.g., "bool", "code", "text")
		options?: Record<string, Record<string, string>>; // Option field values (enum lookups)
		lookups?: Record<string, LookupData>; // Table relation lookup values
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
		fieldTypes = {},
		options = {},
		lookups = {},
		currentFilters = [],
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

	// Filter records by search query
	const filteredRecords = $derived(() => {
		const sourceRecords = editableActive ? editableRecords : records;
		if (!searchQuery.trim()) return sourceRecords;

		const query = searchQuery.toLowerCase().trim();
		const columns = visibleColumns();

		return sourceRecords.filter(record => {
			// Search across all visible columns
			return columns.some(field => {
				const value = record[field.source];
				if (value == null) return false;
				return String(value).toLowerCase().includes(query);
			});
		});
	});

	// Sort records
	const sortedRecords = $derived(() => {
		const sourceRecords = filteredRecords();
		if (!sortField) return sourceRecords;

		const field = sortField; // TypeScript now knows field is non-null
		return [...sourceRecords].sort((a, b) => {
			const aVal = a[field];
			const bVal = b[field];

			// Handle null/undefined
			if (aVal == null && bVal == null) return 0;
			if (aVal == null) return sortDirection === 'asc' ? -1 : 1;
			if (bVal == null) return sortDirection === 'asc' ? 1 : -1;

			// Compare based on type
			let comparison = 0;
			if (typeof aVal === 'number' && typeof bVal === 'number') {
				comparison = aVal - bVal;
			} else if (typeof aVal === 'boolean' && typeof bVal === 'boolean') {
				comparison = aVal === bVal ? 0 : aVal ? 1 : -1;
			} else {
				comparison = String(aVal).localeCompare(String(bVal));
			}

			return sortDirection === 'asc' ? comparison : -comparison;
		});
	});

	// Records to display
	const displayRecords = $derived(sortedRecords());

	// Track list page element for focus
	let listPageElement: HTMLDivElement | null = null;

	// Auto-focus the page on mount and when records load (but not when modal is open)
	$effect(() => {
		if (listPageElement && isNavigation && records.length > 0 && !modalOpen) {
			setTimeout(() => {
				listPageElement?.focus();
			}, 100);
		}
	});

	// Window-level keyboard shortcuts (to capture before browser handles them)
	$effect(() => {
		function handleGlobalKeydown(event: KeyboardEvent) {
			// Skip if modal is open
			if (modalOpen) return;

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

			// Build shortcut key string
			const parts: string[] = [];
			if (event.ctrlKey || event.metaKey) parts.push('Ctrl');
			if (event.altKey) parts.push('Alt');
			if (event.shiftKey) parts.push('Shift');
			let key = event.key;
			if (key.length === 1) key = key.toUpperCase();
			parts.push(key);
			const shortcutKey = parts.join('+');

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

	// Reset selection when search query changes
	$effect(() => {
		// Depend on searchQuery
		searchQuery;
		// Reset to first row if current selection is out of bounds
		const filtered = filteredRecords();
		if (selectedIndex >= filtered.length) {
			selectedIndex = filtered.length > 0 ? 0 : -1;
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

	// Get selected record
	const selectedRecord = $derived(
		selectedIndex >= 0 && selectedIndex < records.length ? records[selectedIndex] : null
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

	// Handle new record - insert blank row below current selection
	function handleNew() {
		// Initialize editable records if not already active
		if (!editableActive) {
			editableRecords = records.map(r => ({ ...r }));
			editableActive = true;
		}

		// If an empty new row already exists, just focus it instead of creating another
		const existingNewRowIndex = editableRecords.findIndex(r => isEmptyNewRecord(r));
		if (existingNewRowIndex >= 0) {
			selectedIndex = existingNewRowIndex;
			currentCellRow = existingNewRowIndex;
			currentCellCol = 0;
			cellState = 'cell-selected';
			focusCellSelectedElement(currentCellRow, currentCellCol);
			return;
		}

		// Create a new empty record with all fields initialized to empty strings
		// This ensures all PK fields are defined (important for delayed insert with composite keys)
		const newRecord: Record<string, any> = {
			_isNew: true,
			_tempId: `new-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`
		};
		if (page.page.layout.repeater?.fields) {
			for (const f of page.page.layout.repeater.fields) {
				newRecord[f.source] = '';
			}
		}

		// Insert below the currently selected row (or at end if none selected)
		const insertIndex = selectedIndex >= 0 ? selectedIndex + 1 : editableRecords.length;
		editableRecords = [
			...editableRecords.slice(0, insertIndex),
			newRecord,
			...editableRecords.slice(insertIndex)
		];

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
			editableRecords = records.map(r => ({ ...r }));
			editableActive = true;
		}

		if (row >= editableRecords.length) return;

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
		const record = editableRecords[currentCellRow];
		if (!field || !record) return;

		// Snapshot current value for Escape revert
		cellEditSnapshot = record[field.source];

		if (clearContent) {
			record[field.source] = typedChar ?? '';
			editableRecords = [...editableRecords];
		}

		cellState = 'cell-editing';
		focusCell(currentCellRow, currentCellCol, !clearContent);
	}

	// Exit cell-editing back to cell-selected
	function exitEditingToCellSelected(revert: boolean) {
		if (cellState !== 'cell-editing') return;

		if (revert && cellEditSnapshot !== undefined) {
			const cols = visibleColumns();
			const field = cols[currentCellCol];
			if (field && editableRecords[currentCellRow]) {
				editableRecords[currentCellRow][field.source] = cellEditSnapshot;
				editableRecords = [...editableRecords];
			}
		}

		cellState = 'cell-selected';
		cellEditSnapshot = undefined;
		focusCellSelectedElement(currentCellRow, currentCellCol);
	}

	// Exit to navigation mode
	function exitToNavigation() {
		// Clean up empty new rows
		cleanupEmptyNewRows();

		cellState = 'navigation';
		currentCellRow = -1;
		currentCellCol = -1;
		cellEditSnapshot = undefined;
		editableRecords = [];
		editableActive = false;

		listPageElement?.focus();
	}

	// Confirm current cell value and move to target cell (enters cell-selected at target)
	async function confirmAndMoveTo(targetRow: number, targetCol: number) {
		const cols = visibleColumns();
		const prevRow = currentCellRow;
		const prevCol = currentCellCol;
		const record = editableRecords[prevRow];

		// Immediately transition state to prevent blur handler from interfering
		cellState = 'cell-selected';
		cellEditSnapshot = undefined;

		// Apply Code uppercase and save the previous cell
		if (record) {
			const field = cols[prevCol];
			if (field && fieldTypes[field.source] === 'code' && typeof record[field.source] === 'string') {
				record[field.source] = record[field.source].toUpperCase();
			}
			const leavingRow = targetRow !== prevRow;
			await handleCellBlur(record, prevRow, field?.source, leavingRow);
		}

		// Clean up empty new row if leaving it
		let adjustedTargetRow = targetRow;
		if (record && isEmptyNewRecord(record) && targetRow !== prevRow) {
			editableRecords = editableRecords.filter((_, i) => i !== prevRow);
			if (targetRow > prevRow) {
				adjustedTargetRow--;
			}
		}

		// Clamp target to valid range
		adjustedTargetRow = Math.max(0, Math.min(adjustedTargetRow, editableRecords.length - 1));
		const adjustedTargetCol = Math.max(0, Math.min(targetCol, cols.length - 1));

		// Enter cell-selected at target
		enterCellSelected(adjustedTargetRow, adjustedTargetCol);
	}

	// Track saving state to prevent concurrent saves
	let isSaving = $state(false);
	let pendingSave: { record: Record<string, any>; rowIndex: number } | null = null;

	// Auto-save when leaving a cell
	// forceInsert: skip delayed insert check (used when we know we're leaving the row but DOM focus hasn't moved yet)
	async function handleCellBlur(record: Record<string, any>, rowIndex: number, fieldName?: string, forceInsert: boolean = false) {
		if (!page || !editableActive) return;

		// If already saving, queue this save for later (must check before async validation)
		if (isSaving) {
			pendingSave = { record, rowIndex };
			return;
		}

		isSaving = true;

		// Validate table_relation fields: check if the value exists in the related table
		// Skip validation for fields using LookupDropdown (it validates internally)
		if (fieldName) {
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
			// Check if this is a new record (has _isNew flag)
			const isNew = record._isNew === true;
			const recordId = getRecordId(record, primaryKeyField, primaryKeyFieldsList);

			if (isNew) {
				// NAV/BC delayed insert: only insert when the user LEAVES THE ROW.
				// Keyboard handlers pass forceInsert=true when moving to a different row,
				// leaving the table, or pressing Enter on the last row.
				// When forceInsert is false, always defer — the user is still filling fields.
				// This avoids relying on document.activeElement which is unreliable when
				// async validation causes Svelte to re-render (destroying the input mid-await).
				if (!forceInsert) {
					return;
				}

				// Delayed insert: required PK fields must be non-empty,
				// optional PK fields can be blank (e.g., blank company = all companies)
				const allPKsFilled = primaryKeyFieldsList.length === 0 || primaryKeyFieldsList.every(pk => {
					const fieldDef = page.page.layout.repeater?.fields?.find(f => f.source === pk);
					if (fieldDef?.required) {
						return record[pk] !== undefined && record[pk] !== '';
					}
					return record[pk] !== undefined;
				});
				if (hasRecordData(record) && allPKsFilled) {
					// Remove temporary flags before saving
					const { _isNew, _tempId, ...recordToSave } = record;
					const savedRecord = await api.insertRecord(page.page.source_table, recordToSave);
					// Update record in place to preserve _tempId (keeps Svelte's keyed each stable)
					// Remove _isNew flag since it's now saved, but keep _tempId for stable rendering
					if (savedRecord) Object.assign(editableRecords[rowIndex], savedRecord, { _tempId });
					delete editableRecords[rowIndex]._isNew;
					// Trigger parent update if callback exists
					if (onsave) {
						await onsave(savedRecord, true);
					}
				}
			} else if (recordId) {
				// Existing record - update it
				const { _isNew, _tempId, ...recordToSave } = record;
				const savedRecord = await api.modifyRecord(page.page.source_table, recordId, recordToSave);
				// Update record in place to preserve any _tempId
				if (savedRecord) Object.assign(editableRecords[rowIndex], savedRecord);
				if (_tempId) editableRecords[rowIndex]._tempId = _tempId;
				// Trigger parent update if callback exists
				if (onsave) {
					await onsave(savedRecord, false);
				}
			}
		} catch (err) {
			console.error('Error saving cell:', err);
			const message = err instanceof Error ? err.message : t(ERR.FAILED_SAVE_RECORD);
			toast.error(message);
			// Revert the cell to its original value
			const originalRecord = records.find(r => getRecordId(r, primaryKeyField, primaryKeyFieldsList) === getRecordId(record, primaryKeyField, primaryKeyFieldsList));
			if (originalRecord) {
				// Existing record - revert to original values but keep temp flags
				const tempFlags = { _tempId: editableRecords[rowIndex]?._tempId };
				editableRecords[rowIndex] = { ...deepCopy(originalRecord), ...tempFlags };
			}
			// For new records without an original, the invalid value stays but won't be saved
		} finally {
			isSaving = false;
			// Process any pending save
			if (pendingSave) {
				const { record: pendingRecord, rowIndex: pendingRowIndex } = pendingSave;
				pendingSave = null;
				// Use setTimeout to avoid stack overflow
				setTimeout(() => handleCellBlur(pendingRecord, pendingRowIndex), 0);
			}
		}
	}

	// Check if a record is an empty new record (marked as new and has no user data)
	function isEmptyNewRecord(record: Record<string, any>): boolean {
		return record._isNew === true && isEmptyRecord(record);
	}

	// Remove empty new rows from editableRecords
	function cleanupEmptyNewRows(exceptRowIndex?: number) {
		const indicesToRemove: number[] = [];
		editableRecords.forEach((record, index) => {
			if (index !== exceptRowIndex && isEmptyNewRecord(record)) {
				indicesToRemove.push(index);
			}
		});
		if (indicesToRemove.length > 0) {
			editableRecords = editableRecords.filter((_, index) => !indicesToRemove.includes(index));
			// Adjust current row if needed
			const removedBefore = indicesToRemove.filter(i => i < currentCellRow).length;
			if (removedBefore > 0) {
				currentCellRow = Math.max(0, currentCellRow - removedBefore);
			}
		}
	}

	// Insert a new row at cursor position
	function insertNewRow(atEnd: boolean = false) {
		if (!editableActive) return;

		// Don't create a new row if we're already on an empty new row
		if (currentCellRow >= 0 && currentCellRow < editableRecords.length) {
			const currentRecord = editableRecords[currentCellRow];
			if (isEmptyNewRecord(currentRecord)) {
				// Already on an empty new row, just focus it
				currentCellCol = 0;
				cellState = 'cell-selected';
				focusCellSelectedElement(currentCellRow, currentCellCol);
				return;
			}
		}

		// Clean up any other empty new rows first
		cleanupEmptyNewRows();

		// Create a new empty record with all fields initialized to empty strings
		const newRecord: Record<string, any> = {
			_isNew: true,
			_tempId: `new-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`
		};
		if (page.page.layout.repeater?.fields) {
			for (const f of page.page.layout.repeater.fields) {
				newRecord[f.source] = '';
			}
		}

		// Insert at end, or at current cursor position
		const insertIndex = atEnd ? editableRecords.length : (currentCellRow >= 0 ? currentCellRow : editableRecords.length);
		editableRecords = [
			...editableRecords.slice(0, insertIndex),
			newRecord,
			...editableRecords.slice(insertIndex)
		];

		// Focus the first cell of the new row in cell-selected mode
		currentCellRow = insertIndex;
		currentCellCol = 0;
		cellState = 'cell-selected';
		focusCellSelectedElement(currentCellRow, currentCellCol);
	}

	// Handle keyboard in cell-selected mode
	function handleCellSelectedKeyDown(event: KeyboardEvent, rowIndex: number, colIndex: number) {
		const cols = visibleColumns();
		const record = editableRecords[rowIndex];
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

		// Ctrl+Insert or Ctrl+N to insert new row
		if ((event.key === 'Insert' || event.key === 'n') && event.ctrlKey) {
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
				if (rowIndex > 0) {
					confirmAndMoveTo(rowIndex - 1, colIndex);
				}
				break;
			case 'ArrowDown':
				event.preventDefault();
				if (rowIndex < editableRecords.length - 1) {
					confirmAndMoveTo(rowIndex + 1, colIndex);
				}
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
					} else if (rowIndex > 0) {
						confirmAndMoveTo(rowIndex - 1, cols.length - 1);
					}
				} else {
					// Move right, wrap to next row
					if (colIndex < cols.length - 1) {
						confirmAndMoveTo(rowIndex, colIndex + 1);
					} else if (rowIndex < editableRecords.length - 1) {
						confirmAndMoveTo(rowIndex + 1, 0);
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
				if (rowIndex < editableRecords.length - 1) {
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
				// Copy from cell above
				event.preventDefault();
				if (rowIndex > 0) {
					const aboveRecord = editableRecords[rowIndex - 1];
					record[field.source] = aboveRecord[field.source];
					editableRecords = [...editableRecords];
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

		// Ctrl+Insert or Ctrl+N to insert new row
		if ((event.key === 'Insert' || event.key === 'n') && event.ctrlKey) {
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

					if (shouldNavigate && rowIndex > 0) {
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
						if (rowIndex < editableRecords.length - 1) {
							confirmAndMoveTo(rowIndex + 1, colIndex);
						} else {
							const currentRecord = editableRecords[rowIndex];
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
					} else if (rowIndex > 0) {
						confirmAndMoveTo(rowIndex - 1, cols.length - 1);
					}
				} else {
					if (colIndex < cols.length - 1) {
						confirmAndMoveTo(rowIndex, colIndex + 1);
					} else if (rowIndex < editableRecords.length - 1) {
						confirmAndMoveTo(rowIndex + 1, 0);
					}
				}
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
					if (rowIndex > 0) {
						const field = cols[colIndex];
						const aboveRecord = editableRecords[rowIndex - 1];
						const currentRecord = editableRecords[rowIndex];
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

						// Trigger reactivity
						editableRecords = [...editableRecords];
					}
				}
				break;
			case 'Enter':
				event.preventDefault();
				if (rowIndex < editableRecords.length - 1) {
					confirmAndMoveTo(rowIndex + 1, colIndex);
				} else {
					const currentRecord = editableRecords[rowIndex];
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
					} else if (rowIndex > 0) {
						confirmAndMoveTo(rowIndex - 1, cols.length - 1);
					}
				} else {
					if (colIndex < cols.length - 1) {
						confirmAndMoveTo(rowIndex, colIndex + 1);
					} else if (rowIndex < editableRecords.length - 1) {
						confirmAndMoveTo(rowIndex + 1, 0);
					}
				}
				break;
			case 'Enter':
				// Only handle Enter when dropdown is closed (LookupDropdown preventDefault's Enter when open)
				event.preventDefault();
				if (rowIndex < editableRecords.length - 1) {
					confirmAndMoveTo(rowIndex + 1, colIndex);
				} else {
					const currentRecord = editableRecords[rowIndex];
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
	function focusCell(rowIndex: number, colIndex: number, selectAll: boolean = true) {
		// Use a longer timeout to ensure Svelte has finished any re-renders
		setTimeout(() => {
			// Try direct input/select first (regular inputs and simple lookups)
			const input = document.querySelector(
				`input[data-row="${rowIndex}"][data-col="${colIndex}"], select[data-row="${rowIndex}"][data-col="${colIndex}"]`
			) as HTMLInputElement | HTMLSelectElement | null;
			if (input) {
				input.focus();
				if (input instanceof HTMLInputElement) {
					if (selectAll) {
						input.select();
					} else {
						const len = input.value?.length || 0;
						input.setSelectionRange(len, len);
					}
				}
				return;
			}
			// Try container div (LookupDropdown/OptionDropdown wrapper) and focus the input inside
			const container = document.querySelector(
				`div[data-row="${rowIndex}"][data-col="${colIndex}"]`
			);
			if (container) {
				const innerInput = container.querySelector('input') as HTMLInputElement | null;
				if (innerInput) {
					innerInput.focus();
					if (selectAll) {
						innerInput.select();
					} else {
						const len = innerInput.value?.length || 0;
						innerInput.setSelectionRange(len, len);
					}
				} else {
					// OptionDropdown uses a focusable div[role="combobox"] trigger
					const combobox = container.querySelector('[role="combobox"]') as HTMLElement | null;
					if (combobox) {
						combobox.focus();
					}
				}
			}
		}, 50);
	}

	// Focus a cell-selected element (div with data-cell-row/data-cell-col)
	function focusCellSelectedElement(row: number, col: number) {
		setTimeout(() => {
			const el = document.querySelector(
				`[data-cell-row="${row}"][data-cell-col="${col}"]`
			) as HTMLElement | null;
			if (el) {
				el.focus();
			}
		}, 50);
	}

	// Handle blur from cell-editing inputs
	function handleEditingInputBlur() {
		const blurredRow = currentCellRow;
		const blurredCol = currentCellCol;

		setTimeout(() => {
			// If state already transitioned (e.g., keyboard/click handler took over), skip
			if (cellState === 'navigation') return;

			// If we've moved to a different cell, the transition was already handled
			if (currentCellRow !== blurredRow || currentCellCol !== blurredCol) return;

			// If focus left the table entirely, save and exit
			if (!listPageElement?.contains(document.activeElement)) {
				const cols = visibleColumns();
				const field = cols[blurredCol];
				const record = editableRecords[blurredRow];
				if (record && field) {
					if (fieldTypes[field.source] === 'code' && typeof record[field.source] === 'string') {
						record[field.source] = record[field.source].toUpperCase();
					}
					handleCellBlur(record, blurredRow, field.source, true);
				}
				exitToNavigation();
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

	// Toggle sort on a column
	function handleSort(fieldSource: string) {
		if (sortField === fieldSource) {
			// Toggle direction if same field
			sortDirection = sortDirection === 'asc' ? 'desc' : 'asc';
		} else {
			// New field, start with ascending
			sortField = fieldSource;
			sortDirection = 'asc';
		}
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

		<div class="table-container">
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
							record._isNew && 'new-row'
						)}
					>
						{#if showRowNumbers}
							<td class="row-number-cell">{index + 1}</td>
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
											<span class="cell-selected-lookup-value">
												{#if lookups[field.source]?.rows?.length}
													{formatLookupValue(record[field.source], lookups[field.source])}
												{:else}
													{formatCellValue(record[field.source], field.source)}
												{/if}
											</span>
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
											<span class="cell-selected-lookup-value">
												{formatOptionValue(record[field.source], options[field.source])}
											</span>
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
											{formatCellValue(record[field.source], field.source)}
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
													onclick={(e) => {
														e.stopPropagation();
														record[field.source] = !record[field.source];
														handleCellBlur(record, index);
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
		</tbody>
		</table>
		</div>
	</div>

	<div class="status-bar">
		<span class="text-sm text-gray-600 dark:text-gray-400">
			{#if searchQuery}
				{displayRecords.length} of {records.length} record{records.length !== 1 ? 's' : ''} (filtered)
			{:else}
				{records.length} record{records.length !== 1 ? 's' : ''}
			{/if}
			{#if isNavigation && selectedIndex >= 0 && selectedIndex < displayRecords.length}
				• Row {selectedIndex + 1} selected
			{:else if !isNavigation && currentCellRow >= 0 && currentCellCol >= 0}
				• Cell [{currentCellRow + 1}, {currentCellCol + 1}] {isCellEditing ? '(editing)' : '(selected)'}
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

	.table th {
		@apply px-4 py-3 text-left text-sm font-semibold;
		@apply bg-nav-blue text-white;
		@apply dark:bg-gray-800;
		border-right: 1px solid rgba(255, 255, 255, 0.1);
		border-bottom: 1px solid rgba(255, 255, 255, 0.2);
		position: sticky;
		top: 0;
		z-index: 10;
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
		color: #6b7280;
		border-right: 1px solid #d1d5db;
		border-bottom: 1px solid #d1d5db;
	}

	:global(.dark) .row-number-cell {
		color: white;
		background-color: rgb(31 41 55); /* gray-800 - matches normal columns */
		border-color: #4b5563; /* gray-600 */
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
		background: rgba(59, 130, 246, 0.5);
	}

	.resize-handle:active {
		background: rgba(59, 130, 246, 0.8);
	}

	.th-content {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 4px;
	}

	.th-label {
		flex: 1;
		color: white;
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

	/* Zebra striping - alternating row colors */
	.table tbody tr:nth-child(even) {
		@apply bg-gray-50;
		@apply dark:bg-gray-800/50;
	}

	.table tbody tr:nth-child(odd) {
		@apply bg-white;
		@apply dark:bg-gray-900;
	}

	.table tbody tr.selected {
		background-color: #dbeafe !important; /* bg-blue-100 */
	}

	.table tbody tr.selected:hover {
		background-color: #dbeafe !important;
	}

	:global(.dark) .table tbody tr.selected {
		background-color: #1e40af !important; /* bg-blue-800 */
		color: white;
	}

	:global(.dark) .table tbody tr.selected:hover {
		background-color: #1e40af !important;
	}

	:global(.dark) .table tbody tr.selected td,
	:global(.dark) .table tbody tr.selected .read-cell-content,
	:global(.dark) .table tbody tr.selected .primary-key-link {
		color: white !important;
	}

	.table tbody tr.new-row {
		background-color: #e0f2fe !important;
	}

	:global(.dark) .table tbody tr.new-row {
		background-color: #1e3a5f !important; /* Dark blue background for new rows */
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
		background-color: #1f2937; /* gray-800 */
		border-color: #374151; /* gray-700 */
		color: #d1d5db; /* gray-300 */
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
		outline: none;
	}

	.cell-selected-active {
		outline: 2px solid #2563eb; /* blue-600 border */
		outline-offset: -2px; /* inset so it doesn't shift layout */
		background: #eff6ff; /* blue-50 subtle highlight */
	}

	:global(.dark) .cell-selected-active {
		outline-color: #3b82f6;
		background: rgba(59, 130, 246, 0.1);
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
		color: #6b7280;
		cursor: pointer;
		line-height: 1;
	}

	.cell-selected-lookup-arrow:hover {
		color: #2563eb;
	}

	:global(.dark) .cell-selected-lookup-arrow {
		color: #9ca3af;
	}

	:global(.dark) .cell-selected-lookup-arrow:hover {
		color: #60a5fa;
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

	/* Quick Search Styles */
	.search-container {
		@apply relative flex items-center;
		margin-left: 1rem;
	}

	.search-icon {
		@apply absolute left-3 w-4 h-4 text-gray-400 pointer-events-none;
	}

	:global(.dark) .search-icon {
		color: #9ca3af;
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
		background-color: #374151;
		border-color: #4b5563;
		color: white;
	}

	:global(.dark) .search-input::placeholder {
		color: #9ca3af;
	}

	:global(.dark) .search-input:focus {
		border-color: #3b82f6;
		box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.3);
	}

	.clear-search-btn {
		@apply absolute right-2 p-0.5 rounded text-gray-400 hover:text-gray-600;
		@apply hover:bg-gray-100 transition-colors;
	}

	:global(.dark) .clear-search-btn {
		color: #9ca3af;
	}

	:global(.dark) .clear-search-btn:hover {
		color: #d1d5db;
		background-color: #4b5563;
	}
</style>
