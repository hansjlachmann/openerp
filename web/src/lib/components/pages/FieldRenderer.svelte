<script lang="ts">
	import type { Field } from '$lib/types/pages';
	import { cn } from '$lib/utils/cn';
	import { getFieldStyleClasses, formatValue } from '$lib/utils/fieldHelpers';

	interface Props {
		field: Field;
		value: any;
		caption?: string;
		editable?: boolean;
		onchange?: (value: any) => void;
		onblur?: () => void;
	}

	let {
		field,
		value = $bindable(),
		caption,
		editable = false,
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
		return 'text';
	});
</script>

{#if isEditable}
	<!-- Editable field -->
	<div class="field-group">
		<label for={field.source} class="field-label">
			{fieldCaption}
		</label>
		<input
			id={field.source}
			type={inputType()}
			class={cn('input', fieldStyle)}
			value={value}
			oninput={handleChange}
			onblur={() => onblur?.()}
		/>
	</div>
{:else}
	<!-- Read-only field -->
	<div class="field-group">
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
	}

	:global(.dark) .field-label {
		color: #d1d5db; /* gray-300 */
	}

	.field-value {
		@apply text-base py-1.5 px-3 bg-gray-50 border border-gray-200 rounded;
		min-height: 2.5rem;
	}

	:global(.dark) .field-value {
		background-color: #374151; /* gray-700 */
		border-color: #4b5563; /* gray-600 */
		color: #f3f4f6; /* gray-100 */
	}

	/* Override input styles for dark mode */
	:global(.dark) .field-group input.input {
		background-color: #374151 !important; /* gray-700 */
		border-color: #4b5563 !important; /* gray-600 */
		color: #f3f4f6 !important; /* gray-100 */
	}
</style>
