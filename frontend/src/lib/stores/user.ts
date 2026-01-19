import { writable } from 'svelte/store';

interface User {
	user_id: string;
	user_name: string;
	email?: string;
	language?: string;
}

function createUserStore() {
	const { subscribe, set, update } = writable<User | null>(null);

	return {
		subscribe,
		setUser: (user: User | null) => {
			if (user) {
				localStorage.setItem('currentUser', JSON.stringify(user));
			} else {
				localStorage.removeItem('currentUser');
			}
			set(user);
		},
		loadFromStorage: () => {
			const stored = localStorage.getItem('currentUser');
			if (stored) {
				try {
					const user = JSON.parse(stored);
					set(user);
				} catch (e) {
					set(null);
				}
			} else {
				set(null);
			}
		},
		clear: () => {
			localStorage.removeItem('currentUser');
			set(null);
		}
	};
}

export const currentUser = createUserStore();
