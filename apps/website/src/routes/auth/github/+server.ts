import { redirect, error } from '@sveltejs/kit';
import { auth } from '#lib/server/auth.js';

export const GET = async ({ request, url }) => {
	const device = url.searchParams.get('device');

	const callbackURL = device && /^[A-F0-9]{8}$/.test(device) ? `${url.origin}/?device=${device}` : `${url.origin}/`;

	const result = await auth.api.signInSocial({ body: { provider: 'github', callbackURL }, headers: request.headers });

	if (!result.url) error(502, 'GitHub sign-in could not start.');

	redirect(302, result.url, { external: true });
};
