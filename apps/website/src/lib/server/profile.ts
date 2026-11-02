import { eq, sql } from 'drizzle-orm';
import { getDB } from './db';
import { users, usage } from './db/schema';
import { hoursForUser, streakForUser } from './activity-stats';

export async function publicProfile(login: string) {
	if (!/^[a-z\d](?:[a-z\d-]{0,38})$/i.test(login)) return null;
	const totals = sql<number>`coalesce(sum(${usage.tokens}), 0)::float8`;
	const profiles = getDB().select({
		login: users.login,
		avatar: users.avatar,
		tokens: totals.as('tokens'),
		sessions: sql<number>`count(${usage.sessionId})::int`.as('sessions'),
		hours: hoursForUser.as('hours'),
		streak: streakForUser.as('streak'),
		rank: sql<number>`(row_number() over (order by ${totals} desc, ${users.login}))::int`.as('rank'),
		builders: sql<number>`(count(*) over ())::int`.as('builders')
	}).from(users).leftJoin(usage, eq(users.id, usage.userId))
		.where(eq(users.public, true)).groupBy(users.id).as('profiles');
	const [profile] = await getDB().select().from(profiles)
		.where(sql`lower(${profiles.login}) = lower(${login})`).limit(1);
	return profile ?? null;
}
