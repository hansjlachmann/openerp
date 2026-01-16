import { writable } from 'svelte/store';
import { browser } from '$app/environment';
import { getString, setString } from '$lib/utils/storage';

type Theme = 'light' | 'dark';

// Get initial theme from localStorage or system preference
function getInitialTheme(): Theme {
	if (!browser) return 'light';

	const stored = getString('theme') as Theme | null;
	if (stored) return stored;

	// Check system preference
	if (window.matchMedia('(prefers-color-scheme: dark)').matches) {
		return 'dark';
	}

	return 'light';
}

function createThemeStore() {
	const { subscribe, set, update } = writable<Theme>(getInitialTheme());

	return {
		subscribe,
		toggle: () => {
			update((currentTheme) => {
				const newTheme = currentTheme === 'light' ? 'dark' : 'light';
				if (browser) {
					setString('theme', newTheme);
					document.documentElement.classList.toggle('dark', newTheme === 'dark');
				}
				return newTheme;
			});
		},
		set: (theme: Theme) => {
			set(theme);
			if (browser) {
				setString('theme', theme);
				document.documentElement.classList.toggle('dark', theme === 'dark');
			}
		}
	};
}

export const theme = createThemeStore();

// Initialize theme on page load
if (browser) {
	const currentTheme = getInitialTheme();
	document.documentElement.classList.toggle('dark', currentTheme === 'dark');
}
