import createClient from 'openapi-fetch';
import { env } from 'node:process';
import type { paths } from './schema.js';

export function serverApi(cookie: string) {
	return createClient<paths>({
		baseUrl: `${env.API_INTERNAL_URL ?? 'http://localhost:8081'}/api/v1`,
		headers: { Cookie: cookie },
		signal: AbortSignal.timeout(5000)
	});
}
