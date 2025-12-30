<script lang="ts">
	import type { Field } from '$lib/types/pages';
	import { cn } from '$lib/utils/cn';

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
	const fieldStyle = $derived(() => {
		const classes: string[] = [];

		// Importance styling
		if (field.importance === 'Promoted') {
			classes.push('font-semibold');
		}

		// Style-based coloring
		switch (field.style) {
			case 'Strong':
				classes.push('text-nav-blue dark:text-blue-400 font-bold');
				break;
			case 'Attention':
				classes.push('text-orange-600 dark:text-orange-400 font-medium');
				break;
			case 'Favorable':
				classes.push('text-green-600 dark:text-green-400');
				break;
			case 'Unfavorable':
				classes.push('text-red-600 dark:text-red-400');
				break;
		}

		return classes.join(' ');
	});

	// Handle value change
	function handleChange(e: Event) {
		const target = e.target as HTMLInputElement;
		const newValue = target.value;
		value = newValue;
		onchange?.(newValue);
	}

	// Format value for display
	function formatValue(val: any): string {
		if (val === null || val === undefined) {
			return '';
		}
		if (typeof val === 'boolean') {
			return val ? 'Yes' : 'No';
		}
		return String(val);
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
			class={cn('input', fieldStyle())}
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
		<div class={cn('field-value', fieldStyle())}>
			{formatValue(value)}
		</div>
	</div>
{/if}

<style>
	.field-group {
		@apply flex flex-col gap-1;
	}

	.field-label {
		@apply text-sm font-medium text-gray-700 dark:text-gray-300;
	}

	.field-value {
		@apply text-base py-1.5 px-3 bg-gray-50 border border-gray-200 rounded;
		@apply dark:bg-gray-700 dark:border-gray-600;
		min-height: 2.5rem;
	}
</style>
