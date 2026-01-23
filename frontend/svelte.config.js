import adapter from '@sveltejs/adapter-node';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	// Consult https://kit.svelte.dev/docs/integrations#preprocessors
	// for more information about preprocessors
	preprocess: vitePreprocess(),

	// Suppress specific warnings for intentional patterns
	onwarn: (warning, handler) => {
		// Ignore a11y warnings for keyboard-navigable containers
		if (warning.code === 'a11y_no_noninteractive_tabindex') return;
		if (warning.code === 'a11y_no_noninteractive_element_interactions') return;
		// Ignore CSS warnings about Tailwind's @apply directive
		if (warning.code === 'css_unused_selector') return;
		handler(warning);
	},

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
