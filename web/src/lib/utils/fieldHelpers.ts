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
