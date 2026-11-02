import { pgTable, text, bigint, timestamp, boolean, primaryKey } from 'drizzle-orm/pg-core';
export const users = pgTable('users', {
	id: text('id').primaryKey(),
	login: text('login').notNull(),
	avatar: text('avatar').notNull(),
	public: boolean('public').notNull().default(false)
});

export const credentials = pgTable('credentials', {
	hash: text('hash').primaryKey(),
	userId: text('user_id')
		.notNull()
		.references(() => users.id),
	kind: text('kind').notNull(),
	expires: timestamp('expires', { withTimezone: true }).notNull()
});

export const devices = pgTable('devices', {
	code: text('code').primaryKey(),
	hash: text('hash').notNull(),
	userId: text('user_id').references(() => users.id),
	expires: timestamp('expires', { withTimezone: true }).notNull()
});

export const usage = pgTable(
	'usage',
	{
		userId: text('user_id')
			.notNull()
			.references(() => users.id),
		sessionId: text('session_id').notNull(),
		tokens: bigint('tokens', { mode: 'number' }).notNull(),
		updated: timestamp('updated', { withTimezone: true }).notNull().defaultNow()
	},
	(t) => [primaryKey({ columns: [t.userId, t.sessionId] })]
);

export * from './auth.schema';

export const activityMinutes = pgTable('activity_minutes', {
	userId: text('user_id').notNull().references(() => users.id),
	minute: bigint('minute', { mode: 'number' }).notNull()
}, (t) => [primaryKey({ columns: [t.userId, t.minute] })]);
