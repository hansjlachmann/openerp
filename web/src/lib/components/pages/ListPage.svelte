<script lang="ts">
	import type { PageDefinition, Field } from '$lib/types/pages';
	import type { TableFilter } from '$lib/types/api';
	import { toast } from '$lib/stores/toast';
	import { confirm } from '$lib/stores/confirm';
	import Button from '$lib/components/Button.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import ModalCardPage from './ModalCardPage.svelte';
	import CustomizeFieldsModal from './CustomizeFieldsModal.svelte';
	import FilterPane from './FilterPane.svelte';
	import ConfirmModal from '../ConfirmModal.svelte';
	import PlusIcon from '$lib/components/icons/PlusIcon.svelte';
	import EditIcon from '$lib/components/icons/EditIcon.svelte';
	import TrashIcon from '$lib/components/icons/TrashIcon.svelte';
	import RefreshIcon from '$lib/components/icons/RefreshIcon.svelte';
	import { shortcuts } from '$lib/utils/shortcuts';
	import { cn } from '$lib/utils/cn';
	import { api } from '$lib/services/api';
	import { currentUser } from '$lib/stores/user';
	import { getFieldCaption, getFieldStyleClasses, formatValue, isItemVisible, type ItemCustomization } from '$lib/utils/fieldHelpers';
	import { loadPageCustomizations, savePageCustomizations, loadColumnWidths, saveColumnWidths, loadRowNumbersPreference, saveRowNumbersPreference } from '$lib/utils/customizationStorage';
	import { getRecordId, deepCopy, hasRecordChanged } from '$lib/utils/recordHelpers';

	// Lookup data structure from API
	interface LookupData {
		columns?: Array<{ source: string; width: number }>;
		rows?: Array<{ _key: string; [key: string]: any }>;
		simple?: Record<string, string>;
	}

	interface Props {
		page: PageDefinition;
		records?: Array<Record<string, any>>;
		captions?: Record<string, string>;
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
		options = {},
		lookups = {},
		currentFilters = [],
		onaction,
		onrowclick,
		onsave,
		ondelete,
		onfilter
	}: Props = $props();

	// Helper to format option field values
	function formatOptionValue(fieldSource: string, value: any): string {
		const fieldOptions = options[fieldSource];
		if (fieldOptions && value !== undefined && value !== null) {
			const stringValue = String(value);
			return fieldOptions[stringValue] || String(value);
		}
		return formatValue(value);
	}

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

	// Edit mode state
	let editMode = $state(false);
	// Editable copy of records for edit mode (to avoid mutating props)
	let editableRecords = $state<Array<Record<string, any>>>([]);
	// Prevent rapid toggling
	let isToggling = false;


	// Filter records by search query
	const filteredRecords = $derived(() => {
		const sourceRecords = editMode ? editableRecords : records;
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

		return [...sourceRecords].sort((a, b) => {
			const aVal = a[sortField];
			const bVal = b[sortField];

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
		if (listPageElement && !editMode && records.length > 0 && !modalOpen) {
			setTimeout(() => {
				listPageElement?.focus();
			}, 100);
		}
	});

	// Window-level keyboard shortcuts (to capture before browser handles them)
	$effect(() => {
		function handleGlobalKeydown(event: KeyboardEvent) {
			// Skip if modal is open or we're in an input field
			if (modalOpen) return;
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

			// Check if this matches any action shortcut
			const action = page.page.actions?.find(a => a.shortcut === shortcutKey);
			if (action) {
				event.preventDefault();
				event.stopPropagation();
				handleAction(action.name);
			}
		}

		window.addEventListener('keydown', handleGlobalKeydown, true); // capture phase
		return () => window.removeEventListener('keydown', handleGlobalKeydown, true);
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

	// Handle action clicks
	async function handleAction(actionName: string) {
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
				// Fallback to inline edit mode toggle
				toggleEditMode();
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
		// Enter edit mode if not already
		if (!editMode) {
			editableRecords = records.map(r => ({ ...r }));
			editMode = true;
		}

		// Create a new empty record with temporary flag and stable ID for keying
		const newRecord: Record<string, any> = {
			_isNew: true,
			_tempId: `new-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`
		};

		// Insert below the currently selected row (or at end if none selected)
		const insertIndex = selectedIndex >= 0 ? selectedIndex + 1 : editableRecords.length;
		editableRecords = [
			...editableRecords.slice(0, insertIndex),
			newRecord,
			...editableRecords.slice(insertIndex)
		];

		// Update selection to the new row
		selectedIndex = insertIndex;

		// Focus the first cell of the new row
		currentCellRow = insertIndex;
		currentCellCol = 0;
		focusCell(currentCellRow, currentCellCol);
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
			// Check if this is a new record (has _isNew flag)
			const isNew = record._isNew === true;
			const recordId = getRecordId(record);

			// Remember current focus position before save
			const focusRow = currentCellRow;
			const focusCol = currentCellCol;

			if (isNew) {
				// New record - only save if user has entered some data
				const hasData = Object.keys(record).some(key => !key.startsWith('_') && record[key] !== undefined && record[key] !== '');
				if (hasData) {
					// Remove temporary flags before saving
					const { _isNew, _tempId, ...recordToSave } = record;
					const savedRecord = await api.insertRecord(page.page.source_table, recordToSave);
					// Update record in place to preserve _tempId (keeps Svelte's keyed each stable)
					// Remove _isNew flag since it's now saved, but keep _tempId for stable rendering
					Object.assign(editableRecords[rowIndex], savedRecord, { _tempId });
					delete editableRecords[rowIndex]._isNew;
					// Re-focus after save to maintain cursor position
					if (focusRow >= 0 && focusCol >= 0) {
						focusCell(focusRow, focusCol);
					}
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
				Object.assign(editableRecords[rowIndex], savedRecord);
				if (_tempId) editableRecords[rowIndex]._tempId = _tempId;
				// Re-focus after save to maintain cursor position
				if (focusRow >= 0 && focusCol >= 0) {
					focusCell(focusRow, focusCol);
				}
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

		// Create a new empty record with temporary flag and stable ID for keying
		const newRecord: Record<string, any> = {
			_isNew: true,
			_tempId: `new-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`
		};

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
		// Use a longer timeout to ensure Svelte has finished any re-renders
		setTimeout(() => {
			const input = document.querySelector(
				`input[data-row="${rowIndex}"][data-col="${colIndex}"]`
			) as HTMLInputElement;
			if (input) {
				input.focus();
				input.select();
			}
		}, 50);
	}

	// Handle delete record
	async function handleDelete() {
		if (selectedRecord) {
			confirm.show(
				'Delete Record',
				'Are you sure you want to delete this record?',
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
				throw new Error(`Failed to load card page: ${response.statusText}`);
			}

			const result = await response.json();
			if (!result.success) {
				throw new Error(result.error || 'Failed to load card page');
			}

			modalCardPage = result.data;
			modalCaptions = result.captions?.fields || {};

			// Use the card page's source table (more reliable)
			const sourceTable = modalCardPage?.page?.source_table || page.page.source_table;

			// Load the record data with options and lookups (for enum/lookup dropdowns)
			const recordId = getRecordId(record);
			if (recordId) {
				const recordResult = await api.getRecordWithCaptions(sourceTable, recordId);
				modalRecord = recordResult.data;
				modalOptions = recordResult.captions?.options || {};
				modalLookups = recordResult.captions?.lookups || {};
				modalOriginalRecord = deepCopy(modalRecord);
				modalIsNewRecord = false;
			} else {
				// New record - open modal immediately, fetch options/lookups in background
				modalRecord = { ...record };
				modalOriginalRecord = {};
				modalOptions = {}; // Start with empty options
				modalLookups = {}; // Start with empty lookups
				modalOptionsLoaded = false; // Reset the loaded flag
				modalIsNewRecord = true;

				// Fetch options and lookups in background (non-blocking)
				api.getTableOptionsAndLookups(sourceTable).then(({ options: opts, lookups: lkps }) => {
					// Only update if modal is still open and options haven't been loaded yet
					if (modalOpen && !modalOptionsLoaded) {
						// Save currently focused element
						const activeElement = document.activeElement;
						const activeElementId = activeElement instanceof HTMLElement ? activeElement.id : null;

						modalOptions = opts;
						modalLookups = lkps;
						modalOptionsLoaded = true;

						// Restore focus if it was lost during re-render
						if (activeElementId) {
							setTimeout(() => {
								const element = document.getElementById(activeElementId);
								if (element && document.activeElement !== element) {
									element.focus();
								}
							}, 0);
						}
					}
				}).catch(err => {
					console.error('Failed to load options/lookups:', err);
					modalOptionsLoaded = true; // Mark as loaded even on error to prevent retries
				});
			}

			// Set initial edit mode
			modalInitialEditMode = editMode || modalIsNewRecord;

			modalHadChanges = false;
			modalRecordDeleted = false;
			modalOpen = true;
		} catch (err) {
			console.error('Error opening modal card:', err);
			toast.error('Failed to open card');
		}
	}

	// Close modal
	function closeModal() {
		const hadChanges = modalHadChanges;
		modalOpen = false;
		modalCardPage = null;
		modalRecord = {};
		modalIsNewRecord = false;
		skipNextAutoSave = false;
		modalCaptions = {};
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
			toast.success('Record saved successfully');
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
			const recordId = getRecordId(savedRecord);

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
				const index = records.findIndex(r => getRecordId(r) === recordId);
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
			const message = err instanceof Error ? err.message : 'Failed to save record';
			toast.error(message);

			// Block further edits if this was a new record that failed to save
			// (likely because the record already exists)
			if (modalIsNewRecord) {
				modalSaveBlocked = true;
				modalSaveBlockedMessage = message;
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
			case 'Delete':
				const deleteRecordId = getRecordId(modalRecord);
				if (deleteRecordId && confirm(`Delete this ${modalCardPage.page.caption}?`)) {
					// Mark as deleted BEFORE API call to prevent any pending auto-saves
					modalRecordDeleted = true;

					try {
						await api.deleteRecord(page.page.source_table, deleteRecordId);

						// Remove the record from the list
						records = records.filter(r => getRecordId(r) !== deleteRecordId);

						// Close the modal
						closeModal();

						toast.success('Record deleted successfully');
					} catch (err) {
						console.error('Delete error:', err);
						toast.error('Failed to delete record');
						// Reset flag if delete failed
						modalRecordDeleted = false;
					}
				}
				break;
			case 'Refresh':
				// Reload the modal record with options
				const refreshRecordId = getRecordId(modalRecord);
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

<div class="list-page" use:shortcuts={shortcutMap()} tabindex="0" bind:this={listPageElement} onkeydown={handleSearchShortcut}>
	<PageHeader title={page.page.caption}>
		<svelte:fragment slot="leftActions">
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
						placeholder="Search... (Ctrl+F)"
						bind:value={searchQuery}
						bind:this={searchInputElement}
					/>
					{#if searchQuery}
						<button
							type="button"
							class="clear-search-btn"
							onclick={clearSearch}
							title="Clear search"
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
		</svelte:fragment>

		<svelte:fragment slot="rightActions">
			<!-- Row Numbers toggle button -->
			<Button
					variant={showRowNumbers ? 'primary' : 'secondary'}
					size="sm"
					onclick={handleToggleRowNumbers}
					title="Toggle row numbers"
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
							<div
								class="resize-handle"
								onmousedown={(e) => handleResizeStart(e, field.source, getColumnWidth(field))}
							></div>
						</th>
					{/each}
				</tr>
			</thead>
			<tbody bind:this={tableBodyElement}>
				{#each displayRecords as record, index (record._tempId || record.code || record.no || record.id || index)}
					<tr
						class={cn(
							editMode ? '' : 'cursor-pointer',
							selectedIndex === index && 'selected',
							record._isNew && 'new-row'
						)}
						onclick={() => !editMode && handleRowClick(index)}
					>
						{#if showRowNumbers}
							<td class="row-number-cell">{index + 1}</td>
						{/if}
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
										{:else if options[field.source]}
											<!-- Option field - always show dropdown -->
											<select
												class="option-select"
												value={String(record[field.source] ?? 0)}
												onchange={async (e) => {
													const target = e.target as HTMLSelectElement;
													const newValue = parseInt(target.value, 10);
													record[field.source] = newValue;
													await handleCellBlur(record, index);
												}}
												onclick={(e) => e.stopPropagation()}
											>
												{#each Object.entries(options[field.source]) as [optValue, optLabel]}
													<option value={optValue}>{optLabel}</option>
												{/each}
											</select>
										{:else}
											{formatOptionValue(field.source, record[field.source])}
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
			{#if selectedIndex >= 0 && selectedIndex < displayRecords.length}
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
	confirmText="Delete"
	variant="danger"
	onconfirm={confirm.confirm}
	oncancel={confirm.cancel}
/>

<style>
	.list-page {
		@apply flex flex-col gap-4 h-full;
	}

	.list-content {
		@apply flex flex-1 gap-4 min-h-0;
	}

	.table-container {
		@apply flex-1 overflow-auto border border-gray-200 rounded-lg bg-white;
		@apply dark:border-gray-700 dark:bg-gray-900;
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
		position: relative;
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
		color: white;
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
		right: 0;
		top: 0;
		bottom: 0;
		width: 6px;
		cursor: col-resize;
		background: transparent;
		z-index: 1;
	}

	.resize-handle:hover {
		background: rgba(255, 255, 255, 0.3);
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

	/* Option field dropdown in list */
	.option-select {
		@apply w-full text-sm;
		@apply bg-transparent border-0 rounded;
		@apply cursor-pointer;
		@apply outline-none;
		appearance: none;
		padding: 2px 1.5rem 2px 6px;
		line-height: 1.3;
		background-image: url("data:image/svg+xml,%3csvg xmlns='http://www.w3.org/2000/svg' fill='none' viewBox='0 0 20 20'%3e%3cpath stroke='%236b7280' stroke-linecap='round' stroke-linejoin='round' stroke-width='1.5' d='M6 8l4 4 4-4'/%3e%3c/svg%3e");
		background-position: right 0.25rem center;
		background-repeat: no-repeat;
		background-size: 1em 1em;
	}

	.option-select:hover {
		@apply bg-gray-100;
	}

	.option-select:focus {
		@apply bg-white ring-2 ring-blue-500/30;
	}

	:global(.dark) .option-select {
		color: var(--color-text-primary);
		background-image: url("data:image/svg+xml,%3csvg xmlns='http://www.w3.org/2000/svg' fill='none' viewBox='0 0 20 20'%3e%3cpath stroke='%239ca3af' stroke-linecap='round' stroke-linejoin='round' stroke-width='1.5' d='M6 8l4 4 4-4'/%3e%3c/svg%3e");
	}

	:global(.dark) .option-select:hover {
		background-color: rgba(255, 255, 255, 0.1);
	}

	:global(.dark) .option-select:focus {
		background-color: var(--color-bg-input);
	}

	:global(.dark) .option-select option {
		background-color: var(--color-bg-primary);
		color: var(--color-text-primary);
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
