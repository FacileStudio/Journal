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
					'sha256-0T6cfH+fUhn5orKAp2IkbCejpY1EdL0lAMTgbJZblwY=',
					'sha256-2q02c/nBkodKd5cpYD9CqDcLRNS56rPl/l9WRua0vQU='
				],
				'style-src': ['self', 'unsafe-inline'],
				'connect-src': ['self', 'https://api-ardoise.facile.studio'],
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
