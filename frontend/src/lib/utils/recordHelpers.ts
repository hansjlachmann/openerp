// Record utility functions

/**
 * Extract the primary key/ID from a record
 * Records can use 'no', 'code', 'user_id', or 'id' as their primary key field
 */
export function getRecordId(record: Record<string, any> | null | undefined): string | undefined {
	if (!record) return undefined;
	return record.no || record.code || record.user_id || record.id;
}

/**
 * Extract a display label from a record (includes user_id for user records)
 */
export function getRecordLabel(record: Record<string, any> | null | undefined): string | undefined {
	if (!record) return undefined;
	return record.no || record.code || record.user_id || record.id;
}

/**
 * Check if a record is new (has no ID)
 */
export function isNewRecord(record: Record<string, any> | null | undefined): boolean {
	return !getRecordId(record);
}

/**
 * Deep copy an object using JSON serialization
 * Note: This won't work with functions, undefined, symbols, or circular references
 */
export function deepCopy<T>(obj: T): T {
	return JSON.parse(JSON.stringify(obj));
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
