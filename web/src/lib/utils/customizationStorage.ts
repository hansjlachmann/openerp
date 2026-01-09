/**
 * Utility functions for managing page customization storage in localStorage
 * Supports user-specific and page-specific customizations
 */

/**
 * Generate storage key for page customizations
 * @param userId - The user ID (or 'anonymous' if not logged in)
 * @param pageId - The page ID
 * @returns Storage key string
 */
function getStorageKey(userId: string, pageId: string): string {
	return `page-customization-${userId}-${pageId}`;
}

/**
 * Load page customizations from localStorage
 * @param userId - The user ID (or 'anonymous' if not logged in)
 * @param pageId - The page ID
 * @returns Parsed customizations object, or empty object if none found or parse error
 */
export function loadPageCustomizations<T = Record<string, any>>(
	userId: string,
	pageId: string
): T {
	const key = getStorageKey(userId, pageId);
	const stored = localStorage.getItem(key);

	if (!stored) {
		return {} as T;
	}

	try {
		return JSON.parse(stored) as T;
	} catch (e) {
		console.error('Failed to load page customizations:', e);
		return {} as T;
	}
}

/**
 * Save page customizations to localStorage
 * @param userId - The user ID (or 'anonymous' if not logged in)
 * @param pageId - The page ID
 * @param customizations - The customizations object to save
 */
export function savePageCustomizations<T = Record<string, any>>(
	userId: string,
	pageId: string,
	customizations: T
): void {
	const key = getStorageKey(userId, pageId);
	try {
		localStorage.setItem(key, JSON.stringify(customizations));
	} catch (e) {
		console.error('Failed to save page customizations:', e);
	}
}

/**
 * Clear page customizations from localStorage
 * @param userId - The user ID (or 'anonymous' if not logged in)
 * @param pageId - The page ID
 */
export function clearPageCustomizations(userId: string, pageId: string): void {
	const key = getStorageKey(userId, pageId);
	localStorage.removeItem(key);
}

/**
 * Generate storage key for column widths
 */
function getColumnWidthsKey(userId: string, pageId: number): string {
	return `column-widths-${userId}-${pageId}`;
}

/**
 * Load column widths from localStorage
 */
export function loadColumnWidths(userId: string, pageId: number): Record<string, number> {
	const key = getColumnWidthsKey(userId, pageId);
	const stored = localStorage.getItem(key);

	if (!stored) {
		return {};
	}

	try {
		return JSON.parse(stored);
	} catch (e) {
		console.error('Failed to load column widths:', e);
		return {};
	}
}

/**
 * Save column widths to localStorage
 */
export function saveColumnWidths(
	userId: string,
	pageId: number,
	widths: Record<string, number>
): void {
	const key = getColumnWidthsKey(userId, pageId);
	try {
		localStorage.setItem(key, JSON.stringify(widths));
	} catch (e) {
		console.error('Failed to save column widths:', e);
	}
}

/**
 * Generate storage key for row numbers preference
 */
function getRowNumbersKey(userId: string, pageId: number): string {
	return `row-numbers-${userId}-${pageId}`;
}

/**
 * Load row numbers preference from localStorage
 */
export function loadRowNumbersPreference(userId: string, pageId: number): boolean {
	const key = getRowNumbersKey(userId, pageId);
	const stored = localStorage.getItem(key);
	return stored === 'true';
}

/**
 * Save row numbers preference to localStorage
 */
export function saveRowNumbersPreference(userId: string, pageId: number, show: boolean): void {
	const key = getRowNumbersKey(userId, pageId);
	localStorage.setItem(key, show ? 'true' : 'false');
}
