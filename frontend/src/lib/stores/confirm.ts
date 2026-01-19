// Confirmation modal store
import { writable } from 'svelte/store';

interface ConfirmState {
	open: boolean;
	title: string;
	message: string;
	action: (() => Promise<void>) | null;
}

const initialState: ConfirmState = {
	open: false,
	title: '',
	message: '',
	action: null
};

function createConfirmStore() {
	const { subscribe, set, update } = writable<ConfirmState>(initialState);

	return {
		subscribe,

		/**
		 * Show a confirmation modal
		 * @param title - Modal title
		 * @param message - Confirmation message
		 * @param action - Async action to execute on confirm
		 */
		show(title: string, message: string, action: () => Promise<void>) {
			set({
				open: true,
				title,
				message,
				action
			});
		},

		/**
		 * Confirm and execute the action
		 */
		async confirm() {
			let actionToExecute: (() => Promise<void>) | null = null;

			update(state => {
				actionToExecute = state.action;
				return { ...state, open: false };
			});

			if (actionToExecute) {
				await actionToExecute();
			}

			set(initialState);
		},

		/**
		 * Cancel and close the modal
		 */
		cancel() {
			set(initialState);
		}
	};
}

export const confirm = createConfirmStore();
