import { redirect } from '@sveltejs/kit';

/* API keys moved under Settings when the dashboard adopted the suite settings standard.
   Keeping the old path alive costs one file and saves every bookmark. */
export function load() {
	redirect(307, '/settings/api');
}
