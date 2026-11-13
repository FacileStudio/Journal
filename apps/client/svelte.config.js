import adapter from '@sveltejs/adapter-static';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	kit: {
		// The Go binary serves this build and owns routing, so every route is a
		// client-side one and falls back to index.html.
		adapter: adapter({ fallback: 'index.html', strict: false }),
		csp: {
			mode: 'auto',
			directives: {
				'script-src': [
					'self',
					'https://vision.facile.studio',
					'sha256-Xeng4whxN8ZlXgd5IUoCjxu9AGt/iFyk3tQDEbwHda4='
				],
				'style-src': ['self', 'unsafe-inline'],
				'connect-src': ['self'],
				'img-src': ['self', 'data:', 'https:'],
				'object-src': ['none'],
				'base-uri': ['self']
			}
		}
	},
	vitePlugin: {
		dynamicCompileOptions: ({ filename }) =>
			filename.includes('node_modules') ? undefined : { runes: true }
	}
};

export default config;
