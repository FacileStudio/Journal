import { expect, test } from 'bun:test';
import { refusesOwnSession } from './session-refusal';

/** Shapes an error the way `@facile/client`'s ApiError carries it. */
function apiError(status: number, code: string, message: string) {
	return Object.assign(new Error(message), { name: 'ApiError', status, code });
}

test('only the own-session refusal asks to sign in again', () => {
	expect(refusesOwnSession(apiError(403, 'permission_denied', "only the person's own session can confirm actions"))).toBe(true);
	expect(refusesOwnSession(apiError(412, 'failed_precondition', 'only a Journal administrator can delete an alert rule'))).toBe(false);
	expect(refusesOwnSession(apiError(403, 'insufficient_scope', 'no'))).toBe(false);
	expect(refusesOwnSession(apiError(403, 'permission_denied', 'only a Journal administrator can delete an alert rule'))).toBe(false);
	expect(refusesOwnSession(new Error('network'))).toBe(false);
	expect(refusesOwnSession(null)).toBe(false);
});
