import { defineEnvVars } from '@sveltejs/kit/env';

export const variables = defineEnvVars({
	DATABASE_URL: {
		description: 'PostgreSQL connection string.',
		schema: (value: string | undefined) => value
	},
	GITHUB_CLIENT_ID: {
		description: 'GitHub OAuth app client ID.',
		schema: (value: string | undefined) => value
	},
	GITHUB_CLIENT_SECRET: {
		description: 'GitHub OAuth app client secret.',
		schema: (value: string | undefined) => value
	},
	ORIGIN: {
		description: 'The app origin (base URL), e.g. `http://localhost:5173`.',
		schema: (value: string | undefined) => value
	},
	BETTER_AUTH_SECRET: {
		description: 'Secret used to sign tokens. For production use 32 characters generated with high entropy. See [Better Auth installation](https://www.better-auth.com/docs/installation).',
		schema: (value: string | undefined) => value
	}
});
