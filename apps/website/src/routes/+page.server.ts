import { leaderboard } from '#lib/server/leaderboard.ts';
import { parseMetric } from '#lib/server/activity-stats.js';
import { identity } from '#lib/server/auth.js';

export const load = async (event) => {
	const agent = event.request.headers.get('user-agent') ?? '';
	const platform = /Android|iPhone|iPad|iPod|Mobile/i.test(agent) ? 'unknown'
		: /Windows/i.test(agent) ? 'Windows' : /Macintosh|Mac OS X/i.test(agent) ? 'macOS'
		: /Linux|X11/i.test(agent) ? 'Linux' : 'unknown';
	const user = await identity(event).catch((error) => {
		console.error('Unable to load the signed-in GitHub profile:', error);
		return null;
	});

	try {
		const entries = await leaderboard(parseMetric(event.url.searchParams.get('rank')));
		return { entries, user, platform, configured: true, demo: false, unavailable: false };
	} catch {
		return { entries: [], user, platform, configured: true, demo: false, unavailable: true };
	}
};
