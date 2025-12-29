<script lang="ts">
	import type { PageDefinition } from '$lib/types/pages';
	import Button from '$lib/components/Button.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import { shortcuts } from '$lib/utils/shortcuts';
	import { cn } from '$lib/utils/cn';

	interface Props {
		page: PageDefinition;
		records?: Array<Record<string, any>>;
		captions?: Record<string, string>;
		onaction?: (actionName: string, record?: Record<string, any>) => void;
		onrowclick?: (record: Record<string, any>) => void;
	}

	let { page, records = [], captions = {}, onaction, onrowclick }: Props = $props();

	// Track selected row index
	let selectedIndex = $state(-1);

	// Get selected record
	const selectedRecord = $derived(
		selectedIndex >= 0 && selectedIndex < records.length ? records[selectedIndex] : null
	);

	// Handle action clicks
	function handleAction(actionName: string) {
		onaction?.(actionName, selectedRecord || undefined);
	}

	// Handle row click
	function handleRowClick(index: number) {
		selectedIndex = index;
		if (page.page.card_page_id) {
			// If there's a card page, navigate to it
			onrowclick?.(records[index]);
		}
	}

	// Handle row double-click
	function handleRowDoubleClick(index: number) {
		if (page.page.card_page_id) {
			onrowclick?.(records[index]);
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

		// Add navigation shortcuts
		map['ArrowDown'] = moveDown;
		map['ArrowUp'] = moveUp;
		map['Home'] = moveFirst;
		map['End'] = moveLast;
		map['Enter'] = openCard;

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

	function openCard() {
		if (selectedRecord && page.page.card_page_id) {
			onrowclick?.(selectedRecord);
		}
	}

	// Get field caption
	function getFieldCaption(fieldSource: string, fieldCaption?: string): string {
		return captions[fieldSource] || fieldCaption || fieldSource;
	}

	// Format cell value
	function formatValue(val: any): string {
		if (val === null || val === undefined) {
			return '';
		}
		if (typeof val === 'boolean') {
			return val ? 'Yes' : 'No';
		}
		return String(val);
	}

	// Get field style classes
	function getFieldStyle(field: any) {
		const classes: string[] = [];

		switch (field.style) {
			case 'Strong':
				classes.push('font-bold text-nav-blue');
				break;
			case 'Attention':
				classes.push('font-medium text-orange-600');
				break;
			case 'Favorable':
				classes.push('text-green-600');
				break;
			case 'Unfavorable':
				classes.push('text-red-600');
				break;
		}

		return classes.join(' ');
	}
</script>

<div class="list-page" use:shortcuts={shortcutMap()}>
	<PageHeader title={page.page.caption}>
		<svelte:fragment slot="actions">
			{#each page.page.actions?.filter((a) => a.promoted) || [] as action}
				<Button
					variant={action.name === 'Delete' ? 'danger' : 'secondary'}
					size="sm"
					onclick={() => handleAction(action.name)}
					disabled={action.enabled === false ||
						(action.name !== 'New' && action.name !== 'Refresh' && !selectedRecord)}
				>
					{action.caption}
					{#if action.shortcut}
						<span class="ml-2 text-xs opacity-70">{action.shortcut}</span>
					{/if}
				</Button>
			{/each}
		</svelte:fragment>
	</PageHeader>

	<div class="table-container">
		<table class="table">
			<thead>
				<tr>
					{#each page.page.layout.repeater?.fields || [] as field}
						<th style={field.width ? `width: ${field.width}px` : ''}>
							{getFieldCaption(field.source, field.caption)}
						</th>
					{/each}
				</tr>
			</thead>
			<tbody>
				{#each records as record, index}
					<tr
						class={cn('cursor-pointer', selectedIndex === index && 'selected')}
						onclick={() => handleRowClick(index)}
						ondblclick={() => handleRowDoubleClick(index)}
					>
						{#each page.page.layout.repeater?.fields || [] as field}
							<td class={getFieldStyle(field)}>
								{formatValue(record[field.source])}
							</td>
						{/each}
					</tr>
				{:else}
					<tr>
						<td colspan={page.page.layout.repeater?.fields?.length || 1} class="text-center py-8">
							<span class="text-gray-500">No records found</span>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>

	<div class="status-bar">
		<span class="text-sm text-gray-600">
			{records.length} record{records.length !== 1 ? 's' : ''}
			{#if selectedRecord}
				• Row {selectedIndex + 1} selected
			{/if}
		</span>
	</div>
</div>

<style>
	.list-page {
		@apply flex flex-col gap-4 h-full;
	}

	.table-container {
		@apply flex-1 overflow-auto border border-gray-200 rounded-lg;
	}

	.table {
		@apply w-full border-collapse;
	}

	.table thead {
		@apply sticky top-0 bg-nav-blue text-white z-10;
	}

	.table th {
		@apply px-4 py-3 text-left text-sm font-semibold;
		border-right: 1px solid rgba(255, 255, 255, 0.1);
	}

	.table th:last-child {
		border-right: none;
	}

	.table tbody tr {
		@apply border-b border-gray-200 hover:bg-blue-50 transition-colors;
	}

	.table tbody tr.selected {
		@apply bg-blue-100 hover:bg-blue-100;
	}

	.table td {
		@apply px-4 py-2 text-sm;
	}

	.status-bar {
		@apply px-4 py-2 bg-gray-50 border-t border-gray-200 rounded-b;
	}
</style>
