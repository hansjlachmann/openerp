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
