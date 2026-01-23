import { describe, it, expect, beforeEach } from 'vitest';
import { get } from 'svelte/store';
import { breadcrumb } from '../breadcrumb';

describe('breadcrumb store', () => {
	beforeEach(() => {
		// Reset to default state - setHomeLabel then clear
		breadcrumb.setHomeLabel('Home');
		breadcrumb.clear();
	});

	describe('initial state', () => {
		it('has default home label', () => {
			const state = get(breadcrumb);
			expect(state.homeLabel).toBe('Home');
		});

		it('has home item after clear', () => {
			const state = get(breadcrumb);
			expect(state.items).toHaveLength(1);
			expect(state.items[0].label).toBe('Home');
			expect(state.items[0].href).toBe('/');
		});
	});

	describe('setHomeLabel', () => {
		it('updates the home label', () => {
			breadcrumb.setHomeLabel('Dashboard');
			const state = get(breadcrumb);
			expect(state.homeLabel).toBe('Dashboard');
		});
	});

	describe('setListPage', () => {
		it('sets breadcrumb for list page', () => {
			breadcrumb.setListPage(21, 'Customers');
			const state = get(breadcrumb);

			expect(state.items).toHaveLength(2);
			expect(state.items[0].label).toBe('Home');
			expect(state.items[0].href).toBe('/');
			expect(state.items[1].label).toBe('Customers');
			expect(state.items[1].href).toBe('/pages/21');
			expect(state.items[1].pageId).toBe(21);
		});

		it('uses custom home label when provided', () => {
			breadcrumb.setListPage(21, 'Customers', 'Start');
			const state = get(breadcrumb);

			expect(state.items[0].label).toBe('Start');
			expect(state.homeLabel).toBe('Start');
		});

		it('preserves existing home label when not provided', () => {
			breadcrumb.setHomeLabel('Dashboard');
			breadcrumb.setListPage(21, 'Customers');
			const state = get(breadcrumb);

			expect(state.items[0].label).toBe('Dashboard');
		});
	});

	describe('setCardPage', () => {
		it('sets breadcrumb for card page without list', () => {
			breadcrumb.setCardPage(22, 'Customer Card', 'CUST-001', 'Acme Corp');
			const state = get(breadcrumb);

			expect(state.items).toHaveLength(2);
			expect(state.items[0].label).toBe('Home');
			expect(state.items[1].label).toBe('Customer Card (Acme Corp)');
			expect(state.items[1].href).toBe('/pages/22/CUST-001');
			expect(state.items[1].pageId).toBe(22);
			expect(state.items[1].recordId).toBe('CUST-001');
		});

		it('sets breadcrumb for card page with parent list', () => {
			breadcrumb.setCardPage(22, 'Customer Card', 'CUST-001', 'Acme Corp', 21, 'Customers');
			const state = get(breadcrumb);

			expect(state.items).toHaveLength(3);
			expect(state.items[0].label).toBe('Home');
			expect(state.items[1].label).toBe('Customers');
			expect(state.items[1].href).toBe('/pages/21');
			expect(state.items[1].pageId).toBe(21);
			expect(state.items[2].label).toBe('Customer Card (Acme Corp)');
		});

		it('handles undefined recordId', () => {
			breadcrumb.setCardPage(22, 'Customer Card', undefined, undefined);
			const state = get(breadcrumb);

			expect(state.items[1].label).toBe('Customer Card');
			expect(state.items[1].href).toBe('/pages/22');
			expect(state.items[1].recordId).toBeUndefined();
		});

		it('handles undefined recordLabel', () => {
			breadcrumb.setCardPage(22, 'Customer Card', 'CUST-001', undefined);
			const state = get(breadcrumb);

			expect(state.items[1].label).toBe('Customer Card');
			expect(state.items[1].href).toBe('/pages/22/CUST-001');
		});

		it('uses custom home label', () => {
			breadcrumb.setCardPage(22, 'Customer Card', 'CUST-001', 'Test', undefined, undefined, 'Start');
			const state = get(breadcrumb);

			expect(state.items[0].label).toBe('Start');
			expect(state.homeLabel).toBe('Start');
		});
	});

	describe('clear', () => {
		it('resets to just home item', () => {
			breadcrumb.setListPage(21, 'Customers');
			breadcrumb.clear();
			const state = get(breadcrumb);

			expect(state.items).toHaveLength(1);
			expect(state.items[0].label).toBe('Home');
		});

		it('preserves home label when clearing', () => {
			breadcrumb.setHomeLabel('Dashboard');
			breadcrumb.setListPage(21, 'Customers');
			breadcrumb.clear();
			const state = get(breadcrumb);

			expect(state.items[0].label).toBe('Dashboard');
		});
	});

	describe('setCustom', () => {
		it('sets custom breadcrumb items', () => {
			breadcrumb.setCustom([
				{ label: 'Settings', href: '/settings' },
				{ label: 'Profile', href: '/settings/profile' }
			]);
			const state = get(breadcrumb);

			expect(state.items).toHaveLength(3);
			expect(state.items[0].label).toBe('Home');
			expect(state.items[1].label).toBe('Settings');
			expect(state.items[2].label).toBe('Profile');
		});

		it('preserves home label', () => {
			breadcrumb.setHomeLabel('Dashboard');
			breadcrumb.setCustom([{ label: 'Settings', href: '/settings' }]);
			const state = get(breadcrumb);

			expect(state.items[0].label).toBe('Dashboard');
		});

		it('handles empty array', () => {
			breadcrumb.setCustom([]);
			const state = get(breadcrumb);

			expect(state.items).toHaveLength(1);
			expect(state.items[0].label).toBe('Home');
		});
	});
});
