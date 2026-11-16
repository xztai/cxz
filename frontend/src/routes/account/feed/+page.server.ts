import { error } from '@sveltejs/kit';
import { serverApi } from '#lib/api/server.js';
import type { PageServerLoad } from './$types';
export const load: PageServerLoad = async ({ request, url }) => {
	const cursor = url.searchParams.get('cursor') ?? '';
	try {
		const result = await serverApi(request.headers.get('cookie') ?? '').GET('/me/feed', {
			params: { query: { cursor } }
		});
		if (!result.data)
			error(result.response.status, result.error?.error.message ?? 'Не удалось загрузить ленту.');
		return { ...result.data, cursor };
	} catch (cause) {
		if (cause && typeof cause === 'object' && 'status' in cause) throw cause;
		error(503, 'Лента временно недоступна. Попробуйте позже.');
	}
};
