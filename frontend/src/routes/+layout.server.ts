import { serverApi } from '#lib/api/server.js';
import type { LayoutServerLoad } from './$types';
export const load: LayoutServerLoad = async ({ request, setHeaders }) => {
	setHeaders({ 'cache-control': 'private, no-store' });
	try {
		const { data, error } = await serverApi(request.headers.get('cookie') ?? '').GET('/me');
		return { user: data?.user ?? null, apiUnavailable: !!error };
	} catch {
		return { user: null, apiUnavailable: true };
	}
};
