import { writable } from 'svelte/store';
import { getJson, setJson, remove } from '$lib/utils/storage';

interface User {
	user_id: string;
	user_name: string;
	email?: string;
	language?: string;
}

const STORAGE_KEY = 'currentUser';

function createUserStore() {
	const { subscribe, set } = writable<User | null>(null);

	return {
		subscribe,
		setUser: (user: User | null) => {
			if (user) {
				setJson(STORAGE_KEY, user);
			} else {
				remove(STORAGE_KEY);
			}
			set(user);
		},
		loadFromStorage: () => {
			const user = getJson<User | null>(STORAGE_KEY, null);
			set(user);
		},
		clear: () => {
			remove(STORAGE_KEY);
			set(null);
		}
	};
}

export const currentUser = createUserStore();
