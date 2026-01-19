<script lang="ts">
	import { cn } from '$lib/utils/cn';
	import { toast } from '$lib/stores/toast';

	interface LookupColumn {
		source: string;
		width: number;
	}

	interface LookupRow {
		_key: string;
		[key: string]: any;
	}

	interface Props {
		columns: LookupColumn[];
		rows: LookupRow[];
		value?: string;
		fieldName?: string; // Field name for error messages
		captions?: Record<string, string>; // Field captions for column headers
		searchTimeout?: number; // Auto-clear search timeout in ms (default 1500)
		tabindex?: number;
		disabled?: boolean;
		error?: boolean;
		onselect?: (key: string) => void;
		onblur?: () => void;
	}

	let {
		columns,
		rows,
		value = $bindable(''),
		fieldName = '',
		captions = {},
		searchTimeout = 1500,
		tabindex,
		disabled = false,
		error = false,
		onselect,
		onblur
	}: Props = $props();

	let isOpen = $state(false);
	let containerRef: HTMLDivElement;
	let inputRef: HTMLInputElement;
	let bodyRef: HTMLDivElement;
	let selectedIndex = $state(-1);

	// Track input value separately from actual value (for typing)
	let inputValue = $state(value || '');

	// Sync inputValue when value changes externally
	$effect(() => {
		inputValue = value || '';
	});

	// Filter rows based on input value (type-ahead filtering)
	const filteredRows = $derived(() => {
		if (!inputValue) return rows;
		const term = inputValue.toLowerCase();
		return rows.filter(row => {
			// Search in _key (code) first, then all columns
			if (row._key.toLowerCase().startsWith(term)) return true;
			return columns.some(col => {
				const val = row[col.source];
				if (val === null || val === undefined) return false;
				return String(val).toLowerCase().includes(term);
			});
		});
	});

	// Scroll selected row into view when navigating with keyboard
	$effect(() => {
		if (isOpen && selectedIndex >= 0 && bodyRef) {
			const rowElements = bodyRef.querySelectorAll('.lookup-row');
			if (rowElements[selectedIndex]) {
				rowElements[selectedIndex].scrollIntoView({ block: 'nearest' });
			}
		}
	});

	// Get display value for current selection (show key/code)
	const displayValue = $derived(() => {
		if (!value) return '';
		return value; // Just show the key/code value
	});

	// Get column header text
	function getColumnHeader(source: string): string {
		return captions[source] || source.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase());
	}

	// Format cell value for display
	function formatCellValue(val: any): string {
		if (val === null || val === undefined) return '';
		if (typeof val === 'boolean') return val ? 'Yes' : 'No';
		return String(val);
	}

	function openDropdown() {
		if (disabled) return;
		isOpen = true;
		// Find current selection index in filtered rows
		selectedIndex = filteredRows().findIndex(r => r._key === value);
		if (selectedIndex < 0 && filteredRows().length > 0) selectedIndex = 0;
	}

	function handleToggle() {
		if (disabled) return;
		if (isOpen) {
			isOpen = false;
		} else {
			openDropdown();
		}
	}

	function handleSelect(row: LookupRow) {
		value = row._key;
		inputValue = row._key;
		isOpen = false;
		onselect?.(row._key);
		onblur?.();
	}

	// Handle input change (user typing)
	function handleInput(e: Event) {
		const target = e.target as HTMLInputElement;
		inputValue = target.value;

		// Open dropdown when typing
		if (!isOpen && inputValue) {
			openDropdown();
		}

		// Reset selection to first match
		selectedIndex = 0;
	}

	// Handle blur - validate and set value
	function handleBlur() {
		// Small delay to allow click on dropdown to register
		setTimeout(() => {
			if (!containerRef?.contains(document.activeElement)) {
				isOpen = false;

				// Validate input - find matching row
				const trimmedInput = inputValue.trim().toUpperCase();
				if (trimmedInput) {
					// Find exact match by key (case-insensitive)
					const matchingRow = rows.find(r => r._key.toUpperCase() === trimmedInput);
					if (matchingRow) {
						value = matchingRow._key;
						inputValue = matchingRow._key;
						onselect?.(matchingRow._key);
					} else {
						// No match - show error and revert to previous value
						const fieldLabel = fieldName || 'Value';
						toast.error(`${fieldLabel} '${inputValue.trim()}' does not exist`);
						inputValue = value || '';
					}
				} else {
					// Empty input - clear value
					value = '';
					inputValue = '';
					onselect?.('');
				}

				onblur?.();
			}
		}, 150);
	}

	function handleKeydown(e: KeyboardEvent) {
		switch (e.key) {
			case 'Escape':
				e.preventDefault();
				if (isOpen) {
					isOpen = false;
					// Revert to current value
					inputValue = value || '';
				}
				break;
			case 'ArrowDown':
				e.preventDefault();
				if (!isOpen) {
					openDropdown();
				} else {
					selectedIndex = Math.min(selectedIndex + 1, filteredRows().length - 1);
				}
				break;
			case 'ArrowUp':
				e.preventDefault();
				if (isOpen) {
					selectedIndex = Math.max(selectedIndex - 1, 0);
				}
				break;
			case 'Enter':
				e.preventDefault();
				if (isOpen && selectedIndex >= 0 && selectedIndex < filteredRows().length) {
					handleSelect(filteredRows()[selectedIndex]);
				} else if (!isOpen) {
					openDropdown();
				}
				break;
			case 'Tab':
				// Let blur handle validation
				isOpen = false;
				break;
			case 'F4':
				// F4 toggles dropdown (Business Central style)
				e.preventDefault();
				handleToggle();
				break;
		}
	}

	function handleClickOutside(e: MouseEvent) {
		if (containerRef && !containerRef.contains(e.target as Node)) {
			isOpen = false;
			// Revert input to actual value
			inputValue = value || '';
		}
	}

	$effect(() => {
		if (isOpen) {
			document.addEventListener('click', handleClickOutside);
			return () => document.removeEventListener('click', handleClickOutside);
		}
	});

	// Calculate total width
	const totalWidth = $derived(columns.reduce((sum, col) => sum + (col.width || 100), 0));
