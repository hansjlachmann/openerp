<script lang="ts">
	import type { Field } from '$lib/types/pages';
	import type { LookupData } from '$lib/types/api';
	import { cn } from '$lib/utils/cn';
	import { getFieldStyleClasses, formatValue, formatOptionValue, formatLookupValue } from '$lib/utils/fieldHelpers';
	import LookupDropdown from './LookupDropdown.svelte';

	interface Props {
		field: Field;
		value: any;
		caption?: string;
		editable?: boolean;
		required?: boolean;
		error?: string;
		options?: Record<string, string>; // Option field values (key = stored value, value = display text)
		lookups?: LookupData; // Lookup values (advanced or simple mode)
		fieldCaptions?: Record<string, string>; // Field captions for lookup column headers
		tabindex?: number;
		onchange?: (value: any) => void;
		onblur?: () => void;
	}

	let {
		field,
		value = $bindable(),
		caption,
		editable = false,
		required = false,
		error,
		options,
		lookups,
		fieldCaptions = {},
		tabindex,
		onchange,
		onblur
	}: Props = $props();

	// Determine if field is editable
	const isEditable = $derived(editable);

	// Check if this is an option/enum field (has options provided)
	const isOptionField = $derived(options && Object.keys(options).length > 0);

	// Check if this is a lookup field with advanced columns (table-style dropdown)
	const isAdvancedLookup = $derived(
		lookups && lookups.columns && lookups.columns.length > 0 && lookups.rows
	);

	// Check if this is a simple lookup field (basic dropdown)
	const isSimpleLookup = $derived(
		lookups && lookups.simple && Object.keys(lookups.simple).length > 0 && !isAdvancedLookup
	);

	// Check if this is any lookup field
	const isLookupField = $derived(isAdvancedLookup || isSimpleLookup);

	// Get field caption (from props, field definition, or field source)
	const fieldCaption = $derived(caption || field.caption || field.source);

	// Determine field style classes based on metadata
	const fieldStyle = $derived(getFieldStyleClasses(field));

	// Handle value change for text inputs
	function handleChange(e: Event) {
		const target = e.target as HTMLInputElement;
		const newValue = target.value;
		value = newValue;
		onchange?.(newValue);
	}

	// Handle value change for select (option fields)
	function handleSelectChange(e: Event) {
		const target = e.target as HTMLSelectElement;
		// Convert back to number since options are stored as integers
		const newValue = parseInt(target.value, 10);
		value = newValue;
		onchange?.(newValue);
		// Trigger blur to save immediately after selection
		onblur?.();
	}

	// Handle value change for lookup select (table relation fields)
	function handleLookupSelectChange(e: Event) {
		const target = e.target as HTMLSelectElement;
		// Lookups store the code as a string
		const newValue = target.value;
		value = newValue;
		onchange?.(newValue);
		// Trigger blur to save immediately after selection
		onblur?.();
	}

	// Get display value for option field (convert stored integer to display text)
	const optionDisplayValue = $derived(() => formatOptionValue(value, options));

	// Get display value for lookup field (show "code - description" or just description)
	const lookupDisplayValue = $derived(() => formatLookupValue(value, lookups, !isAdvancedLookup));

	// Handle lookup selection from LookupDropdown
	function handleLookupSelect(key: string) {
		value = key;
		onchange?.(key);
	}

	// Determine input type based on field
	const inputType = $derived(() => {
		if (field.source === 'password') {
			return 'password';
		}
		if (field.source.includes('email')) {
			return 'email';
		}
		return 'text';
	});
</script>

