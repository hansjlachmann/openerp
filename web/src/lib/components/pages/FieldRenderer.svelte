<script lang="ts">
	import type { Field } from '$lib/types/pages';
	import { cn } from '$lib/utils/cn';
	import { getFieldStyleClasses, formatValue } from '$lib/utils/fieldHelpers';

	interface Props {
		field: Field;
		value: any;
		caption?: string;
		editable?: boolean;
		required?: boolean;
		error?: string;
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
		onchange,
		onblur
	}: Props = $props();

	// Determine if field is editable
	const isEditable = $derived(editable);

	// Get field caption (from props, field definition, or field source)
	const fieldCaption = $derived(caption || field.caption || field.source);

	// Determine field style classes based on metadata
	const fieldStyle = $derived(getFieldStyleClasses(field));

	// Handle value change
	function handleChange(e: Event) {
		const target = e.target as HTMLInputElement;
		const newValue = target.value;
		value = newValue;
		onchange?.(newValue);
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
			<input
				id={field.source}
				type={inputType()}
				class={cn('input', fieldStyle, error ? 'input-error' : '')}
				value={value ?? ''}
				oninput={handleChange}
				onblur={() => onblur?.()}
				aria-invalid={!!error}
				aria-describedby={error ? `${field.source}-error` : undefined}
			/>
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
		<div class={cn('field-value', fieldStyle)}>
			{formatValue(value)}
		</div>
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

	.field-group :global(input.input) {
		@apply w-full py-2 px-3;
		@apply bg-white border border-gray-300 rounded-md;
		@apply text-gray-900 text-base;
		@apply transition-all duration-150;
		@apply outline-none;
	}

	.field-group :global(input.input:hover:not(:focus)) {
		@apply border-gray-400;
	}

	.field-group :global(input.input:focus) {
		@apply border-blue-500 ring-2 ring-blue-500/20;
	}

	.field-group.has-error :global(input.input) {
		@apply border-red-500;
	}

	.field-group.has-error :global(input.input:focus) {
		@apply border-red-500 ring-2 ring-red-500/20;
	}

	/* Dark mode input styles */
	:global(.dark) .field-group :global(input.input) {
		background-color: var(--color-bg-input);
		border-color: var(--color-border-secondary);
		color: var(--color-text-primary);
	}

	:global(.dark) .field-group :global(input.input:hover:not(:focus)) {
		border-color: var(--color-text-muted);
	}

	:global(.dark) .field-group :global(input.input:focus) {
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
