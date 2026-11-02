import { redirect } from '@sveltejs/kit';

export const GET = () => {
	redirect(307, 'https://raw.githubusercontent.com/ge0rg3e/codex-rpc/main/apps/client/scripts/install.sh', { external: true });
};
