import { error, json } from '@sveltejs/kit';
import { and, eq, gt, sql } from 'drizzle-orm';
import { randomBytes } from 'node:crypto';
import { getDB } from '#lib/server/db/index.js';
import { credentials, devices, usage, users, activityMinutes } from '#lib/server/db/schema.js';
import { parseMetric } from '#lib/server/activity-stats.js';
import { auth, identity, hash } from '#lib/server/auth.js';
import { leaderboard } from '#lib/server/leaderboard.js';
import type { RequestHandler } from './$types';

export const GET: RequestHandler = async (event) => {
	if (event.params.path === 'leaderboard') return json(await leaderboard(parseMetric(event.url.searchParams.get('rank'))));
	if (event.params.path === 'me') {
		const user = await identity(event, 'device');
		if (!user) error(401, 'Reconnect your GitHub account.');
		const rows = await leaderboard(),
			index = rows.findIndex((row) => row.id === user.id);
		return json({
			login: user.login,
			public: user.public,
			rank: index < 0 ? null : index + 1
		});
	}
	error(404);
};

export const POST: RequestHandler = async (event) => {
	const { request, params, cookies, url } = event;
	if (Number(request.headers.get('content-length') ?? 0) > 1500000) error(413);
	if (request.headers.has('cookie') && request.headers.get('origin') !== url.origin) error(403);

	const db = getDB();

	if (params.path === 'logout') {
		await auth.api.signOut({ headers: request.headers });
		return json({ ok: true });
	}

	if (params.path === 'device/start') {
		const token = randomBytes(32).toString('hex'),
			code = randomBytes(4).toString('hex').toUpperCase();
		await db.delete(devices).where(sql`${devices.expires} < now()`);
		await db.insert(devices).values({
			code,
			hash: hash(token),
			expires: new Date(Date.now() + 600000)
		});
		return json({ token, code, url: `${url.origin}/?device=${code}` });
	}

	if (params.path === 'device/poll') {
		const { token } = await request.json();
		if (typeof token !== 'string') error(400);
		const [device] = await db
			.select()
			.from(devices)
			.where(and(eq(devices.hash, hash(token)), gt(devices.expires, new Date())));
		if (!device) error(410, 'Pairing expired.');
		if (!device.userId) return json({ pending: true });
		const [claimed] = await db.delete(devices).where(eq(devices.code, device.code)).returning();
		if (!claimed) error(410);
		await db.insert(credentials).values({
			hash: hash(token),
			userId: device.userId,
			kind: 'device',
			expires: new Date(Date.now() + 90 * 86400000)
		});
		return json({ pending: false });
	}

	if (params.path === 'device/approve' || params.path === 'profile') {
		const kind = params.path === 'profile' && request.headers.has('authorization') ? 'device' : 'browser';
		const user = await identity(event, kind);
		if (!user) error(401, 'Sign in with GitHub first.');
		const body = await request.json();
		if (params.path === 'profile') {
			if (typeof body.public !== 'boolean') error(400);
			await db.update(users).set({ public: body.public }).where(eq(users.id, user.id));
		} else {
			if (typeof body.code !== 'string') error(400);
			const result = await db
				.update(devices)
				.set({ userId: user.id })
				.where(and(eq(devices.code, body.code), sql`${devices.userId} is null`, gt(devices.expires, new Date())))
				.returning();
			if (!result.length) error(410, 'Pairing expired or already used. Reconnect from the tray app.');
		}
		return json({ ok: true });
	}

	if (params.path === 'activity') {
		const user = await identity(event, 'device');
		if (!user) error(401);
		const { minutes } = await request.json();
		const now = Math.floor(Date.now() / 60000);
		if (!Array.isArray(minutes) || minutes.length > 5000 || minutes.some((minute) => !Number.isSafeInteger(minute) || minute < 26297280 || minute > now)) error(400, 'Invalid activity minutes.');
		const unique: number[] = [...new Set<number>(minutes)];
		if (unique.length) await db.insert(activityMinutes).values(unique.map((minute) => ({ userId: user.id, minute }))).onConflictDoNothing();
		return json({ ok: true });
	}

	if (params.path === 'usage') {
		const user = await identity(event, 'device');
		if (!user) error(401);
		const body = await request.json();
		if (!Array.isArray(body.sessions) || body.sessions.length > 500) error(400, 'Send at most 500 sessions per batch.');
		for (const row of body.sessions) {
			if (typeof row.id !== 'string' || !/^[\w-]{1,100}$/.test(row.id) || !Number.isSafeInteger(row.tokens) || row.tokens < 0 || row.tokens > 1e12) error(400, 'Invalid session totals.');
		}
		if (body.sessions.length)
			await db
				.insert(usage)
				.values(
					body.sessions.map((row: { id: string; tokens: number }) => ({
						userId: user.id,
						sessionId: row.id,
						tokens: row.tokens
					}))
				)
				.onConflictDoUpdate({
					target: [usage.userId, usage.sessionId],
					set: {
						tokens: sql`greatest(${usage.tokens}, excluded.tokens)`,
						updated: new Date()
					}
				});
		return json({ ok: true });
	}

	error(404);
};