{#if isEditable}
	<!-- Editable field -->
	<div class="field-group" class:has-error={!!error}>
		<label for={field.source} class="field-label">
			{fieldCaption}
			{#if required}
				<span class="required-indicator">*</span>
			{/if}
		</label>
		<div class="input-wrapper">
			{#if isOptionField && options}
				<!-- Option/Enum field - render as dropdown -->
				<select
					id={field.source}
					class={cn('select', fieldStyle, error ? 'input-error' : '')}
					value={String(value ?? 0)}
					{tabindex}
					onchange={handleSelectChange}
					onblur={() => onblur?.()}
					aria-invalid={!!error}
					aria-describedby={error ? `${field.source}-error` : undefined}
				>
					{#each Object.entries(options) as [optValue, optLabel]}
						<option value={optValue}>{optLabel}</option>
					{/each}
				</select>
			{:else if isAdvancedLookup && lookups?.columns && lookups?.rows}
				<!-- Advanced lookup with columns - render as table-style dropdown -->
				<LookupDropdown
					columns={[...lookups.columns]}
					rows={[...lookups.rows]}
					value={value || ''}
					fieldName={fieldCaption}
					captions={fieldCaptions}
					searchTimeout={lookups.search_timeout}
					{tabindex}
					error={!!error}
					onselect={handleLookupSelect}
					onblur={() => onblur?.()}
				/>
			{:else if isSimpleLookup && lookups?.simple}
				<!-- Simple lookup - render as basic dropdown -->
				<select
					id={field.source}
					class={cn('select', fieldStyle, error ? 'input-error' : '')}
					value={value ?? ''}
					{tabindex}
					onchange={handleLookupSelectChange}
					onblur={() => onblur?.()}
					aria-invalid={!!error}
					aria-describedby={error ? `${field.source}-error` : undefined}
				>
					<option value=""></option>
					{#each Object.entries(lookups!.simple!) as [lookupCode, lookupDisplay]}
						<option value={lookupCode} title={lookupDisplay}>{lookupCode}</option>
					{/each}
				</select>
			{:else}
				<!-- Regular text field -->
				<input
					id={field.source}
					type={inputType()}
					class={cn('input', fieldStyle, error ? 'input-error' : '')}
					value={value ?? ''}
					{tabindex}
					oninput={handleChange}
					onblur={() => onblur?.()}
					aria-invalid={!!error}
					aria-describedby={error ? `${field.source}-error` : undefined}
				/>
			{/if}
		</div>
		{#if error}
			<p id="{field.source}-error" class="error-message">{error}</p>
		{/if}
	</div>
{:else}
	<!-- Read-only field -->
	<div class="field-group readonly">
		<div class="field-label">
			{fieldCaption}
		</div>
		{#if isAdvancedLookup && lookups?.columns && lookups?.rows}
			<!-- Advanced lookup - show dropdown even in non-edit mode -->
			<div class="input-wrapper">
				<LookupDropdown
					columns={[...lookups.columns]}
					rows={[...lookups.rows]}
					value={value || ''}
					fieldName={fieldCaption}
					captions={fieldCaptions}
					searchTimeout={lookups.search_timeout}
					{tabindex}
					error={false}
					onselect={handleLookupSelect}
					onblur={() => onblur?.()}
				/>
			</div>
		{:else if isSimpleLookup && lookups?.simple}
			<!-- Simple lookup - show dropdown even in non-edit mode -->
			<div class="input-wrapper">
				<select
					id={field.source}
					class={cn('select', fieldStyle)}
					value={value ?? ''}
					{tabindex}
					onchange={handleLookupSelectChange}
					onblur={() => onblur?.()}
				>
					<option value=""></option>
					{#each Object.entries(lookups!.simple!) as [lookupCode, lookupDisplay]}
						<option value={lookupCode} title={lookupDisplay}>{lookupCode}</option>
					{/each}
				</select>
			</div>
		{:else}
			<div class={cn('field-value', fieldStyle)}>
				{#if isOptionField}
					{optionDisplayValue()}
				{:else}
					{formatValue(value)}
				{/if}
			</div>
		{/if}
	</div>
{/if}

<style>
	.field-group {
		@apply flex flex-col gap-1;
	}

	.field-label {
		@apply text-sm font-medium text-gray-700;
		@apply flex items-center gap-1;
	}

	:global(.dark) .field-label {
		color: var(--color-text-secondary);
	}

	.required-indicator {
		@apply text-red-500 font-bold;
	}

	.input-wrapper {
		@apply relative;
	}

	.field-group :global(input.input),
	.field-group :global(select.select) {
		@apply w-full py-2 px-3;
		@apply bg-white border border-gray-300 rounded-md;
		@apply text-gray-900 text-base;
		@apply transition-all duration-150;
		@apply outline-none;
	}

	.field-group :global(select.select) {
		@apply cursor-pointer;
		@apply appearance-none;
		background-image: url("data:image/svg+xml,%3csvg xmlns='http://www.w3.org/2000/svg' fill='none' viewBox='0 0 20 20'%3e%3cpath stroke='%236b7280' stroke-linecap='round' stroke-linejoin='round' stroke-width='1.5' d='M6 8l4 4 4-4'/%3e%3c/svg%3e");
		background-position: right 0.5rem center;
		background-repeat: no-repeat;
		background-size: 1.5em 1.5em;
		padding-right: 2.5rem;
	}

	.field-group :global(input.input:hover:not(:focus)),
	.field-group :global(select.select:hover:not(:focus)) {
		@apply border-gray-400;
	}

	.field-group :global(input.input:focus),
	.field-group :global(select.select:focus) {
		@apply border-blue-500 ring-2 ring-blue-500/20;
	}

	.field-group.has-error :global(input.input),
	.field-group.has-error :global(select.select) {
		@apply border-red-500;
	}

	.field-group.has-error :global(input.input:focus),
	.field-group.has-error :global(select.select:focus) {
		@apply border-red-500 ring-2 ring-red-500/20;
	}

	/* Dark mode input styles */
	:global(.dark) .field-group :global(input.input),
	:global(.dark) .field-group :global(select.select) {
		background-color: var(--color-bg-input);
		border-color: var(--color-border-secondary);
		color: var(--color-text-primary);
	}

	:global(.dark) .field-group :global(select.select) {
		background-image: url("data:image/svg+xml,%3csvg xmlns='http://www.w3.org/2000/svg' fill='none' viewBox='0 0 20 20'%3e%3cpath stroke='%239ca3af' stroke-linecap='round' stroke-linejoin='round' stroke-width='1.5' d='M6 8l4 4 4-4'/%3e%3c/svg%3e");
	}

	:global(.dark) .field-group :global(select.select) option {
		background-color: var(--color-bg-primary);
		color: var(--color-text-primary);
	}

	:global(.dark) .field-group :global(input.input:hover:not(:focus)),
	:global(.dark) .field-group :global(select.select:hover:not(:focus)) {
		border-color: var(--color-text-muted);
	}

	:global(.dark) .field-group :global(input.input:focus),
	:global(.dark) .field-group :global(select.select:focus) {
		border-color: #3b82f6; /* blue-500 */
		box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.3);
	}

	:global(.dark) .field-group :global(input.input::placeholder) {
		color: var(--color-text-muted);
	}

	.error-message {
		@apply text-xs text-red-600 mt-1;
	}

	:global(.dark) .error-message {
		color: #fca5a5; /* red-300 */
	}

	.field-value {
		@apply text-base py-2 px-3;
		@apply bg-gray-50 border border-gray-200 rounded-md;
		@apply text-gray-900;
		min-height: 2.5rem;
		@apply flex items-center;
	}

	:global(.dark) .field-value {
		background-color: var(--color-bg-input);
		border-color: var(--color-border-secondary);
		color: var(--color-text-primary);
	}

	/* Strong style for important values */
	.field-value :global(.strong) {
		@apply font-semibold;
	}

	/* Readonly visual distinction */
	.field-group.readonly .field-value {
		@apply bg-gray-100;
	}

	:global(.dark) .field-group.readonly .field-value {
		background-color: var(--color-bg-primary);
	}
</style>
