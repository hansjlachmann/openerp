import { describe, it, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';
import { confirm } from '../confirm';

describe('confirm store', () => {
	beforeEach(() => {
		confirm.cancel(); // Reset state before each test
	});

	describe('initial state', () => {
		it('starts with modal closed', () => {
			const state = get(confirm);
			expect(state.open).toBe(false);
			expect(state.title).toBe('');
			expect(state.message).toBe('');
			expect(state.action).toBeNull();
		});
	});

	describe('show', () => {
		it('opens the modal with provided values', () => {
			const mockAction = vi.fn().mockResolvedValue(undefined);
			confirm.show('Delete Item', 'Are you sure?', mockAction);

			const state = get(confirm);
			expect(state.open).toBe(true);
			expect(state.title).toBe('Delete Item');
			expect(state.message).toBe('Are you sure?');
			expect(state.action).toBe(mockAction);
		});

		it('can be called multiple times', () => {
			const action1 = vi.fn().mockResolvedValue(undefined);
			const action2 = vi.fn().mockResolvedValue(undefined);

			confirm.show('First', 'First message', action1);
			confirm.show('Second', 'Second message', action2);

			const state = get(confirm);
			expect(state.title).toBe('Second');
			expect(state.action).toBe(action2);
		});
	});

	describe('confirm', () => {
		it('executes the action when confirmed', async () => {
			const mockAction = vi.fn().mockResolvedValue(undefined);
			confirm.show('Title', 'Message', mockAction);

			await confirm.confirm();

			expect(mockAction).toHaveBeenCalledTimes(1);
		});

		it('closes the modal after confirm', async () => {
			const mockAction = vi.fn().mockResolvedValue(undefined);
			confirm.show('Title', 'Message', mockAction);

			await confirm.confirm();

			const state = get(confirm);
			expect(state.open).toBe(false);
		});

		it('resets state after confirm', async () => {
			const mockAction = vi.fn().mockResolvedValue(undefined);
			confirm.show('Title', 'Message', mockAction);

			await confirm.confirm();

			const state = get(confirm);
			expect(state.title).toBe('');
			expect(state.message).toBe('');
			expect(state.action).toBeNull();
		});

		it('handles action errors gracefully', async () => {
			const mockAction = vi.fn().mockRejectedValue(new Error('Action failed'));
			confirm.show('Title', 'Message', mockAction);

			await expect(confirm.confirm()).rejects.toThrow('Action failed');
		});

		it('does nothing when no action is set', async () => {
			// Directly test confirm without show
			await confirm.confirm();
			const state = get(confirm);
			expect(state.open).toBe(false);
		});
	});

	describe('cancel', () => {
		it('closes the modal', () => {
			const mockAction = vi.fn().mockResolvedValue(undefined);
			confirm.show('Title', 'Message', mockAction);

			confirm.cancel();

			const state = get(confirm);
			expect(state.open).toBe(false);
		});

		it('does not execute the action', () => {
			const mockAction = vi.fn().mockResolvedValue(undefined);
			confirm.show('Title', 'Message', mockAction);

			confirm.cancel();

			expect(mockAction).not.toHaveBeenCalled();
		});

		it('resets all state', () => {
			const mockAction = vi.fn().mockResolvedValue(undefined);
			confirm.show('Title', 'Message', mockAction);

			confirm.cancel();

			const state = get(confirm);
			expect(state.title).toBe('');
			expect(state.message).toBe('');
			expect(state.action).toBeNull();
		});
	});
});
