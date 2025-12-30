import { browser } from '$app/environment';
import { goto } from '$app/navigation';

export const ssr = false; // Disable server-side rendering for this app

export async function load({ url }) {
	if (browser) {
		// Check if user is authenticated
		const currentUser = localStorage.getItem('currentUser');
		const isLoginPage = url.pathname === '/login';

		// If not logged in and not on login page, redirect to login
		if (!currentUser && !isLoginPage) {
			goto('/login');
			return {};
		}

		// If logged in and on login page, redirect to home
		if (currentUser && isLoginPage) {
			goto('/');
			return {};
		}
	}

	return {};
}
