import tailwindcss from '@tailwindcss/vite';
import adapter from '@sveltejs/adapter-node';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

const __impeccableLiveDev =
	process.env.NODE_ENV === 'development' ? ['http://localhost:8400' as const] : [];

export default defineConfig({
	server: { proxy: { '/api': { target: 'http://localhost:8081', changeOrigin: false } } },
	plugins: [
		tailwindcss(),
		sveltekit({
			csp: {
				mode: 'auto',
				directives: {
					'default-src': ['self'],
					'script-src': ['self', ...__impeccableLiveDev],
					'connect-src': ['self', ...__impeccableLiveDev],
					'style-src': ['self', 'unsafe-inline'],
					'img-src': ['self', 'data:'],
					'font-src': ['self', 'data:'],
					'object-src': ['none'],
					'base-uri': ['self'],
					'frame-ancestors': ['none'],
					'form-action': ['self']
				}
			},
			compilerOptions: {
				// Force runes mode for the project, except for libraries. Can be removed in svelte 6.
				runes: ({ filename }) =>
					filename.split(/[/\\]/).includes('node_modules') ? undefined : true
			},

			adapter: adapter()
		})
	]
});
