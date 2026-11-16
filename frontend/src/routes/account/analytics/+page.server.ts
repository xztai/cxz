import { error } from '@sveltejs/kit';
import { serverApi } from '#lib/api/server.js';
import type { PageServerLoad } from './$types';
export const load: PageServerLoad = async ({ request, url }) => {
	try {
		const result = await serverApi(request.headers.get('cookie') ?? '').GET('/me/analytics', {
			params: {
				query: {
					from: url.searchParams.get('from') ?? undefined,
					to: url.searchParams.get('to') ?? undefined
				}
			}
		});
		if (!result.data) {
			if (result.error?.error.code === 'profile_required') return { analytics: null };
			error(
				result.response.status,
				result.error?.error.message ?? 'Не удалось загрузить аналитику.'
			);
		}
		return { analytics: result.data };
	} catch (cause) {
		if (cause && typeof cause === 'object' && 'status' in cause) throw cause;
		error(503, 'Аналитика временно недоступна. Попробуйте позже.');
	}
};
