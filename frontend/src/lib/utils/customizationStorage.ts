/**
 * Utility functions for managing page customization storage in localStorage
 * Supports user-specific and page-specific customizations
 */

import { getJson, setJson, getBoolean, setBoolean, remove } from './storage';

/**
 * Generate storage key for page customizations
 */
function getStorageKey(userId: string, pageId: string | number): string {
	return `page-customization-${userId}-${pageId}`;
}

/**
 * Generate storage key for column widths
 */
function getColumnWidthsKey(userId: string, pageId: number): string {
	return `column-widths-${userId}-${pageId}`;
}

/**
 * Generate storage key for row numbers preference
 */
function getRowNumbersKey(userId: string, pageId: number): string {
	return `row-numbers-${userId}-${pageId}`;
}

/**
 * Load page customizations from localStorage
 * @param userId - The user ID (or 'anonymous' if not logged in)
 * @param pageId - The page ID
 * @returns Parsed customizations object, or empty object if none found
 */
export function loadPageCustomizations<T = Record<string, any>>(
	userId: string,
	pageId: string | number
): T {
	return getJson<T>(getStorageKey(userId, pageId), {} as T);
}

/**
 * Save page customizations to localStorage
 * @param userId - The user ID (or 'anonymous' if not logged in)
 * @param pageId - The page ID
 * @param customizations - The customizations object to save
 */
export function savePageCustomizations<T = Record<string, any>>(
	userId: string,
	pageId: string | number,
	customizations: T
): void {
	setJson(getStorageKey(userId, pageId), customizations);
}

/**
 * Clear page customizations from localStorage
 * @param userId - The user ID (or 'anonymous' if not logged in)
 * @param pageId - The page ID
 */
export function clearPageCustomizations(userId: string, pageId: string | number): void {
	remove(getStorageKey(userId, pageId));
}

/**
 * Load column widths from localStorage
 */
export function loadColumnWidths(userId: string, pageId: number): Record<string, number> {
	return getJson<Record<string, number>>(getColumnWidthsKey(userId, pageId), {});
}

/**
 * Save column widths to localStorage
 */
export function saveColumnWidths(
	userId: string,
	pageId: number,
	widths: Record<string, number>
): void {
	setJson(getColumnWidthsKey(userId, pageId), widths);
}

/**
 * Load row numbers preference from localStorage
 */
export function loadRowNumbersPreference(userId: string, pageId: number): boolean {
	return getBoolean(getRowNumbersKey(userId, pageId), false);
}

/**
 * Save row numbers preference to localStorage
 */
export function saveRowNumbersPreference(userId: string, pageId: number, show: boolean): void {
	setBoolean(getRowNumbersKey(userId, pageId), show);
}
