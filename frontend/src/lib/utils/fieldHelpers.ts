import type { Field } from '$lib/types/pages';

/**
 * Get the display caption for a field
 * @param fieldSource - The field source name
 * @param captions - Record of field captions from the page definition
 * @param fieldCaption - Optional caption from the field definition
 * @returns The caption to display
 */
export function getFieldCaption(
	fieldSource: string,
	captions: Record<string, string>,
	fieldCaption?: string
): string {
	return captions[fieldSource] || fieldCaption || fieldSource;
}

/**
 * Get CSS classes for field styling based on field properties
 * @param field - The field object with style and importance properties
 * @returns Space-separated CSS class string
 */
export function getFieldStyleClasses(field: Field | { style?: string; importance?: string }): string {
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
}

/**
 * Format a value for display
 * @param val - The value to format
 * @returns Formatted string representation
 */
export function formatValue(val: any): string {
	if (val === null || val === undefined) {
		return '';
	}
	if (typeof val === 'boolean') {
		return val ? 'Yes' : 'No';
	}
	return String(val);
}

/**
 * Format an option field value (convert stored integer to display text)
 * @param value - The stored value (typically an integer index)
 * @param options - Map of option values to display labels
 * @returns The display label or formatted value
 */
export function formatOptionValue(
	value: any,
	options?: Record<string, string>
): string {
	if (value === undefined || value === null) {
		return '';
	}
	if (options) {
		const stringValue = String(value);
		return options[stringValue] || stringValue;
	}
	return formatValue(value);
}

/**
 * Format a lookup field value for display
 * @param value - The stored key value
 * @param lookups - Lookup data with simple map or rows/columns
 * @param showKeyWithDescription - Whether to show "key - description" format
 * @returns The display value
 */
export function formatLookupValue(
	value: any,
	lookups?: { simple?: Record<string, string>; rows?: any[]; columns?: { source: string }[] },
	showKeyWithDescription: boolean = true
): string {
	if (value === undefined || value === null || value === '') {
		return '';
	}
	if (!lookups) {
		return String(value);
	}

	const stringValue = String(value);

	// For advanced lookup with rows, get description from first column
	if (lookups.rows && lookups.columns && lookups.columns.length > 0) {
		const row = lookups.rows.find(r => r._key === stringValue);
		if (row) {
			return row[lookups.columns[0].source] ?? stringValue;
		}
	}

	// For simple lookup
	if (lookups.simple) {
		const description = lookups.simple[stringValue];
		if (description && description !== stringValue && showKeyWithDescription) {
			return `${stringValue} - ${description}`;
		}
		if (description) {
			return description;
		}
	}

	return stringValue;
}

/**
 * Item customization type for visibility checks
 */
export interface ItemCustomization {
	visible: boolean;
	order?: number;
	section?: string;
}

/**
 * Check if a field/column is visible based on customizations
 * @param field - The field object with source and visible properties
 * @param customizations - User customizations for the page
 * @returns Whether the field should be visible
 */
export function isItemVisible(
	field: { source: string; visible?: boolean },
	customizations: Record<string, ItemCustomization>
): boolean {
	// If user has customized this field, use that preference
	if (field.source in customizations) {
		return customizations[field.source].visible;
	}
	// Otherwise use the field's visible property (default true)
	return field.visible !== false;
}
