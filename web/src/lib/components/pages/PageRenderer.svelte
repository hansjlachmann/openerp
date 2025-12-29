<script lang="ts">
	import { onMount } from 'svelte';
	import type { PageDefinition } from '$lib/types/pages';
	import { fetchPage } from '$lib/services/pages';
	import { api } from '$lib/services/api';
	import CardPage from './CardPage.svelte';
	import ListPage from './ListPage.svelte';

	interface Props {
		pageid: number;
		recordid?: string;
	}

	let { pageid, recordid }: Props = $props();

	// State
	let page: PageDefinition | null = $state(null);
	let captions: Record<string, string> = $state({});
	let loading = $state(true);
	let error = $state<string | null>(null);

	// Data for the page
	let record: Record<string, any> = $state({});
	let records: Array<Record<string, any>> = $state([]);

	// Load page definition and data
	onMount(async () => {
		try {
			loading = true;
			error = null;

			// Fetch page definition
			const response = await fetch(`/api/pages/${pageid}`);
			if (!response.ok) {
				throw new Error(`Failed to load page: ${response.statusText}`);
			}

			const result = await response.json();
			if (!result.success) {
				throw new Error(result.error || 'Failed to load page');
			}

			page = result.data;
			captions = result.captions?.fields || {};

			// Load data based on page type
			if (page.page.type === 'Card') {
				await loadCardData();
			} else if (page.page.type === 'List') {
				await loadListData();
			}
		} catch (err) {
			error = err instanceof Error ? err.message : 'Unknown error';
			console.error('Error loading page:', err);
		} finally {
			loading = false;
		}
	});

	// Load data for card page
	async function loadCardData() {
		if (!page) return;

		try {
			if (recordid) {
				// Load specific record
				const response = await api.getRecord(page.page.source_table, recordid);
				if (response.success) {
					record = response.data;
				}
			} else {
				// New record
				record = {};
			}
		} catch (err) {
			console.error('Error loading card data:', err);
		}
	}

	// Load data for list page
	async function loadListData() {
		if (!page) return;

		try {
			const response = await api.listRecords(page.page.source_table);
			if (response.success) {
				records = response.data;
			}
		} catch (err) {
			console.error('Error loading list data:', err);
		}
	}

	// Handle actions from card page
	async function handleCardAction(actionName: string) {
		if (!page) return;

		switch (actionName) {
			case 'New':
				record = {};
				break;
			case 'Delete':
				if (recordid) {
					try {
						await api.deleteRecord(page.page.source_table, recordid);
						// Navigate back or show message
						alert('Record deleted');
					} catch (err) {
						alert('Failed to delete record');
					}
				}
				break;
			case 'Refresh':
				await loadCardData();
				break;
		}
	}

	// Handle save from card page
	async function handleCardSave(savedRecord: Record<string, any>) {
		if (!page) return;

		try {
			if (recordid) {
				// Update existing record
				await api.modifyRecord(page.page.source_table, recordid, savedRecord);
				alert('Record updated');
			} else {
				// Insert new record
				const response = await api.insertRecord(page.page.source_table, savedRecord);
				if (response.success) {
					alert('Record created');
					// Could navigate to the new record
				}
			}
		} catch (err) {
			alert('Failed to save record');
			console.error('Save error:', err);
		}
	}

	// Handle actions from list page
	async function handleListAction(actionName: string, selectedRecord?: Record<string, any>) {
		if (!page) return;

		switch (actionName) {
			case 'New':
				// Navigate to card page in new mode
				if (page.page.card_page_id) {
					window.location.href = `/pages/${page.page.card_page_id}`;
				}
				break;
			case 'Edit':
				if (selectedRecord && page.page.card_page_id) {
					// Navigate to card page with record ID
					const recordId = selectedRecord['no'] || selectedRecord['code'] || selectedRecord['id'];
					window.location.href = `/pages/${page.page.card_page_id}/${recordId}`;
				}
				break;
			case 'Delete':
				if (selectedRecord) {
					const recordId = selectedRecord['no'] || selectedRecord['code'] || selectedRecord['id'];
					if (confirm('Delete this record?')) {
						try {
							await api.deleteRecord(page.page.source_table, recordId);
							await loadListData();
							alert('Record deleted');
						} catch (err) {
							alert('Failed to delete record');
						}
					}
				}
				break;
			case 'Refresh':
				await loadListData();
				break;
		}
	}

	// Handle row click in list page
	function handleRowClick(clickedRecord: Record<string, any>) {
		if (!page || !page.page.card_page_id) return;

		const recordId = clickedRecord['no'] || clickedRecord['code'] || clickedRecord['id'];
		window.location.href = `/pages/${page.page.card_page_id}/${recordId}`;
	}
</script>

{#if loading}
	<div class="flex items-center justify-center h-full">
		<div class="text-center">
			<div class="animate-spin rounded-full h-12 w-12 border-b-2 border-nav-blue mx-auto mb-4"></div>
			<p class="text-gray-600">Loading page...</p>
		</div>
	</div>
{:else if error}
	<div class="flex items-center justify-center h-full">
		<div class="text-center">
			<div class="text-red-600 text-5xl mb-4">⚠</div>
			<h2 class="text-xl font-semibold text-gray-800 mb-2">Error Loading Page</h2>
			<p class="text-gray-600">{error}</p>
		</div>
	</div>
{:else if page}
	{#if page.page.type === 'Card'}
		<CardPage
			{page}
			bind:record
			{captions}
			onaction={handleCardAction}
			onsave={handleCardSave}
		/>
	{:else if page.page.type === 'List'}
		<ListPage {page} {records} {captions} onaction={handleListAction} onrowclick={handleRowClick} />
	{:else}
		<div class="flex items-center justify-center h-full">
			<div class="text-center">
				<p class="text-gray-600">Page type "{page.page.type}" not yet supported</p>
			</div>
		</div>
	{/if}
{/if}
