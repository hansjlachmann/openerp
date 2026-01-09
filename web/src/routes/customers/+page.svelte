<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import PageHeader from '$components/PageHeader.svelte';
	import Card from '$components/Card.svelte';
	import { customerApi } from '$services/api';
	import { shortcuts, createShortcutMap } from '$utils/shortcuts';
	import type { Customer } from '$types/api';
	import { currentLanguage } from '$stores/session';
	import { toast } from '$lib/stores/toast';

	let customers: Customer[] = [];
	let loading = true;
	let error: string | null = null;
	let selectedIndex = 0;

	// Load customers on mount
	onMount(async () => {
		await loadCustomers();
	});

	async function loadCustomers() {
		try {
			loading = true;
			error = null;
			const response = await customerApi.list({
				sort_by: 'no',
				sort_order: 'asc'
			});
			customers = response.records;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load customers';
			console.error('Error loading customers:', err);
		} finally {
			loading = false;
		}
	}

	function handleNew() {
		goto('/customers/new');
	}

	function handleEdit() {
		if (customers[selectedIndex]) {
			goto(`/customers/${customers[selectedIndex].no}`);
		}
	}

	async function handleDelete() {
		if (!customers[selectedIndex]) return;

		const customer = customers[selectedIndex];
		if (confirm(`Delete customer ${customer.no} - ${customer.name}?`)) {
			try {
				await customerApi.delete(customer.no);
				await loadCustomers();
				toast.success('Customer deleted successfully');
			} catch (err) {
				toast.error(`Failed to delete: ${err instanceof Error ? err.message : 'Unknown error'}`);
			}
		}
	}

	function handleRefresh() {
		loadCustomers();
	}

	function handleFirst() {
		selectedIndex = 0;
		scrollToSelected();
	}

	function handleLast() {
		selectedIndex = customers.length - 1;
		scrollToSelected();
	}

	function handleNext() {
		if (selectedIndex < customers.length - 1) {
			selectedIndex++;
			scrollToSelected();
		}
	}

	function handlePrevious() {
		if (selectedIndex > 0) {
			selectedIndex--;
			scrollToSelected();
		}
	}

	function scrollToSelected() {
		const row = document.querySelector(`[data-index="${selectedIndex}"]`);
		row?.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
	}

	function selectRow(index: number) {
		selectedIndex = index;
	}

	function handleRowDoubleClick(customer: Customer) {
		goto(`/customers/${customer.no}`);
	}

	// Keyboard shortcuts
	const shortcutMap = createShortcutMap({
		onNew: handleNew,
		onEdit: handleEdit,
		onDelete: handleDelete,
		onRefresh: handleRefresh,
		onFirst: handleFirst,
		onLast: handleLast,
		onNext: handleNext,
		onPrevious: handlePrevious
	});

	const pageActions = [
		{ label: 'New', onClick: handleNew, variant: 'primary' as const, shortcut: 'Ctrl+N' },
		{ label: 'Edit', onClick: handleEdit, variant: 'secondary' as const, shortcut: 'Ctrl+E' },
		{ label: 'Delete', onClick: handleDelete, variant: 'danger' as const, shortcut: 'Ctrl+D' },
		{ label: 'Refresh', onClick: handleRefresh, variant: 'secondary' as const, shortcut: 'F5' }
	];
</script>

<svelte:head>
	<title>Customer List - OpenERP</title>
</svelte:head>

<div class="min-h-screen bg-gray-50" use:shortcuts={shortcutMap}>
	<PageHeader title="Customer List" subtitle="View and manage customers" actions={pageActions} />

	<div class="container mx-auto px-6 py-6">
		{#if loading}
			<Card>
				<div class="text-center py-12">
					<div class="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
					<p class="mt-4 text-gray-600">Loading customers...</p>
				</div>
			</Card>
		{:else if error}
			<Card>
				<div class="text-center py-12">
					<p class="text-red-600">{error}</p>
					<button class="mt-4 btn btn-secondary" on:click={loadCustomers}>Retry</button>
				</div>
			</Card>
		{:else if customers.length === 0}
			<Card>
				<div class="text-center py-12">
					<p class="text-gray-600">No customers found.</p>
					<button class="mt-4 btn btn-primary" on:click={handleNew}>Create First Customer</button>
				</div>
			</Card>
		{:else}
			<Card>
				<div class="overflow-x-auto">
					<table class="table">
						<thead>
							<tr>
								<th>No.</th>
								<th>Name</th>
								<th>City</th>
								<th>Phone No.</th>
								<th class="text-right">Balance (LCY)</th>
								<th class="text-right">Sales (LCY)</th>
								<th>Status</th>
							</tr>
						</thead>
						<tbody>
							{#each customers as customer, index}
								<tr
									data-index={index}
									class:selected={index === selectedIndex}
									tabindex="0"
									on:click={() => selectRow(index)}
									on:dblclick={() => handleRowDoubleClick(customer)}
									on:keydown={(e) => {
										if (e.key === 'Enter') handleRowDoubleClick(customer);
									}}
								>
									<td class="font-mono text-sm">{customer.no}</td>
									<td class="font-medium">{customer.name}</td>
									<td>{customer.city || ''}</td>
									<td>{customer.phone_number || ''}</td>
									<td class="text-right font-mono text-sm">{customer.balance_lcy || '0.00'}</td>
									<td class="text-right font-mono text-sm">{customer.sales_lcy || '0.00'}</td>
									<td>
										{#if customer.status === 0}
											<span class="px-2 py-1 text-xs rounded bg-green-100 text-green-800">Open</span>
										{:else if customer.status === 1}
											<span class="px-2 py-1 text-xs rounded bg-red-100 text-red-800">Blocked</span>
										{:else if customer.status === 2}
											<span class="px-2 py-1 text-xs rounded bg-gray-100 text-gray-800">Closed</span>
										{/if}
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>

				<div class="mt-4 text-sm text-gray-600 text-center">
					{customers.length} record{customers.length !== 1 ? 's' : ''}
					{#if customers.length > 0}
						| Selected: {selectedIndex + 1}
					{/if}
				</div>
			</Card>
		{/if}

		<!-- Keyboard shortcuts help -->
		<div class="mt-6 text-sm text-gray-500 text-center">
			<p>
				Use keyboard shortcuts: <span class="font-mono">Ctrl+N</span> New,
				<span class="font-mono">Ctrl+E</span> Edit,
				<span class="font-mono">Ctrl+D</span> Delete,
				<span class="font-mono">↑↓</span> Navigate,
				<span class="font-mono">Enter</span> Open
			</p>
		</div>
	</div>
</div>
