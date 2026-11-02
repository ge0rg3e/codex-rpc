import { drizzle } from 'drizzle-orm/postgres-js';
import postgres from 'postgres';
import * as schema from './schema';
import { DATABASE_URL } from '$app/env/private';
import { error } from '@sveltejs/kit';
const connectionURL = (() => {
	try {
		const value = new URL(DATABASE_URL || '');
		return ['postgres:', 'postgresql:'].includes(value.protocol) ? DATABASE_URL : undefined;
	} catch {
		return undefined;
	}
})();

export const databaseConfigured = Boolean(connectionURL);

export const db = drizzle(postgres(connectionURL || 'postgres://localhost/codex_rpc_unconfigured', { max: 5, connect_timeout: 3 }), { schema });

export function getDB() {
	if (!databaseConfigured) error(503, 'Set a valid DATABASE_URL and run npm run db:push to enable accounts and syncing.');
	return db;
}
