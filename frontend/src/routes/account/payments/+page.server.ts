import { error } from '@sveltejs/kit';
import { serverApi } from '#lib/api/server.js';
import type { PageServerLoad } from './$types';
export const load: PageServerLoad = async ({ request, url }) => {
	const offset = Number(url.searchParams.get('offset') ?? 0);
	if (!Number.isInteger(offset) || offset < 0 || offset > 100000) error(400, 'Неверная страница.');
	try {
		const result = await serverApi(request.headers.get('cookie') ?? '').GET('/me/payments', {
			params: { query: { offset } }
		});
		if (!result.data)
			error(result.response.status, result.error?.error.message ?? 'Не удалось загрузить платежи.');
		return { ...result.data, offset };
	} catch (cause) {
		if (cause && typeof cause === 'object' && 'status' in cause) throw cause;
		error(503, 'Платежи временно недоступны. Попробуйте позже.');
	}
};
