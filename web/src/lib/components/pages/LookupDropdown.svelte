<script lang="ts">
	import { cn } from '$lib/utils/cn';

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
		captions = {},
		searchTimeout = 1500,
		tabindex,
		disabled = false,
		error = false,
		onselect,
		onblur
	}: Props = $props();

	// Use configured timeout (0 or undefined = no auto-clear)
	const hasAutoTimeout = $derived(searchTimeout > 0);

	let isOpen = $state(false);
	let containerRef: HTMLDivElement;
	let bodyRef: HTMLDivElement;
	let selectedIndex = $state(-1);
	let searchTerm = $state('');
	let searchTimer: ReturnType<typeof setTimeout> | null = null;

	// Filter rows based on search term
	const filteredRows = $derived(() => {
		if (!searchTerm) return rows;
		const term = searchTerm.toLowerCase();
		return rows.filter(row => {
			// Search in all column values
			return columns.some(col => {
				const val = row[col.source];
				if (val === null || val === undefined) return false;
				return String(val).toLowerCase().includes(term);
			});
		});
	});

	// Reset search when dropdown closes
	$effect(() => {
		if (!isOpen) {
			searchTerm = '';
			if (searchTimer) {
				clearTimeout(searchTimer);
				searchTimer = null;
			}
		}
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

	// Get display value for current selection
	const displayValue = $derived(() => {
		if (!value) return '';
		const row = rows.find(r => r._key === value);
		if (!row) return value;
		// Show first column value
		if (columns.length > 0) {
			return row[columns[0].source] ?? value;
		}
		return value;
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

	function handleToggle() {
		if (disabled) return;
		isOpen = !isOpen;
		if (isOpen) {
			// Find current selection index
			selectedIndex = rows.findIndex(r => r._key === value);
		}
	}

	function handleSelect(row: LookupRow) {
		value = row._key;
		isOpen = false;
		onselect?.(row._key);
		onblur?.();
	}

	function handleKeydown(e: KeyboardEvent) {
		if (!isOpen) {
			if (e.key === 'Enter' || e.key === ' ' || e.key === 'ArrowDown') {
				e.preventDefault();
				isOpen = true;
				selectedIndex = rows.findIndex(r => r._key === value);
				if (selectedIndex < 0) selectedIndex = 0;
			}
			return;
		}

		switch (e.key) {
			case 'Escape':
				e.preventDefault();
				if (searchTerm) {
					// First Escape clears search, second closes dropdown
					searchTerm = '';
					selectedIndex = 0;
				} else {
					isOpen = false;
				}
				break;
			case 'ArrowDown':
				e.preventDefault();
				selectedIndex = Math.min(selectedIndex + 1, filteredRows().length - 1);
				break;
			case 'ArrowUp':
				e.preventDefault();
				selectedIndex = Math.max(selectedIndex - 1, 0);
				break;
			case 'Enter':
				e.preventDefault();
				if (selectedIndex >= 0 && selectedIndex < filteredRows().length) {
					handleSelect(filteredRows()[selectedIndex]);
				}
				break;
			case 'Tab':
				isOpen = false;
				break;
			case 'Backspace':
				e.preventDefault();
				if (searchTerm.length > 0) {
					searchTerm = searchTerm.slice(0, -1);
					selectedIndex = 0;
				}
				break;
			default:
				// Type-ahead search: single printable character
				if (e.key.length === 1 && !e.ctrlKey && !e.altKey && !e.metaKey) {
					e.preventDefault();
					searchTerm += e.key;
					selectedIndex = 0;

					// Auto-clear search after configured timeout (only if timeout is set)
					if (searchTimer) clearTimeout(searchTimer);
					if (hasAutoTimeout) {
						searchTimer = setTimeout(() => {
							searchTerm = '';
						}, searchTimeout);
					}
				}
				break;
		}
	}

	function handleClickOutside(e: MouseEvent) {
		if (containerRef && !containerRef.contains(e.target as Node)) {
			isOpen = false;
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
	<button
		type="button"
		class={cn('lookup-trigger', error && 'input-error')}
		{tabindex}
		{disabled}
		onclick={handleToggle}
		onkeydown={handleKeydown}
		aria-haspopup="listbox"
		aria-expanded={isOpen}
	>
		<span class="lookup-value">{displayValue()}</span>
		<span class="lookup-arrow">{isOpen ? '▲' : '▼'}</span>
	</button>

	{#if isOpen}
		<div class="lookup-panel" role="listbox">
			<!-- Search indicator -->
			{#if searchTerm}
				<div class="lookup-search">
					<span class="search-icon">🔍</span>
					<span class="search-term">{searchTerm}</span>
					<span class="search-hint">({filteredRows().length} matches)</span>
				</div>
			{/if}

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
					<div class="lookup-empty">{searchTerm ? 'No matches found' : 'No records found'}</div>
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

	.lookup-trigger {
		@apply w-full px-2 py-1 text-left bg-white border border-gray-300 rounded;
		@apply flex items-center justify-between gap-2;
		@apply focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500;
		@apply disabled:opacity-50 disabled:cursor-not-allowed;
		min-height: 2rem;
		font-size: 0.875rem;
	}

	:global(.dark) .lookup-trigger {
		background-color: var(--color-bg-input, #1f2937);
		border-color: var(--color-border-secondary, #374151);
		color: var(--color-text-primary, #f9fafb);
	}

	.lookup-trigger.input-error {
		@apply border-red-500;
	}

	.lookup-value {
		@apply flex-1 truncate;
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

	.lookup-search {
		@apply px-2 py-1 bg-blue-50 border-b border-blue-200 flex items-center gap-2;
		font-size: 0.75rem;
	}

	:global(.dark) .lookup-search {
		background-color: rgba(59, 130, 246, 0.1);
		border-color: rgba(59, 130, 246, 0.3);
	}

	.search-icon {
		font-size: 0.875rem;
	}

	.search-term {
		@apply font-semibold text-blue-700;
	}

	:global(.dark) .search-term {
		color: #93c5fd;
	}

	.search-hint {
		@apply text-gray-500;
	}

	:global(.dark) .search-hint {
		color: #9ca3af;
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
