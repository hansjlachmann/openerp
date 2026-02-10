// Record utility functions
import type { PageDefinition, Field } from '$lib/types/pages';

/**
 * Get the primary key field name from a page definition
 * Searches through sections (Card pages) and repeater (List pages) for a field with primary_key: true
 */
export function getPrimaryKeyField(page: PageDefinition | null | undefined): string | undefined {
	if (!page) return undefined;

	// Check repeater fields (List pages)
	if (page.page.layout.repeater?.fields) {
		const pkField = page.page.layout.repeater.fields.find((f: Field) => f.primary_key);
		if (pkField) return pkField.source;
	}

	// Check section fields (Card pages)
	if (page.page.layout.sections) {
		for (const section of page.page.layout.sections) {
			const pkField = section.fields.find((f: Field) => f.primary_key);
			if (pkField) return pkField.source;
		}
	}

	return undefined;
}

/**
 * Get all primary key field names from a page definition (supports composite keys)
 * Searches through sections (Card pages) and repeater (List pages) for fields with primary_key: true
 */
export function getPrimaryKeyFields(page: PageDefinition | null | undefined): string[] {
	if (!page) return [];

	const pkFields: string[] = [];

	// Check repeater fields (List pages)
	if (page.page.layout.repeater?.fields) {
		for (const f of page.page.layout.repeater.fields) {
			if (f.primary_key) pkFields.push(f.source);
		}
	}

	// Check section fields (Card pages)
	if (page.page.layout.sections) {
		for (const section of page.page.layout.sections) {
			for (const f of section.fields) {
				if (f.primary_key) pkFields.push(f.source);
			}
		}
	}

	return pkFields;
}

/**
 * Extract the primary key/ID from a record
 * @param record - The record to extract ID from
 * @param primaryKeyField - Optional primary key field name (from page definition)
 * @param primaryKeyFields - Optional array of PK field names for composite keys
 *
 * For composite keys: joins all PK values with comma (e.g., "HANS2,READER,TEST-COMPANY")
 * For single keys: returns the PK value directly
 * Otherwise falls back to checking common field names: no, code, user_id, id
 */
export function getRecordId(
	record: Record<string, any> | null | undefined,
	primaryKeyField?: string,
	primaryKeyFields?: string[]
): string | undefined {
	if (!record) return undefined;

	// Composite key: join all PK values with comma
	// Note: empty strings are valid PK values (e.g., blank company = all companies)
	if (primaryKeyFields && primaryKeyFields.length > 1) {
		const values = primaryKeyFields.map(f => record[f]);
		if (values.every(v => v !== undefined)) {
			return values.map(v => String(v ?? '')).join(',');
		}
		return undefined;
	}

	// If primary key field is specified, use it directly
	if (primaryKeyField && record[primaryKeyField] !== undefined && record[primaryKeyField] !== '') {
		return String(record[primaryKeyField]);
	}

	// Fallback to common field names for backwards compatibility
	if (record.no !== undefined && record.no !== '') return String(record.no);
	if (record.code !== undefined && record.code !== '') return String(record.code);
	if (record.user_id !== undefined && record.user_id !== '') return String(record.user_id);
	if (record.id !== undefined && record.id !== '') return String(record.id);

	return undefined;
}

/**
 * Extract a display label from a record
 * @param record - The record to extract label from
 * @param primaryKeyField - Optional primary key field name (from page definition)
 */
export function getRecordLabel(
	record: Record<string, any> | null | undefined,
	primaryKeyField?: string
): string | undefined {
	return getRecordId(record, primaryKeyField);
}

/**
 * Check if a record is new (has no ID)
 * @param record - The record to check
 * @param primaryKeyField - Optional primary key field name (from page definition)
 */
export function isNewRecord(
	record: Record<string, any> | null | undefined,
	primaryKeyField?: string
): boolean {
	return !getRecordId(record, primaryKeyField);
}

/**
 * Get the record key for Svelte's keyed each blocks
 * Uses _tempId for new unsaved records, otherwise uses the primary key value
 * @param record - The record to get key from
 * @param primaryKeyField - Optional primary key field name (from page definition)
 * @param primaryKeyFields - Optional array of PK field names for composite keys
 */
export function getRecordKey(
	record: Record<string, any>,
	primaryKeyField?: string,
	primaryKeyFields?: string[]
): string {
	// For new unsaved records, use the temporary ID
	if (record._tempId) return record._tempId;

	// Use the composite or single primary key value
	const id = getRecordId(record, primaryKeyField, primaryKeyFields);
	if (id) return id;

	// Last resort fallback - should not happen in practice
	return `record-${Math.random().toString(36).substr(2, 9)}`;
}

/**
 * Deep copy an object using JSON serialization
 * Note: This won't work with functions, undefined, symbols, or circular references
 */
export function deepCopy<T>(obj: T): T {
	return JSON.parse(JSON.stringify(obj));
}

/**
 * Check if a record is empty (contains no user data, only internal flags)
 * @param record - The record to check
 */
export function isEmptyRecord(record: Record<string, any>): boolean {
	return !Object.keys(record).some(key =>
		!key.startsWith('_') && record[key] !== undefined && record[key] !== ''
	);
}

/**
 * Check if a record has any user-entered data (ignoring internal flags)
 * @param record - The record to check
 */
export function hasRecordData(record: Record<string, any>): boolean {
	return Object.keys(record).some(key =>
		!key.startsWith('_') && record[key] !== undefined && record[key] !== ''
	);
}

/**
 * Check if a record has changed from its original state
 * Handles type coercion for number/string comparisons
 */
export function hasRecordChanged(
	current: Record<string, any>,
	original: Record<string, any>
): boolean {
	const currentKeys = Object.keys(current);
	for (const key of currentKeys) {
		// Skip internal fields
		if (key.startsWith('_')) continue;

		const currentVal = current[key];
		const originalVal = original[key];

		// Handle null/undefined/empty string as equivalent
		const currentEmpty = currentVal == null || currentVal === '';
		const originalEmpty = originalVal == null || originalVal === '';
		if (currentEmpty && originalEmpty) continue;
		if (currentEmpty || originalEmpty) return true;

		// Compare values with type coercion for numbers/strings
		if (typeof currentVal === 'number' || typeof originalVal === 'number') {
			if (Number(currentVal) !== Number(originalVal)) return true;
		} else if (String(currentVal) !== String(originalVal)) {
			return true;
		}
	}
	return false;
}
