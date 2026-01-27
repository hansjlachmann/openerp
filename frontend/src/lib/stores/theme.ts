import { writable } from 'svelte/store';
import { browser } from '$app/environment';
import { getString, setString } from '$lib/utils/storage';

type Theme = 'light' | 'dark';

const THEME_STORAGE_KEY = 'theme';

// Get initial theme from localStorage or default to dark
function getInitialTheme(): Theme {
	if (!browser) return 'dark';

	const stored = getString(THEME_STORAGE_KEY) as Theme | null;
	if (stored) return stored;

	// Default to dark mode for first-time users
	return 'dark';
}

function createThemeStore() {
	const { subscribe, set, update } = writable<Theme>(getInitialTheme());

	return {
		subscribe,
		toggle: () => {
			update((currentTheme) => {
				const newTheme = currentTheme === 'light' ? 'dark' : 'light';
				if (browser) {
					setString(THEME_STORAGE_KEY, newTheme);
					document.documentElement.classList.toggle('dark', newTheme === 'dark');
				}
				return newTheme;
			});
		},
		set: (theme: Theme) => {
			set(theme);
			if (browser) {
				setString(THEME_STORAGE_KEY, theme);
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
