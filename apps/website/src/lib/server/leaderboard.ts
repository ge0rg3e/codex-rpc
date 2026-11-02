import { desc, eq, sql } from 'drizzle-orm';
import { getDB } from './db';
import { users, usage } from './db/schema';
import { hoursForUser, streakForUser, type Metric } from './activity-stats';

export async function leaderboard(metric: Metric = 'tokens') {
	const scores = { tokens: sql<number>`coalesce(sum(${usage.tokens}), 0)::float8`, sessions: sql<number>`count(${usage.sessionId})::int`, hours: hoursForUser, streak: streakForUser };
	return getDB()
		.select({ id: users.id, login: users.login, avatar: users.avatar, ...scores })
		.from(users)
		.leftJoin(usage, eq(users.id, usage.userId))
		.where(eq(users.public, true))
		.groupBy(users.id)
		.orderBy(desc(scores[metric]), users.login)
		.limit(100);
}
