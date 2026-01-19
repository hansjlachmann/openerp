import type { Handle } from '@sveltejs/kit';

const BACKEND_URL = process.env.BACKEND_URL || 'http://backend:8080';

export const handle: Handle = async ({ event, resolve }) => {
	// Proxy /api requests to the backend
	if (event.url.pathname.startsWith('/api')) {
		const backendUrl = `${BACKEND_URL}${event.url.pathname}${event.url.search}`;

		try {
			const response = await fetch(backendUrl, {
				method: event.request.method,
				headers: event.request.headers,
				body: event.request.method !== 'GET' && event.request.method !== 'HEAD'
					? await event.request.text()
					: undefined,
				duplex: 'half'
			} as RequestInit);

			return new Response(response.body, {
				status: response.status,
				statusText: response.statusText,
				headers: response.headers
			});
		} catch (error) {
			console.error('API proxy error:', error);
			return new Response(JSON.stringify({ success: false, error: 'Backend unavailable' }), {
				status: 503,
				headers: { 'Content-Type': 'application/json' }
			});
		}
	}

	return resolve(event);
};
