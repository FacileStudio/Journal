/**
 * Reports whether err, an `ApiError` from `@facile/client`, is porte's refusal
 * of a session no browser sign-in opened (a 403 `permission_denied`), which
 * signing in again fixes. Any other refusal, such as a lost administrator
 * right, keeps its own message.
 */
export function refusesOwnSession(err: unknown): boolean {
	const refusal = err as { status?: unknown; code?: unknown } | null;
	return refusal?.status === 403 && refusal.code === 'permission_denied';
}
