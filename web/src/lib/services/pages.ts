// Page and menu API service

import type { PageDefinition, MenuDefinition, PageResponse, MenuResponse } from '$lib/types/pages';

const API_BASE = '/api';

// Cache for page definitions
const pageCache = new Map<number, PageDefinition>();
let menuCache: MenuDefinition | null = null;

/**
 * Fetch a page definition by ID
 */
export async function fetchPage(pageId: number): Promise<PageDefinition> {
	// Check cache first
	if (pageCache.has(pageId)) {
		return pageCache.get(pageId)!;
	}

	const response = await fetch(`${API_BASE}/pages/${pageId}`);
	if (!response.ok) {
		throw new Error(`Failed to fetch page ${pageId}: ${response.statusText}`);
	}

	const result: PageResponse = await response.json();
	if (!result.success) {
		throw new Error(`API error: ${result}`);
	}

	// Cache the page definition
	pageCache.set(pageId, result.data);
	return result.data;
}

/**
 * Fetch all page definitions
 */
export async function fetchAllPages(): Promise<PageDefinition[]> {
	const response = await fetch(`${API_BASE}/pages`);
	if (!response.ok) {
		throw new Error(`Failed to fetch pages: ${response.statusText}`);
	}

	const result = await response.json();
	if (!result.success) {
		throw new Error(`API error: ${result}`);
	}

	// Cache all pages
	const pages = result.data as PageDefinition[];
	pages.forEach((page) => {
		pageCache.set(page.page.id, page);
	});

	return pages;
}

/**
 * Fetch the menu definition
 */
export async function fetchMenu(): Promise<MenuDefinition> {
	// Check cache first
	if (menuCache) {
		return menuCache;
	}

	const response = await fetch(`${API_BASE}/menu`);
	if (!response.ok) {
		throw new Error(`Failed to fetch menu: ${response.statusText}`);
	}

	const result: MenuResponse = await response.json();
	if (!result.success) {
		throw new Error(`API error: ${result}`);
	}

	// Cache the menu
	menuCache = result.data;
	return result.data;
}

/**
 * Clear page cache
 */
export function clearPageCache() {
	pageCache.clear();
}

/**
 * Clear menu cache
 */
export function clearMenuCache() {
	menuCache = null;
}
