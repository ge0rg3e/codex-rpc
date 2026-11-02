import { ORIGIN, BETTER_AUTH_SECRET, GITHUB_CLIENT_ID, GITHUB_CLIENT_SECRET, DATABASE_URL } from '$app/env/private';
import { drizzleAdapter } from 'better-auth/adapters/drizzle';
import { sveltekitCookies } from 'better-auth/svelte-kit';
import type { RequestEvent } from '@sveltejs/kit';
import { betterAuth } from 'better-auth/minimal';
import { credentials, users } from './db/schema';
import { getRequestEvent } from '$app/server';
import { account, user as authUsers } from './db/auth.schema';
import { db } from '#lib/server/db/index.ts';
import { and, eq, gt } from 'drizzle-orm';
import { createHash } from 'node:crypto';
import { getDB } from './db';

export const hash = (value: string) => createHash('sha256').update(value).digest('hex');

export async function identity(event: RequestEvent, kind = 'browser') {
	if (kind === 'browser') {
		const session = await auth.api.getSession({ headers: event.request.headers });

		if (!session) return null;

		const [github] = await getDB()
			.select()
			.from(account)
			.where(and(eq(account.userId, session.user.id), eq(account.providerId, 'github')));

		if (!github) return null;

		let login = session.user.githubLogin;
		if (!login) {
			// Older accounts (and cached sessions) may lack the mapped GitHub login.
			const [stored] = await getDB().select({ login: authUsers.githubLogin }).from(authUsers).where(eq(authUsers.id, session.user.id));
			login = stored?.login ?? undefined;
			if (!login) {
				const response = await fetch(`https://api.github.com/user/${encodeURIComponent(github.accountId)}`, {
					headers: { Accept: 'application/vnd.github+json', 'User-Agent': 'codex-rpc' },
					signal: AbortSignal.timeout(5000)
				});
				if (!response.ok) throw new Error('Unable to retrieve your GitHub profile. Please try again.');
				const profile = await response.json();
				if (String(profile.id) !== github.accountId || typeof profile.login !== 'string' || !profile.login) {
					throw new Error('Unable to verify your GitHub profile.');
				}
				login = profile.login;
				await getDB().update(authUsers).set({ githubLogin: login }).where(eq(authUsers.id, session.user.id));
			}
		}
		if (!login) throw new Error('Unable to retrieve your GitHub username.');

		const [profile] = await getDB()
			.insert(users)
			.values({ id: github.accountId, login, avatar: session.user.image ?? '' })
			.onConflictDoUpdate({ target: users.id, set: { login, avatar: session.user.image ?? '' } })
			.returning();

		return profile;
	}

	const token = kind === 'browser' ? event.cookies.get('session') : event.request.headers.get('authorization')?.replace(/^Bearer /, '');

	if (!token) return null;

	const [result] = await getDB()
		.select({ user: users })
		.from(credentials)
		.innerJoin(users, eq(credentials.userId, users.id))
		.where(and(eq(credentials.hash, hash(token)), eq(credentials.kind, kind), gt(credentials.expires, new Date())));
	return result?.user ?? null;
}

export const auth = betterAuth({
	baseURL: ORIGIN,
	secret: BETTER_AUTH_SECRET,
	database: drizzleAdapter(db, { provider: 'pg' }),
	emailAndPassword: { enabled: true },
	user: { additionalFields: { githubLogin: { type: 'string', required: false, input: false } } },
	socialProviders: {
		github: {
			clientId: GITHUB_CLIENT_ID || '',
			clientSecret: GITHUB_CLIENT_SECRET || '',
			mapProfileToUser: (profile) => ({ githubLogin: profile.login })
		}
	},
	plugins: [
		sveltekitCookies(getRequestEvent) // make sure this is the last plugin in the array
	]
});
