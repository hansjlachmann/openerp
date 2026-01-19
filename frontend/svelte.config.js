import adapter from '@sveltejs/adapter-node';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	// Consult https://kit.svelte.dev/docs/integrations#preprocessors
	// for more information about preprocessors
	preprocess: vitePreprocess(),

	kit: {
		// adapter-node for Docker deployment
		adapter: adapter({
			out: 'build',
			precompress: false,
			envPrefix: 'VITE_'
		}),
		alias: {
			$components: 'src/lib/components',
			$stores: 'src/lib/stores',
			$services: 'src/lib/services',
			$types: 'src/lib/types',
			$utils: 'src/lib/utils'
		}
	}
};

export default config;
