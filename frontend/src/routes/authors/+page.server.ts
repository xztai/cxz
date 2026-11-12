import { error } from '@sveltejs/kit';
import { serverApi } from '#lib/api/server.js';
import type { PageServerLoad } from './$types';
export const load: PageServerLoad = async ({ url }) => {
	const q = (url.searchParams.get('q') ?? '').trim();
	const rawOffset = url.searchParams.get('offset') ?? '0';
	const offset = /^\d+$/.test(rawOffset) ? Number(rawOffset) : -1;
	if (q.length > 100 || offset < 0 || offset > 100000) error(400, 'Проверьте поисковый запрос.');
	try {
		const result = await serverApi('').GET('/authors', { params: { query: { q, offset } } });
		if (!result.data)
			error(result.response.status, result.error?.error.message ?? 'Не удалось загрузить каталог.');
		return { ...result.data, q, offset };
	} catch (cause) {
		if (cause && typeof cause === 'object' && 'status' in cause) throw cause;
		error(503, 'Каталог временно недоступен. Попробуйте позже.');
	}
};
