import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [tailwindcss(), sveltekit()],
	/*
	 * muse ships uncompiled source, including `.svelte.ts` rune modules. Vite's dev-only
	 * dependency optimizer hands those to esbuild without the TypeScript transform, so the
	 * first type annotation kills `vite dev` while `vite build`, which never runs the
	 * optimizer, stays green. Excluding the package leaves it to the svelte plugin.
	 */
	optimizeDeps: {
		exclude: ['@facile/muse']
	},
	server: {
		proxy: {
			'/api': 'http://localhost:4010'
		}
	}
});
