<script lang="ts">
	import Modal from '$lib/components/Modal.svelte';
	import Button from '$lib/components/Button.svelte';
	import type { PageDefinition, Field } from '$lib/types/pages';
	import { createDragAndDrop, type DragItem } from '$lib/utils/dragAndDrop.svelte';

	interface ColumnCustomization {
		visible: boolean;
		order?: number;
	}

	interface Props {
		open?: boolean;
		page: PageDefinition;
		customizations: Record<string, ColumnCustomization>;
		onclose?: () => void;
		onsave?: (customizations: Record<string, ColumnCustomization>) => void;
	}

	let { open = false, page, customizations, onclose, onsave }: Props = $props();

	// Local copy of customizations for editing
	let localCustomizations = $state<Record<string, ColumnCustomization>>({ ...customizations });

	// Drag and drop utility
	const drag = createDragAndDrop<ColumnCustomization>();

	// Sync when customizations prop changes
	$effect(() => {
		localCustomizations = { ...customizations };
	});

	// Get all columns from repeater with applied order
	const allColumns = $derived(() => {
		const columns: Array<DragItem<{ index: number }>> = (page.page.layout.repeater?.fields || []).map((field, index) => ({
			field,
			data: { index }
		}));

		// Sort by custom order if available
		return columns.sort((a, b) => {
			const orderA = localCustomizations[a.field.source]?.order ?? a.data.index;
			const orderB = localCustomizations[b.field.source]?.order ?? b.data.index;
			return orderA - orderB;
		});
	});

	// Toggle column visibility
	function toggleColumn(fieldSource: string, currentVisible: boolean) {
		localCustomizations = {
			...localCustomizations,
			[fieldSource]: {
				...(localCustomizations[fieldSource] || {}),
				visible: !currentVisible
			}
		};
	}

	// Get effective visibility for a column
	function getEffectiveVisibility(field: Field): boolean {
		// If user has customized this column, use that
		if (field.source in localCustomizations) {
			return localCustomizations[field.source].visible;
		}
		// Otherwise use the field's visible property (default true)
		return field.visible !== false;
	}

	// Save customizations
	function handleSave() {
		onsave?.(localCustomizations);
		onclose?.();
	}

	// Reset to defaults
	function handleReset() {
		localCustomizations = {};
	}

	// Drag and drop handler wrapper
	function handleDropColumn(e: DragEvent, targetIndex: number) {
		localCustomizations = drag.handleDrop(e, targetIndex, allColumns(), localCustomizations);
	}
</script>

<Modal {open} onclose={onclose} size="expanded">
	<div class="customize-modal">
		<div class="customize-header">
			<h2 class="customize-title">Customize Columns: {page.page.caption}</h2>
			<p class="customize-subtitle">Show/hide columns and drag to reorder</p>
		</div>

		<div class="customize-body">
			<div class="fields-list">
				{#each allColumns() as item, idx}
					{@const isVisible = getEffectiveVisibility(item.field)}
					{@const isDragging = drag.isDragging(item.field.source)}
					{@const isDragOver = drag.isDragOver(idx)}
					<div
						class="field-item"
						class:dragging={isDragging}
						class:drag-over={isDragOver}
						draggable="true"
						ondragstart={(e) => drag.handleDragStart(e, item)}
						ondragover={(e) => drag.handleDragOver(e, idx)}
						ondragleave={drag.handleDragLeave}
						ondrop={(e) => handleDropColumn(e, idx)}
						ondragend={drag.handleDragEnd}
					>
						<!-- Drag handle -->
						<div class="drag-handle" title="Drag to reorder">
							<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
								<path d="M10 6a2 2 0 110-4 2 2 0 010 4zM10 12a2 2 0 110-4 2 2 0 010 4zM10 18a2 2 0 110-4 2 2 0 010 4z" />
							</svg>
						</div>

						<label class="field-checkbox" class:pointer-events-none={isDragging}>
							<input
								type="checkbox"
								checked={isVisible}
								onchange={(e) => {
									e.stopPropagation();
									toggleColumn(item.field.source, isVisible);
								}}
								onclick={(e) => e.stopPropagation()}
							/>
							<div class="field-info">
								<span class="field-name">{item.field.caption || item.field.source}</span>
								{#if item.field.width}
									<span class="field-meta">Width: {item.field.width}px</span>
								{/if}
							</div>
							{#if item.field.visible === false && !localCustomizations[item.field.source]}
								<span class="field-badge">Hidden by default</span>
							{/if}
						</label>
					</div>
				{/each}
			</div>
		</div>

		<div class="customize-footer">
			<Button variant="secondary" size="sm" onclick={handleReset}>
				Reset to Defaults
			</Button>
			<div class="footer-actions">
				<Button variant="secondary" size="sm" onclick={onclose}>
					Cancel
				</Button>
				<Button variant="primary" size="sm" onclick={handleSave}>
					Save Customization
				</Button>
			</div>
		</div>
	</div>
</Modal>

<style>
	.customize-modal {
		@apply flex flex-col h-full max-h-[80vh];
	}

	.customize-header {
		@apply px-6 py-4 border-b border-gray-200 dark:border-gray-700;
		@apply bg-white dark:bg-gray-800;
	}

	.customize-title {
		@apply text-xl font-bold text-nav-blue dark:text-blue-400;
	}

	.customize-subtitle {
		@apply mt-1 text-sm text-gray-600 dark:text-gray-400;
	}

	.customize-body {
		@apply flex-1 overflow-y-auto p-6;
		@apply bg-gray-50 dark:bg-gray-900;
	}

	.fields-list {
		@apply flex flex-col gap-2;
	}

	.field-item {
		@apply bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700;
		@apply hover:border-blue-300 dark:hover:border-blue-700 transition-colors;
		@apply flex items-center gap-2;
		cursor: grab;
	}

	.field-item.dragging {
		@apply opacity-50;
		cursor: grabbing;
	}

	.field-item.drag-over {
		@apply border-blue-500 dark:border-blue-400 border-2;
		@apply bg-blue-50 dark:bg-blue-900/20;
	}

	.drag-handle {
		@apply p-2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300;
		@apply cursor-grab;
		touch-action: none;
	}

	.field-item.dragging .drag-handle {
		@apply cursor-grabbing;
	}

	.field-checkbox {
		@apply flex items-center gap-3 p-4 cursor-pointer flex-1;
	}

	.field-checkbox.pointer-events-none {
		pointer-events: none;
	}

	.field-checkbox input[type='checkbox'] {
		@apply w-5 h-5 rounded border-gray-300 text-blue-600;
		@apply focus:ring-2 focus:ring-blue-500 focus:ring-offset-0;
		@apply cursor-pointer;
	}

	.field-info {
		@apply flex flex-col flex-1;
	}

	.field-name {
		@apply font-medium text-gray-900 dark:text-gray-100;
	}

	.field-meta {
		@apply text-sm text-gray-500 dark:text-gray-400;
	}

	.field-badge {
		@apply px-2 py-1 text-xs rounded-full;
		@apply bg-orange-100 text-orange-700 dark:bg-orange-900 dark:text-orange-300;
	}

	.customize-footer {
		@apply flex items-center justify-between px-6 py-4;
		@apply border-t border-gray-200 dark:border-gray-700;
		@apply bg-white dark:bg-gray-800;
	}

	.footer-actions {
		@apply flex gap-3;
	}
</style>
