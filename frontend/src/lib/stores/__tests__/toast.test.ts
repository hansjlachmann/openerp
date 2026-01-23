import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { get } from 'svelte/store';
import { toastStore, toast } from '../toast';

// Mock crypto.randomUUID
vi.stubGlobal('crypto', {
	randomUUID: vi.fn(() => 'test-uuid-' + Math.random().toString(36).substring(7))
});

describe('toastStore', () => {
	beforeEach(() => {
		vi.useFakeTimers();
		// Clear all toasts before each test
		const state = get(toastStore);
		state.toasts.forEach(t => toastStore.removeToast(t.id));
	});

	afterEach(() => {
		vi.useRealTimers();
	});

	describe('addToast', () => {
		it('adds a toast with default values', () => {
			toastStore.addToast('Test message');
			const state = get(toastStore);

			expect(state.toasts).toHaveLength(1);
			expect(state.toasts[0].message).toBe('Test message');
			expect(state.toasts[0].type).toBe('info');
			expect(state.toasts[0].duration).toBe(4000);
		});

		it('adds a toast with custom type', () => {
			toastStore.addToast('Error message', 'error');
			const state = get(toastStore);

			expect(state.toasts[0].type).toBe('error');
		});

		it('adds a toast with custom duration', () => {
			toastStore.addToast('Custom duration', 'info', 2000);
			const state = get(toastStore);

			expect(state.toasts[0].duration).toBe(2000);
		});

		it('returns toast id', () => {
			const id = toastStore.addToast('Test');
			expect(id).toBeDefined();
			expect(typeof id).toBe('string');
		});

		it('adds multiple toasts', () => {
			toastStore.addToast('First');
			toastStore.addToast('Second');
			toastStore.addToast('Third');
			const state = get(toastStore);

			expect(state.toasts).toHaveLength(3);
		});

		it('auto-removes toast after duration', () => {
			toastStore.addToast('Auto remove', 'info', 3000);
			expect(get(toastStore).toasts).toHaveLength(1);

			vi.advanceTimersByTime(3000);
			expect(get(toastStore).toasts).toHaveLength(0);
		});

		it('does not auto-remove when duration is 0', () => {
			toastStore.addToast('Persistent', 'info', 0);
			expect(get(toastStore).toasts).toHaveLength(1);

			vi.advanceTimersByTime(10000);
			expect(get(toastStore).toasts).toHaveLength(1);
		});
	});

	describe('removeToast', () => {
		it('removes a toast by id', () => {
			const id = toastStore.addToast('To remove', 'info', 0);
			expect(get(toastStore).toasts).toHaveLength(1);

			toastStore.removeToast(id);
			expect(get(toastStore).toasts).toHaveLength(0);
		});

		it('only removes the specified toast', () => {
			toastStore.addToast('First', 'info', 0);
			const id = toastStore.addToast('Second', 'info', 0);
			toastStore.addToast('Third', 'info', 0);

			toastStore.removeToast(id);
			const state = get(toastStore);

			expect(state.toasts).toHaveLength(2);
			expect(state.toasts.find(t => t.message === 'Second')).toBeUndefined();
		});

		it('handles removing non-existent id gracefully', () => {
			toastStore.addToast('Test', 'info', 0);
			toastStore.removeToast('non-existent');
			expect(get(toastStore).toasts).toHaveLength(1);
		});
	});

	describe('convenience methods', () => {
		it('success adds a success toast', () => {
			toastStore.success('Success!');
			const state = get(toastStore);

			expect(state.toasts[0].type).toBe('success');
			expect(state.toasts[0].message).toBe('Success!');
		});

		it('error adds an error toast with longer duration', () => {
			toastStore.error('Error!');
			const state = get(toastStore);

			expect(state.toasts[0].type).toBe('error');
			expect(state.toasts[0].duration).toBe(6000);
		});

		it('warning adds a warning toast', () => {
			toastStore.warning('Warning!');
			const state = get(toastStore);

			expect(state.toasts[0].type).toBe('warning');
		});

		it('info adds an info toast', () => {
			toastStore.info('Info!');
			const state = get(toastStore);

			expect(state.toasts[0].type).toBe('info');
		});

		it('allows custom duration for convenience methods', () => {
			toastStore.success('Custom', 1000);
			expect(get(toastStore).toasts[0].duration).toBe(1000);
		});
	});
});

describe('toast shorthand', () => {
	beforeEach(() => {
		vi.useFakeTimers();
		const state = get(toastStore);
		state.toasts.forEach(t => toast.remove(t.id));
	});

	afterEach(() => {
		vi.useRealTimers();
	});

	it('toast.success works', () => {
		toast.success('Success via shorthand');
		expect(get(toastStore).toasts[0].type).toBe('success');
	});

	it('toast.error works', () => {
		toast.error('Error via shorthand');
		expect(get(toastStore).toasts[0].type).toBe('error');
	});

	it('toast.warning works', () => {
		toast.warning('Warning via shorthand');
		expect(get(toastStore).toasts[0].type).toBe('warning');
	});

	it('toast.info works', () => {
		toast.info('Info via shorthand');
		expect(get(toastStore).toasts[0].type).toBe('info');
	});

	it('toast.remove works', () => {
		const id = toast.success('To remove', 0);
		expect(get(toastStore).toasts).toHaveLength(1);

		toast.remove(id);
		expect(get(toastStore).toasts).toHaveLength(0);
	});
});
