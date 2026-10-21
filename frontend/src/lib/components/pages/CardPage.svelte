<script lang="ts">
	import { goto } from '$app/navigation';
	import type { PageDefinition, Field, FieldState, Action } from '$lib/types/pages';
	import type { LookupData } from '$lib/types/api';
	import FieldRenderer from './FieldRenderer.svelte';
	import Button from '$lib/components/Button.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import Card from '$lib/components/Card.svelte';
	import CustomizeFieldsModal from './CustomizeFieldsModal.svelte';
	import PlusIcon from '$lib/components/icons/PlusIcon.svelte';
	import NavigationButtons from '$lib/components/NavigationButtons.svelte';
	import { shortcuts, createShortcutMap, normalizeShortcut } from '$lib/utils/shortcuts';
	import { currentUser } from '$lib/stores/user';
	import { cardTitle, getFieldCaption, isItemVisible, type ItemCustomization } from '$lib/utils/fieldHelpers';
	import { loadPageCustomizations, savePageCustomizations } from '$lib/utils/customizationStorage';
	import { getRecordId, isNewRecord, deepCopy, getPrimaryKeyField, getPrimaryKeyFields, stripInternalFields } from '$lib/utils/recordHelpers';
	import { withReturn } from '$lib/utils/returnUrl';
	import { toast } from '$lib/stores/toast';
	import { api } from '$lib/services/api';
	import { t, ERR, MSG, LIST, CARD } from '$lib/services/i18n.svelte';
	import { handleFieldEnterKey } from '$lib/utils/fieldNavigation';
	import { getJson, setJson } from '$lib/utils/storage';
	import PageRenderer from './PageRenderer.svelte';
	import ActionButton from '../ActionButton.svelte';
	import ActionMenu from '../ActionMenu.svelte';
	import ObjectRunner from '../ObjectRunner.svelte';
	import type { Subpage } from '$lib/types/pages';
	import type { FixedFilter } from '$lib/types/api';

	interface Props {
		page: PageDefinition;
		record?: Record<string, any>;
		captions?: Record<string, string>;
		fieldTypes?: Record<string, string>; // Field type metadata (e.g., "bool", "code", "text")
		options?: Record<string, Record<string, string>>; // Option field values (enum lookups)
		lookups?: Record<string, LookupData>; // Table relation lookup values
		onaction?: (actionName: string) => void;
		onsave?: (record: Record<string, any>) => Promise<boolean> | boolean | void;
		onclose?: () => void; // Callback when closing the card
		// A card opened over its list (modal card): going back to the list closes it
		onbacktolist?: () => void;
		saveBlocked?: boolean; // Block editing due to save error (e.g., record already exists)
		saveBlockedMessage?: string; // Error message to display when blocked
		onclearerror?: () => void; // Callback to clear the error state
		navigationEnabled?: boolean;
		initialEditMode?: boolean;
		canNavigateFirst?: boolean;
		canNavigatePrevious?: boolean;
		canNavigateNext?: boolean;
		canNavigateLast?: boolean;
		onNavigateFirst?: () => void;
		onNavigatePrevious?: () => void;
		onNavigateNext?: () => void;
		onNavigateLast?: () => void;
		// A subpage's lines changed: the card refreshes its FlowFields (totals)
		onrefresh?: () => void;
	}

	let {
		page,
		record = $bindable({}),
		captions = {},
		fieldTypes = {},
		options = {},
		lookups = {},
		onaction,
		onsave,
		onclose,
		onbacktolist,
		saveBlocked = false,
		saveBlockedMessage = '',
		onclearerror,
		navigationEnabled = false,
		initialEditMode,
		canNavigateFirst = false,
		canNavigatePrevious = false,
		canNavigateNext = false,
		canNavigateLast = false,
		onNavigateFirst,
		onNavigatePrevious,
		onNavigateNext,
		onNavigateLast,
		onrefresh
	}: Props = $props();

	// Get primary key field name from page definition
	const primaryKeyField = $derived(getPrimaryKeyField(page));

	// Page triggers (OnAfterGetRecord): fields switched for the current record, asked from
	// the server when the record or a field value changes
	let fieldStates = $state<Record<string, FieldState>>({});
	let fieldStatesRequest = 0;
	$effect(() => {
		if (!page.page.has_record_triggers) return;
		const current = stripInternalFields($state.snapshot(record) ?? {});
		const request = ++fieldStatesRequest;
		const timer = setTimeout(async () => {
			try {
				const states = await api.getFieldStates(page.page.id, current);
				if (request === fieldStatesRequest) fieldStates = states;
			} catch {
				// keep the last states: the page definition still applies
			}
		}, 150);
		return () => clearTimeout(timer);
	});

	// Customization state
	let customizeModalOpen = $state(false);
	let fieldCustomizations = $state<Record<string, ItemCustomization>>({});

	// Reference to the card page element for focus management
	let cardPageElement = $state<HTMLDivElement | null>(null);


	// Check if record is empty (no data loaded)
	const isEmptyRecord = $derived(() => {
		if (!record) return true;
		const keys = Object.keys(record).filter(k => !k.startsWith('_'));
		return keys.length === 0;
	});

	// Check if this is a new record (no ID)
	function checkIsNewRecord(): boolean {
		return isNewRecord(record);
	}

	// Edit mode state - start in edit mode for new records or if initialEditMode is true
	// svelte-ignore state_referenced_locally - initialEditMode is intentionally only used for initialization
	let editMode = $state(initialEditMode ?? checkIsNewRecord());

	// Track if we've already focused the initial field (to avoid refocusing on every state change)
	let initialFocusDone = $state(false);

	// Track the last known record ID to detect when a truly new record is loaded
	let lastRecordId = $state<string | undefined>(undefined);

	// Auto-enable edit mode when a new record is loaded (not when field values change)
	$effect(() => {
		const id = getRecordId(record, primaryKeyField);
		// Only trigger when the record ID actually changes (not on every field change)
		if (id !== lastRecordId) {
			lastRecordId = id;
			if (!id) {
				editMode = true;
				// Reset focus flag only when loading a truly new record
				initialFocusDone = false;
			}
		}
	});

	// Focus the configured field when page opens in edit mode, or focus the card page for keyboard shortcuts
	$effect(() => {
		if (!page?.page) return; // Guard during cleanup
		if (editMode && page.page.focus_field && !initialFocusDone) {
			initialFocusDone = true;
			// Use longer timeout to ensure modal animation completes and DOM is ready
			setTimeout(() => {
				const input = document.getElementById(page.page.focus_field!);
				if (input) {
					input.focus();
					// Select all text if it's an input
					if (input instanceof HTMLInputElement) {
						input.select();
					}
				}
			}, 250);
		} else if (!editMode && !initialFocusDone && cardPageElement) {
			// In non-edit mode, focus the card page element so keyboard shortcuts work
			initialFocusDone = true;
			setTimeout(() => {
				cardPageElement?.focus();
			}, 100);
		}
	});

	// Load customizations from localStorage on mount
	$effect(() => {
		if (!page?.page) return; // Guard during cleanup
		const userId = $currentUser?.user_id || 'anonymous';
		fieldCustomizations = loadPageCustomizations<Record<string, ItemCustomization>>(
			userId,
			page.page.id
		);
	});

	// Auto-save state
	let saveState = $state<'idle' | 'saving' | 'saved'>('idle');
	let saveTimeout: number | null = null;
	let savedTimeout: number | null = null;

	// Track last saved record state for reverting on validation error
	let lastSavedRecord = $state<Record<string, any>>({});

	// Update lastSavedRecord when record is loaded from parent
	$effect(() => {
		const id = getRecordId(record, primaryKeyField);
		// When a record is first loaded (has ID and we don't have it saved yet)
		if (id && (!lastSavedRecord || getRecordId(lastSavedRecord, primaryKeyField) !== id)) {
			lastSavedRecord = deepCopy(record);
		}
	});

	// Auto-save with debouncing
	function autoSave() {
		// Skip if already saving or blocked due to error
		if (saveState === 'saving' || saveBlocked) {
			return;
		}

		// Clear any pending timeouts
		if (saveTimeout) {
			clearTimeout(saveTimeout);
		}
		if (savedTimeout) {
			clearTimeout(savedTimeout);
		}

		// Debounce: wait 300ms after last change before saving
		saveTimeout = setTimeout(async () => {
			// Double-check save state before proceeding
			if (saveState === 'saving') {
				return;
			}

			saveState = 'saving';
			try {
				// onsave returns true if a save actually happened
				const didSave = await onsave?.(record);

				if (didSave) {
					// Update lastSavedRecord after successful save
					lastSavedRecord = deepCopy(record);
					// The header changed: the subpages reload their lines (the server may have
					// changed them, e.g. a new discount)
					linesRefresh++;
					saveState = 'saved';
					// Show "Saved" for 1.5 seconds then hide
					savedTimeout = setTimeout(() => {
						saveState = 'idle';
					}, 1500) as unknown as number;
				} else {
					// No actual save happened (no changes)
					saveState = 'idle';
				}
			} catch (err) {
				saveState = 'idle';
				// Validation error - revert to last saved state
				const errorMessage = err instanceof Error ? err.message : t(ERR.VALIDATION_FAILED);
				toast.error(errorMessage);

				// Revert record to last saved state
				if (lastSavedRecord && Object.keys(lastSavedRecord).length > 0) {
					// Copy each field back to revert changes
					for (const key of Object.keys(record)) {
						if (key in lastSavedRecord) {
							record[key] = lastSavedRecord[key];
						}
					}
				}
			}
		}, 300) as unknown as number;
	}

	// Handle field blur (when user leaves a field)
	function handleFieldBlur() {
		autoSave();
	}

	// Handle action clicks
	function handleAction(actionName: string) {
		// Handle built-in actions locally first
		switch (actionName) {
			case 'Edit':
				toggleEditMode();
				return;
			case 'New':
				handleNew();
				return;
		}

		// Pass all other actions to parent (including Delete)
		onaction?.(actionName);
	}

	// Handle run_object actions (codeunits)
	// URL of an action's run_page, filtered to this record when the action names a filter
	// (run_page_filter_field = record[run_page_filter_value]), e.g. the customer's ledger entries
	function runPageUrl(action: { run_page?: number; run_page_filter_field?: string; run_page_filter_value?: string }): string {
		let url = `/pages/${action.run_page}`;
		const value = action.run_page_filter_value ? record?.[action.run_page_filter_value] : undefined;
		if (action.run_page_filter_field && value !== undefined && value !== null && value !== '') {
			url += `?filter=${encodeURIComponent(`${action.run_page_filter_field}=${value}`)}`;
		}
		// Esc / close on the opened page comes back here (from a modal card: to its list,
		// on this record)
		return withReturn(url, getRecordId(record, getPrimaryKeyField(page), getPrimaryKeyFields(page)));
	}

	// Run a page object with the card's record: a codeunit, a report or a function
	// ("table:Job_Queue.ShowStatus", with arguments "…(0)") — its dialogs and message box
	// come from the ObjectRunner; afterwards the card and its subpages refresh when needed
	async function handleRunObject(runObject: string) {
		if (!record || Object.keys(record).length === 0 || checkIsNewRecord()) {
			toast.error(t(ERR.NO_RECORD_SELECTED));
			return;
		}
		await runner?.run(runObject, stripInternalFields($state.snapshot(record)));
	}

	function handleRunDone(refresh: boolean) {
		if (!refresh) return;
		onaction?.('Refresh');
		linesRefresh++;
	}

	// A card action or menu item: open a page, run an object, or a standard action
	// The action going back to the card's own list (Back to List)
	function isBackToList(action: Action): boolean {
		return !!action.run_page && action.run_page === page.page.list_page_id && !action.run_page_filter_field;
	}

	function runCardAction(action: Action) {
		if (isBackToList(action)) {
			// Over the list (modal card): close the card; a card page: open the list on this
			// record — not as a page to return from (its Esc would come back here)
			if (onbacktolist) onbacktolist();
			else goto(`/pages/${action.run_page}?select=${encodeURIComponent(getRecordId(record, getPrimaryKeyField(page), getPrimaryKeyFields(page)) ?? '')}`);
		} else if (action.run_page) {
			goto(runPageUrl(action));
		} else if (action.run_object) {
			handleRunObject(action.run_object);
		} else if (!action.items) {
			handleAction(action.name);
		}
	}

	// The card's actions with the items of its menu buttons (names and shortcuts)
	function allActions(): Action[] {
		const flat = (actions: Action[] | undefined): Action[] => (actions ?? []).flatMap((a) => [a, ...flat(a.items)]);
		return flat(page.page.actions);
	}

	function handleNew() {
		record = {};
	}

	function toggleEditMode() {
		editMode = !editMode;
	}

	// Build keyboard shortcut map from actions
	const shortcutMap = $derived(() => {
		const map: Record<string, () => void> = {};

		// Guard against page being undefined during cleanup
		if (!page?.page) return map;

		allActions().forEach((action) => {
			if (action.shortcut && action.enabled !== false) {
				// Normalize shortcut (e.g., "Esc" -> "Escape") to match keyboard event key names
				const normalizedShortcut = normalizeShortcut(action.shortcut);
				// Over the list, Esc belongs to the modal (it closes the card once)
				if (onbacktolist && normalizedShortcut === 'Escape') return;
				map[normalizedShortcut] = () => runCardAction(action);
			}
		});

		// Add navigation shortcuts if enabled
		if (navigationEnabled) {
			if (canNavigateFirst) map['Ctrl+Home'] = () => onNavigateFirst?.();
			// On a card with subpages Ctrl+↑/↓ move between the parts (handlePartKeys);
			// Ctrl+Shift+↑/↓ go to the previous/next record on every card
			if (canNavigatePrevious && !hasSubpages) map['Ctrl+ArrowUp'] = () => onNavigatePrevious?.();
			if (canNavigateNext && !hasSubpages) map['Ctrl+ArrowDown'] = () => onNavigateNext?.();
			if (canNavigateLast) map['Ctrl+End'] = () => onNavigateLast?.();
		}

		// Add Escape to close
		if (onclose) {
			map['Escape'] = () => onclose();
		}

		// PgUp/PgDn scroll the sections a screen up/down — also from a field, unless an open
		// dropdown (lookup, option list) used the key
		const scrollPage = (direction: number) => ((event?: KeyboardEvent) => {
			if (event?.defaultPrevented) return false;
			if (!sectionsElement) return false;
			sectionsElement.scrollBy({ top: direction * sectionsElement.clientHeight * 0.85, behavior: 'smooth' });
		}) as () => void;
		map['PageDown'] = scrollPage(1);
		map['PageUp'] = scrollPage(-1);

		return map;
	});

	// Runs the card's page objects (codeunits, reports, functions) with their dialogs
	let runner = $state<ObjectRunner | undefined>(undefined);
	// Bumped after a save or a function asking for it: the subpages reload their lines
	let linesRefresh = $state(0);

	// The scrolling sections (PgUp/PgDn)
	let sectionsElement = $state<HTMLDivElement | null>(null);

	// ---- Subpages: list pages in the card, tied to its record (section.subpage.link) ----
	const hasSubpages = $derived((page.page.layout.sections ?? []).some((s) => !!s.subpage));

	// The link of a subpage for the current record: subpage field → the card field's value
	function linkValues(sub: Subpage): Record<string, any> {
		return Object.fromEntries(Object.entries(sub.link).map(([field, cardField]) => [field, record?.[cardField]]));
	}
	function linkFilters(sub: Subpage): FixedFilter[] {
		return Object.entries(linkValues(sub)).map(([field, value]) => ({ field, value }));
	}
	// Lines need a saved card record with its key
	function canShowLines(sub: Subpage): boolean {
		return !checkIsNewRecord() && Object.values(linkValues(sub)).every((v) => v !== undefined && v !== null && v !== '');
	}

	// Parts of the card: its fields, then each subpage. F6 / Shift+F6 and (cards with subpages)
	// Ctrl+↓ / Ctrl+↑ move the cursor to the next/previous part; Ctrl+Shift+↓/↑ go to the
	// next/previous record. Handled before the parts' own keys (capture).
	let lastCardField: HTMLElement | null = null;
	function handleFocusIn(e: FocusEvent) {
		const target = e.target as HTMLElement;
		if (!target.closest('.subpage-part') && target.matches('input, select, textarea, button')) lastCardField = target;
	}
	function currentPart(): number {
		const parts = subpageParts();
		const active = document.activeElement;
		return parts.findIndex((p) => p.contains(active)) + 1; // 0 = the card's fields
	}
	function subpageParts(): HTMLElement[] {
		return Array.from(cardPageElement?.querySelectorAll<HTMLElement>('[data-subpage-part]') ?? []);
	}
	function focusPart(index: number) {
		const parts = subpageParts();
		const count = parts.length + 1;
		const i = ((index % count) + count) % count;
		if (i === 0) {
			const field =
				lastCardField && cardPageElement?.contains(lastCardField)
					? lastCardField
					: sectionsElement?.querySelector<HTMLElement>('.section-fields input, .section-fields select, .section-fields button');
			(field ?? cardPageElement)?.focus();
			return;
		}
		const part = parts[i - 1];
		// The subpage's list (navigation mode), or the folded section's header
		const target =
			part.querySelector<HTMLElement>('.list-page') ??
			part.closest('.card')?.querySelector<HTMLElement>('.card-header') ??
			part;
		target.focus();
		target.scrollIntoView({ block: 'nearest' });
	}
	function handlePartKeys(e: KeyboardEvent) {
		const arrow = e.key === 'ArrowDown' ? 1 : e.key === 'ArrowUp' ? -1 : 0;
		if (e.ctrlKey && e.shiftKey && !e.altKey && arrow !== 0 && navigationEnabled) {
			e.preventDefault();
			e.stopPropagation();
			if (arrow > 0 && canNavigateNext) onNavigateNext?.();
			if (arrow < 0 && canNavigatePrevious) onNavigatePrevious?.();
			return;
		}
		if (!hasSubpages) return;
		const f6 = e.key === 'F6' && !e.ctrlKey && !e.altKey;
		const ctrlArrow = e.ctrlKey && !e.shiftKey && !e.altKey && arrow !== 0;
		if (!f6 && !ctrlArrow) return;
		e.preventDefault();
		e.stopPropagation();
		const step = f6 ? (e.shiftKey ? -1 : 1) : arrow;
		focusPart(currentPart() + step);
	}
	$effect(() => {
		const element = cardPageElement;
		if (!element) return;
		element.addEventListener('keydown', handlePartKeys, true);
		element.addEventListener('focusin', handleFocusIn);
		return () => {
			element.removeEventListener('keydown', handlePartKeys, true);
			element.removeEventListener('focusin', handleFocusIn);
		};
	});

	// Folded sections (FastTabs), remembered per page in the browser
	const collapsedKey = $derived(`card-collapsed-sections-${page.page.id}`);
	let collapsedSections = $state<Record<string, boolean>>({});
	$effect(() => {
		collapsedSections = getJson<Record<string, boolean>>(collapsedKey, {});
	});
	function setCollapsed(section: string, collapsed: boolean) {
		collapsedSections = { ...collapsedSections, [section]: collapsed };
		setJson(collapsedKey, collapsedSections);
	}

	// Get customized sections (fields reorganized by user preferences)
	const customizedSections = $derived(() => {
		if (!page?.page?.layout?.sections) return [];

		// Create a map of section names to fields
		const sectionMap = new Map<string, Field[]>();

		// Initialize with empty arrays for all sections
		page.page.layout.sections.forEach(section => {
			sectionMap.set(section.name, []);
		});

		// Distribute fields to sections based on customizations
		let globalIndex = 0;
		page.page.layout.sections.forEach(section => {
			(section.fields ?? []).forEach(field => {
				// Get the target section (customized or original)
				const targetSection =
					(field.source in fieldCustomizations && fieldCustomizations[field.source].section)
						? fieldCustomizations[field.source].section!
						: section.name;

				// Get order (customized or original index)
				const order = fieldCustomizations[field.source]?.order ?? globalIndex;
				globalIndex++;

				// Add field to target section if visible with order info (a page trigger may hide it)
				if (isItemVisible(field, fieldCustomizations) && fieldStates[field.source]?.visible !== false) {
					const fields = sectionMap.get(targetSection) || [];
					fields.push({ field, order } as any);
					sectionMap.set(targetSection, fields as any);
				}
			});
		});

		// Convert map back to section array (only include non-empty sections)
		// Sort fields within each section by order
		return page.page.layout.sections
			.map(section => ({
				...section,
				fields: ((sectionMap.get(section.name) || []) as any[])
					.sort((a: any, b: any) => a.order - b.order)
					.map((item: any) => item.field)
			}))
			.filter(section => section.fields.length > 0 || !!section.subpage);
	});

	// Open customize modal
	function handleCustomize() {
		customizeModalOpen = true;
	}

	// Save customizations
	function handleSaveCustomizations(customizations: Record<string, ItemCustomization>) {
		fieldCustomizations = customizations;
		const userId = $currentUser?.user_id || 'anonymous';
		savePageCustomizations(userId, page.page.id, customizations);
	}