</script>

<div
	class="lookup-dropdown"
	bind:this={containerRef}
	style="--dropdown-width: {Math.max(totalWidth + 20, 200)}px"
>
	<div class="lookup-input-wrapper">
		<input
			type="text"
			class={cn('lookup-input', error && 'input-error')}
			bind:this={inputRef}
			value={inputValue}
			{tabindex}
			{disabled}
			oninput={handleInput}
			onkeydown={handleKeydown}
			onblur={handleBlur}
			onfocus={openDropdown}
			aria-haspopup="listbox"
			aria-expanded={isOpen}
			autocomplete="off"
		/>
		<button
			type="button"
			class="lookup-arrow-btn"
			tabindex={-1}
			{disabled}
			onclick={handleToggle}
			aria-label="Toggle dropdown"
		>
			<span class="lookup-arrow">{isOpen ? '▲' : '▼'}</span>
		</button>
	</div>

	{#if isOpen}
		<div class="lookup-panel" role="listbox">
			<!-- Column headers -->
			<div class="lookup-header">
				{#each columns as col}
					<div
						class="lookup-header-cell"
						style="width: {col.width || 100}px"
					>
						{getColumnHeader(col.source)}
					</div>
				{/each}
			</div>

			<!-- Rows -->
			<div class="lookup-body" bind:this={bodyRef}>
				{#each filteredRows() as row, i}
					<div
						class={cn('lookup-row', row._key === value && 'selected', i === selectedIndex && 'focused')}
						role="option"
						aria-selected={row._key === value}
						onclick={() => handleSelect(row)}
						onmouseenter={() => selectedIndex = i}
					>
						{#each columns as col}
							<div
								class="lookup-cell"
								style="width: {col.width || 100}px"
							>
								{formatCellValue(row[col.source])}
							</div>
						{/each}
					</div>
				{/each}
				{#if filteredRows().length === 0}
					<div class="lookup-empty">{inputValue ? 'No matches found' : 'No records found'}</div>
				{/if}
			</div>
		</div>
	{/if}
</div>

<style>
	.lookup-dropdown {
		position: relative;
		width: 100%;
	}

	.lookup-input-wrapper {
		@apply relative flex items-center;
	}

	.lookup-input {
		@apply w-full px-2 py-1 pr-8 text-left bg-white border border-gray-300 rounded;
		@apply focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500;
		@apply disabled:opacity-50 disabled:cursor-not-allowed;
		min-height: 2rem;
		font-size: 0.875rem;
	}

	:global(.dark) .lookup-input {
		background-color: var(--color-bg-input, #1f2937);
		border-color: var(--color-border-secondary, #374151);
		color: var(--color-text-primary, #f9fafb);
	}

	.lookup-input.input-error {
		@apply border-red-500;
	}

	.lookup-arrow-btn {
		@apply absolute right-0 top-0 bottom-0 px-2;
		@apply flex items-center justify-center;
		@apply bg-transparent border-none cursor-pointer;
		@apply hover:bg-gray-100 rounded-r;
		@apply disabled:opacity-50 disabled:cursor-not-allowed;
	}

	:global(.dark) .lookup-arrow-btn:hover {
		background-color: var(--color-bg-secondary, #374151);
	}

	.lookup-arrow {
		@apply text-xs text-gray-400;
	}

	.lookup-panel {
		position: absolute;
		top: 100%;
		left: 0;
		z-index: 50;
		width: var(--dropdown-width);
		min-width: 100%;
		max-height: 300px;
		margin-top: 2px;
		@apply bg-white border border-gray-300 rounded shadow-lg;
		overflow: hidden;
	}

	:global(.dark) .lookup-panel {
		background-color: var(--color-bg-primary, #111827);
		border-color: var(--color-border-secondary, #374151);
	}

	.lookup-header {
		@apply flex bg-gray-100 border-b border-gray-200;
		position: sticky;
		top: 0;
	}

	:global(.dark) .lookup-header {
		background-color: var(--color-bg-secondary, #1f2937);
		border-color: var(--color-border-secondary, #374151);
	}

	.lookup-header-cell {
		@apply px-2 py-1 text-xs font-semibold text-gray-600 truncate;
		flex-shrink: 0;
	}

	:global(.dark) .lookup-header-cell {
		color: var(--color-text-secondary, #9ca3af);
	}

	.lookup-body {
		max-height: 250px;
		overflow-y: auto;
	}

	.lookup-row {
		@apply flex cursor-pointer;
		@apply hover:bg-gray-100;
	}

	:global(.dark) .lookup-row:hover {
		background-color: var(--color-bg-secondary, #1f2937);
	}

	.lookup-row.selected {
		@apply bg-blue-50;
	}

	:global(.dark) .lookup-row.selected {
		background-color: rgba(59, 130, 246, 0.1);
	}

	.lookup-row.focused {
		@apply bg-gray-100;
	}

	:global(.dark) .lookup-row.focused {
		background-color: var(--color-bg-secondary, #1f2937);
	}

	.lookup-row.selected.focused {
		@apply bg-blue-100;
	}

	:global(.dark) .lookup-row.selected.focused {
		background-color: rgba(59, 130, 246, 0.2);
	}

	.lookup-cell {
		@apply px-2 py-1 text-sm truncate border-r border-gray-100;
		flex-shrink: 0;
	}

	:global(.dark) .lookup-cell {
		border-color: var(--color-border-secondary, #374151);
		color: var(--color-text-primary, #f9fafb);
	}

	.lookup-cell:last-child {
		@apply border-r-0;
	}

	.lookup-empty {
		@apply px-3 py-4 text-sm text-gray-400 text-center;
	}
</style>
