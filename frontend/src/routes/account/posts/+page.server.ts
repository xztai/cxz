import { error } from '@sveltejs/kit';
import { serverApi } from '#lib/api/server.js';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ request, url }) => {
	const offset = Number(url.searchParams.get('offset') ?? 0);
	if (!Number.isInteger(offset) || offset < 0 || offset > 100000) error(400, 'Неверная страница.');
	try {
		const api = serverApi(request.headers.get('cookie') ?? '');
		const [posts, profile] = await Promise.all([
			api.GET('/me/posts', { params: { query: { offset } } }),
			api.GET('/me/profile')
		]);
		if (!posts.data || !profile.data)
			error(503, 'Не удалось загрузить публикации. Попробуйте позже.');
		return { ...posts.data, profile: profile.data.profile, offset };
	} catch (cause) {
		if (cause && typeof cause === 'object' && 'status' in cause) throw cause;
		error(503, 'Не удалось загрузить публикации. Попробуйте позже.');
	}
};