</script>

<div class="card-page" use:shortcuts={shortcutMap()} tabindex="0" bind:this={cardPageElement}>

	<!-- Edge Navigation Buttons (classic ERP style) -->
	{#if navigationEnabled}
		<NavigationButtons
			onPrevious={onNavigatePrevious}
			onNext={onNavigateNext}
			canNavigatePrevious={canNavigatePrevious}
			canNavigateNext={canNavigateNext}
		/>
	{/if}

	<PageHeader title={cardTitle(page.page.caption, page.page.data_caption_fields, record, options)} {onclose}>
		{#snippet leftActions()}
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

			{#each page.page.actions?.filter((a) => a.promoted) || [] as action}
				{#if action.items}
					<ActionMenu {action} itemDisabled={(item) => !!item.run_object && checkIsNewRecord()} onselect={runCardAction} />
				{:else}
					<ActionButton
						caption={action.caption}
						image={action.image}
						shortcut={action.shortcut}
						tabindex={-1}
						onclick={() => runCardAction(action)}
						disabled={action.enabled === false}
					/>
				{/if}
			{/each}
		{/snippet}

		{#snippet rightActions()}
			<!-- Customize button -->
			<ActionButton caption={t(LIST.CUSTOMIZE)} image="Customize" tabindex={-1} onclick={handleCustomize} title={t(LIST.CUSTOMIZE_COLUMNS)} />

		{/snippet}
	</PageHeader>

	<!-- Error banner when save is blocked -->
	{#if saveBlocked}
		<div class="error-banner">
			<div class="error-content">
				<svg class="error-icon" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
				</svg>
				<span class="error-message">{saveBlockedMessage || t(ERR.SAVE_BLOCKED)}</span>
			</div>
			<div class="error-actions">
				<button class="error-clear-btn" onclick={() => onclearerror?.()}>
					{t(CARD.CLEAR_FORM)}
				</button>
			</div>
		</div>
	{/if}

	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div
		class="sections-container"
		bind:this={sectionsElement}
		onkeydown={(e) => {
			// Enter moves between the card's fields — not in a subpage (its list handles Enter)
			if ((e.target as HTMLElement).closest('.subpage-part')) return;
			handleFieldEnterKey(e, e.currentTarget as HTMLElement);
		}}
	>
		{#if isEmptyRecord() && !editMode}
			<!-- Empty state -->
			<div class="empty-state">
				<div class="empty-state-icon">
					<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
					</svg>
				</div>
				<h3 class="empty-state-title">{t(CARD.NO_RECORD)}</h3>
				<p class="empty-state-text">{t(CARD.SELECT_OR_CREATE)}</p>
				<Button variant="primary" onclick={() => handleAction('New')}>
					{#snippet icon()}
						<PlusIcon size={16} color="currentColor" />
					{/snippet}
					{t(CARD.CREATE_NEW)}
				</Button>
			</div>
		{:else}
			{@const allFields = customizedSections().flatMap(s => s.fields)}
			{#each customizedSections() as section}
				<Card
					collapsible
					bind:collapsed={
						() => collapsedSections[section.name] ?? false,
						(v) => setCollapsed(section.name, v)
					}
				>
					{#snippet header()}
						<h3 class="section-title text-heading dark:text-blue-400">
							{section.caption}
						</h3>
					{/snippet}

					{#if section.subpage}
						{@const sub = section.subpage}
						<!-- A subpage: the list page with the lines of this record (its own keys) -->
						<div class="subpage-part" data-subpage-part data-own-keys>
							{#if canShowLines(sub)}
								{#key JSON.stringify(linkValues(sub))}
									<PageRenderer
										pageid={sub.page_id}
										subpage={true}
										fixedFilters={linkFilters(sub)}
										newRecordDefaults={linkValues(sub)}
										onexit={() => focusPart(0)}
										onlineschange={onrefresh}
										refreshToken={linesRefresh}
									/>
								{/key}
							{:else}
								<p class="subpage-empty">{t(CARD.SAVE_FIRST)}</p>
							{/if}
						</div>
					{:else}
					<div class="section-fields">
						{#each section.fields as field (field.source)}
							{@const fieldEditable = field.editable !== false && fieldStates[field.source]?.editable !== false && editMode && !saveBlocked}
							{@const fieldTabIndex = allFields.findIndex(f => f.source === field.source) + 1}
							<FieldRenderer
								{field}
								bind:value={record[field.source]}
								caption={getFieldCaption(field.source, captions, field.caption)}
								editable={fieldEditable}
								fieldType={fieldTypes[field.source]}
								options={options[field.source]}
								lookups={lookups[field.source]}
								fieldCaptions={captions}
								tabindex={fieldTabIndex}
								onblur={handleFieldBlur}
							/>
						{/each}
					</div>
					{/if}
				</Card>
			{/each}
		{/if}
	</div>
</div>

<!-- The card's page objects: progress, questions, input, message boxes -->
<ObjectRunner bind:this={runner} ondone={handleRunDone} />

<!-- Customize Page Modal -->
{#if customizeModalOpen}
	<CustomizeFieldsModal
		open={customizeModalOpen}
		{page}
		customizations={fieldCustomizations}
		mode="card"
		onclose={() => customizeModalOpen = false}
		onsave={handleSaveCustomizations}
	/>
{/if}

<style>
	.card-page {
		@apply flex flex-col flex-1 min-h-0;
		@apply relative;
		@apply bg-surface;
		/* The sections scroll inside the window (as the list page) */
		max-height: calc(100vh - 180px);
	}

	:global(.dark) .card-page {
		background-color: rgb(var(--gray-900)); /* gray-900 */
	}



	.sections-container {
		@apply flex flex-col gap-2 overflow-y-auto flex-1 min-h-0;
		@apply px-1; /* Small padding for scrollbar spacing */
	}

	/* A subpage inside its section */
	.subpage-part {
		@apply -mx-2;
	}

	.subpage-empty {
		@apply text-sm text-gray-500 dark:text-gray-400 py-2;
	}

	.section-fields {
		@apply grid grid-cols-1 md:grid-cols-2 gap-x-6 gap-y-1.5;
	}

	/* Compact card: section headers, inputs and values take less vertical space */
	.section-title {
		@apply text-base font-semibold;
	}

	.card-page :global(.card-header) {
		@apply py-1.5;
	}

	.card-page :global(.card-body) {
		@apply px-4 py-2;
	}

	.card-page :global(.field-group) {
		gap: 0.125rem;
	}

	.card-page :global(.field-group .field-label) {
		@apply text-xs;
	}

	.card-page :global(.field-group input.input),
	.card-page :global(.field-group select.select),
	.card-page :global(.field-group .field-value),
	.card-page :global(.option-dropdown .option-trigger) {
		@apply py-1 px-2.5 text-sm;
		min-height: 2rem;
	}

	.card-page :global(.field-group .checkbox-wrapper) {
		@apply py-1;
		min-height: 2rem;
	}

	/* Save state container - fixed width to prevent layout shift */
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

	/* Empty state styles */
	.empty-state {
		@apply flex flex-col items-center justify-center;
		@apply py-16 px-8;
		@apply text-center;
	}

	.empty-state-icon {
		@apply w-16 h-16 mb-4;
		@apply text-gray-300;
	}

	:global(.dark) .empty-state-icon {
		color: rgb(var(--gray-600)); /* gray-600 */
	}

	.empty-state-icon svg {
		@apply w-full h-full;
	}

	.empty-state-title {
		@apply text-xl font-semibold text-gray-700 mb-2;
	}

	:global(.dark) .empty-state-title {
		color: rgb(var(--gray-300)); /* gray-300 */
	}

	.empty-state-text {
		@apply text-gray-500 mb-6;
	}

	:global(.dark) .empty-state-text {
		color: rgb(var(--gray-400)); /* gray-400 */
	}

	.card-page :global(.edge-nav-btn:not(:disabled):hover) {
		@apply shadow-xl;
	}

	/* Ensure PageHeader stays at top */
	.card-page :global(.page-header) {
		@apply sticky top-0 z-10 bg-surface;
		@apply mb-4;
	}

	:global(.dark) .card-page :global(.page-header) {
		background-color: rgb(var(--gray-900)); /* gray-900 */
	}

	/* Error banner styles */
	.error-banner {
		@apply flex items-center justify-between;
		@apply mx-4 mb-4 px-4 py-3;
		@apply bg-red-50 border border-red-200 rounded-lg;
	}

	:global(.dark) .error-banner {
		background-color: rgba(127, 29, 29, 0.3); /* red-900 with opacity */
		border-color: #991b1b; /* red-800 */
	}

	.error-content {
		@apply flex items-center gap-3;
	}

	.error-icon {
		@apply w-5 h-5 text-red-600 flex-shrink-0;
	}

	:global(.dark) .error-icon {
		color: #f87171; /* red-400 */
	}

	.error-message {
		@apply text-sm text-red-700 font-medium;
	}

	:global(.dark) .error-message {
		color: #fca5a5; /* red-300 */
	}

	.error-actions {
		@apply flex items-center gap-2;
	}

	.error-clear-btn {
		@apply px-3 py-1.5 text-sm font-medium;
		@apply bg-red-600 text-white rounded;
		@apply hover:bg-red-700 transition-colors;
	}

	:global(.dark) .error-clear-btn {
		background-color: #dc2626; /* red-600 */
	}

	:global(.dark) .error-clear-btn:hover {
		background-color: #b91c1c; /* red-700 */
	}
</style>
