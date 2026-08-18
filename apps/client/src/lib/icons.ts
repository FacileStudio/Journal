import { registerIcons } from '@facile/muse';

/**
 * The one glyph muse does not carry.
 *
 * muse bundles the paths for every name in its own map, so nothing it draws touches the network.
 * A name it does not carry falls back to `<iconify-icon>`, which needs a custom element this app
 * no longer registers and a request to api.iconify.design the CSP no longer allows — so it would
 * render nothing at all. This is the same artwork as `static/logo.svg`, which is where Journal's
 * mark already lived.
 *
 * Call it before the first render. Adding the key to muse's own map is the better answer for a
 * glyph the whole suite wants; regenerating that map is currently blocked upstream because
 * `solar:magnifer-linear` no longer exists in the collection.
 */
export const NOTEBOOK = 'solar:notebook-bold-duotone';

export function registerJournalIcons(): void {
	registerIcons({
		[NOTEBOOK]: {
			body: '<path fill="currentColor" opacity=".5" d="M3 8c0-2.828 0-4.243.879-5.121C4.757 2 6.172 2 9 2h6c2.828 0 4.243 0 5.121.879C21 3.757 21 5.172 21 8v8c0 2.828 0 4.243-.879 5.121C19.243 22 17.828 22 15 22H9c-2.828 0-4.243 0-5.121-.879C3 20.243 3 18.828 3 16z" /> <path fill="currentColor" fill-rule="evenodd" clip-rule="evenodd" d="M8.75 2.012v20h-1.5v-20zM1.25 8A.75.75 0 0 1 2 7.25h2a.75.75 0 0 1 0 1.5H2A.75.75 0 0 1 1.25 8m0 4a.75.75 0 0 1 .75-.75h2a.75.75 0 0 1 0 1.5H2a.75.75 0 0 1-.75-.75m0 4a.75.75 0 0 1 .75-.75h2a.75.75 0 0 1 0 1.5H2a.75.75 0 0 1-.75-.75" /> <path fill="currentColor" d="M10.75 6.5a.75.75 0 0 1 .75-.75h5a.75.75 0 0 1 0 1.5h-5a.75.75 0 0 1-.75-.75m0 3.5a.75.75 0 0 1 .75-.75h5a.75.75 0 0 1 0 1.5h-5a.75.75 0 0 1-.75-.75" />',
			width: 24,
			height: 24
		}
	});
}
