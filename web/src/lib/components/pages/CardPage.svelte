<script lang="ts">
	import type { PageDefinition } from '$lib/types/pages';
	import FieldRenderer from './FieldRenderer.svelte';
	import Button from '$lib/components/Button.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import Card from '$lib/components/Card.svelte';
	import { shortcuts, createShortcutMap } from '$lib/utils/shortcuts';

	interface Props {
		page: PageDefinition;
		record?: Record<string, any>;
		captions?: Record<string, string>;
		onaction?: (actionName: string) => void;
		onsave?: (record: Record<string, any>) => void;
	}

	let { page, record = $bindable({}), captions = {}, onaction, onsave }: Props = $props();

	// Track if page is in edit mode
	let isEditing = $state(false);

	// Local copy of record for editing
	let editedRecord = $state({ ...record });

	// Sync editedRecord when record changes and not editing
	$effect(() => {
		if (!isEditing) {
			editedRecord = { ...record };
		}
	});

	// Handle action clicks
	function handleAction(actionName: string) {
		onaction?.(actionName);

		// Handle built-in actions
		switch (actionName) {
			case 'New':
				handleNew();
				break;
			case 'Edit':
				handleEdit();
				break;
			case 'Delete':
				handleDelete();
				break;
			case 'Save':
				handleSave();
				break;
			case 'Cancel':
				handleCancel();
				break;
		}
	}

	function handleNew() {
		editedRecord = {};
		isEditing = true;
	}

	function handleEdit() {
		editedRecord = { ...record };
		isEditing = true;
	}

	function handleDelete() {
		if (confirm(`Delete this ${page.page.caption}?`)) {
			onaction?.('Delete');
		}
	}

	function handleSave() {
		record = { ...editedRecord };
		onsave?.(record);
		isEditing = false;
	}

	function handleCancel() {
		editedRecord = { ...record };
		isEditing = false;
	}

	// Build keyboard shortcut map from actions
	const shortcutMap = $derived(() => {
		const map: Record<string, () => void> = {};

		page.page.actions?.forEach((action) => {
			if (action.shortcut && action.enabled !== false) {
				map[action.shortcut] = () => handleAction(action.name);
			}
		});

		return map;
	});

	// Get field caption (from captions or field definition)
	function getFieldCaption(fieldSource: string, fieldCaption?: string): string {
		return captions[fieldSource] || fieldCaption || fieldSource;
	}
</script>

<div class="card-page" use:shortcuts={shortcutMap()}>
	<PageHeader title={page.page.caption}>
		<svelte:fragment slot="actions">
			{#each page.page.actions?.filter((a) => a.promoted) || [] as action}
				<Button
					variant={action.name === 'Delete' ? 'danger' : 'secondary'}
					size="sm"
					on:click={() => handleAction(action.name)}
					disabled={action.enabled === false}
				>
					{action.caption}
					{#if action.shortcut}
						<span class="ml-2 text-xs opacity-70">{action.shortcut}</span>
					{/if}
				</Button>
			{/each}

			{#if isEditing}
				<Button variant="primary" size="sm" on:click={handleSave}>
					Save
				</Button>
				<Button variant="secondary" size="sm" on:click={handleCancel}>
					Cancel
				</Button>
			{/if}
		</svelte:fragment>
	</PageHeader>

	<div class="sections-container">
		{#each page.page.layout.sections || [] as section}
			<Card>
				<svelte:fragment slot="header">
					<h3 class="text-lg font-semibold text-nav-blue">{section.caption}</h3>
				</svelte:fragment>

				<div class="section-fields">
					{#each section.fields as field}
						<FieldRenderer
							{field}
							bind:value={editedRecord[field.source]}
							caption={getFieldCaption(field.source, field.caption)}
							editable={isEditing}
							readonly={!isEditing}
						/>
					{/each}
				</div>
			</Card>
		{/each}
	</div>
</div>

<style>
	.card-page {
		@apply flex flex-col gap-4 h-full;
	}

	.sections-container {
		@apply flex flex-col gap-4 overflow-y-auto;
	}

	.section-fields {
		@apply grid grid-cols-1 md:grid-cols-2 gap-4;
	}
</style>
