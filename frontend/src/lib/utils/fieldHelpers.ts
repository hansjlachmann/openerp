import type { Field } from '$lib/types/pages';

// --- Date/DateTime type detection and formatting ---

/** Check if a field type represents a Date (from YAML "types.Date" or Go "Date") */
export function isDateType(fieldType: string | undefined): boolean {
	if (!fieldType) return false;
	return fieldType === 'types.Date' || fieldType === 'Date';
}

/** Check if a field type represents a DateTime (from YAML "types.DateTime" or Go "DateTime") */
export function isDateTimeType(fieldType: string | undefined): boolean {
	if (!fieldType) return false;
	return fieldType === 'types.DateTime' || fieldType === 'DateTime';
}

/** Format an ISO date string (YYYY-MM-DD) to locale display format */
export function formatDate(isoDate: string, locale: string): string {
	if (!isoDate) return '';
	// Parse YYYY-MM-DD manually to avoid timezone issues
	const parts = isoDate.split('T')[0].split('-');
	if (parts.length !== 3) return isoDate;
	const year = parseInt(parts[0], 10);
	const month = parseInt(parts[1], 10) - 1;
	const day = parseInt(parts[2], 10);
	if (isNaN(year) || isNaN(month) || isNaN(day)) return isoDate;
	const date = new Date(year, month, day);
	try {
		return new Intl.DateTimeFormat(locale, {
			year: 'numeric',
			month: '2-digit',
			day: '2-digit'
		}).format(date);
	} catch {
		return isoDate;
	}
}

/** Format an ISO datetime string to locale display format */
export function formatDateTime(isoDateTime: string, locale: string): string {
	if (!isoDateTime) return '';
	try {
		const date = new Date(isoDateTime);
		if (isNaN(date.getTime())) return isoDateTime;
		return new Intl.DateTimeFormat(locale, {
			year: 'numeric',
			month: '2-digit',
			day: '2-digit',
			hour: '2-digit',
			minute: '2-digit'
		}).format(date);
	} catch {
		return isoDateTime;
	}
}

/** Get a locale-specific date format pattern string (e.g., "DD.MM.YYYY" for nb-NO) */
export function getDateFormatPattern(locale: string): string {
	try {
		const parts = new Intl.DateTimeFormat(locale, {
			year: 'numeric',
			month: '2-digit',
			day: '2-digit'
		}).formatToParts(new Date(2026, 0, 15)); // Jan 15, 2026
		return parts
			.map((p) => {
				switch (p.type) {
					case 'day':
						return 'DD';
					case 'month':
						return 'MM';
					case 'year':
						return 'YYYY';
					case 'literal':
						return p.value;
					default:
						return '';
				}
			})
			.join('');
	} catch {
		return 'YYYY-MM-DD';
	}
}

/** Parse a locale-formatted date string back to ISO YYYY-MM-DD */
export function parseLocaleDate(input: string, locale: string): string | null {
	if (!input) return null;
	// If already ISO format, return as-is
	if (/^\d{4}-\d{2}-\d{2}$/.test(input)) return input;
	try {
		// Determine part order from locale
		const parts = new Intl.DateTimeFormat(locale, {
			year: 'numeric',
			month: '2-digit',
			day: '2-digit'
		}).formatToParts(new Date(2026, 0, 15));
		const order = parts
			.filter((p) => p.type === 'day' || p.type === 'month' || p.type === 'year')
			.map((p) => p.type);
		const separator = parts.find((p) => p.type === 'literal')?.value || '/';
		const inputParts = input.split(separator).map((s) => s.trim());
		if (inputParts.length !== 3) return null;
		let day = 0,
			month = 0,
			year = 0;
		for (let i = 0; i < 3; i++) {
			const num = parseInt(inputParts[i], 10);
			if (isNaN(num)) return null;
			switch (order[i]) {
				case 'day':
					day = num;
					break;
				case 'month':
					month = num;
					break;
				case 'year':
					year = num;
					break;
			}
		}
		if (year < 100) year += 2000;
		if (month < 1 || month > 12 || day < 1 || day > 31) return null;
		return `${String(year).padStart(4, '0')}-${String(month).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
	} catch {
		return null;
	}
}

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
