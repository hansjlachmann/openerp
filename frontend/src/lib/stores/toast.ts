import { writable } from 'svelte/store';

export type ToastType = 'success' | 'error' | 'warning' | 'info';

export interface Toast {
	id: string;
	message: string;
	type: ToastType;
	duration: number;
}

interface ToastStore {
	toasts: Toast[];
}

// Generate unique ID - fallback for non-secure contexts (HTTP without localhost)
function generateId(): string {
	if (typeof crypto !== 'undefined' && crypto.randomUUID) {
		return crypto.randomUUID();
	}
	// Fallback for non-secure contexts
	return 'xxxx-xxxx-xxxx-xxxx'.replace(/x/g, () =>
		Math.floor(Math.random() * 16).toString(16)
	);
}

function createToastStore() {
	const { subscribe, update } = writable<ToastStore>({ toasts: [] });

	function addToast(message: string, type: ToastType = 'info', duration: number = 4000) {
		const id = generateId();
		const toast: Toast = { id, message, type, duration };

		update(state => ({
			toasts: [...state.toasts, toast]
		}));

		// Auto-remove after duration
		if (duration > 0) {
			setTimeout(() => {
				removeToast(id);
			}, duration);
		}

		return id;
	}

	function removeToast(id: string) {
		update(state => ({
			toasts: state.toasts.filter(t => t.id !== id)
		}));
	}

	return {
		subscribe,
		addToast,
		removeToast,
		// Convenience methods
		success: (message: string, duration?: number) => addToast(message, 'success', duration),
		error: (message: string, duration?: number) => addToast(message, 'error', duration ?? 6000),
		warning: (message: string, duration?: number) => addToast(message, 'warning', duration),
		info: (message: string, duration?: number) => addToast(message, 'info', duration)
	};
}

export const toastStore = createToastStore();

// Export convenience functions for direct import
export const toast = {
	success: toastStore.success,
	error: toastStore.error,
	warning: toastStore.warning,
	info: toastStore.info,
	remove: toastStore.removeToast
};
