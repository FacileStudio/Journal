import type { RequestHandler } from './$types';
import { env } from '$env/dynamic/private';

const API_URL = env.API_URL || 'http://localhost:4010';

const strippedRequestHeaders = [
	'host',
	'connection',
	'keep-alive',
	'te',
	'upgrade',
	'proxy-authorization'
];

export const fallback: RequestHandler = async ({ request, params, url }) => {
	const path = params.path.split('/').map(encodeURIComponent).join('/');
	const target = `${API_URL}/${path}${url.search}`;

	const headers = new Headers(request.headers);
	for (const header of strippedRequestHeaders) headers.delete(header);

	const init: RequestInit = {
		method: request.method,
		headers,
		redirect: 'manual',
		signal: AbortSignal.timeout(10_000)
	};

	if (request.method !== 'GET' && request.method !== 'HEAD') {
		init.body = request.body;
		// @ts-expect-error — needed for streaming request bodies
		init.duplex = 'half';
	}

	let response: Response;
	try {
		response = await fetch(target, init);
	} catch {
		return new Response(JSON.stringify({ error: 'upstream unavailable' }), {
			status: 502,
			headers: { 'Content-Type': 'application/json' }
		});
	}

	const responseHeaders = new Headers(response.headers);
	responseHeaders.delete('transfer-encoding');
	responseHeaders.delete('content-encoding');
	responseHeaders.delete('content-length');

	return new Response(response.body, {
		status: response.status,
		statusText: response.statusText,
		headers: responseHeaders
	});
};
