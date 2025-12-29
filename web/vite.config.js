import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [sveltekit()],
	server: {
		host: '0.0.0.0', // Required for Docker
		port: 5173,
		proxy: {
			// Proxy API requests to Go backend
			'/api': {
				target: process.env.VITE_API_URL || 'http://localhost:8080',
				changeOrigin: true
			},
			'/ws': {
				target: process.env.VITE_WS_URL || 'ws://localhost:8080',
				ws: true
			}
		}
	}
});
