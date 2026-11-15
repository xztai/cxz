import { error } from '@sveltejs/kit';
import { serverApi } from '#lib/api/server.js';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ request, params }) => {
	try {
		const api = serverApi(request.headers.get('cookie') ?? '');
		const [post, tiers] = await Promise.all([
			api.GET('/me/posts/{id}', { params: { path: { id: params.id } } }),
			api.GET('/me/tiers')
		]);
		if (!post.data)
			error(post.response.status, post.error?.error.message ?? 'Публикация не найдена.');
		if (!tiers.data) error(503, 'Не удалось загрузить уровни. Попробуйте позже.');
		return { post: post.data.post, tiers: tiers.data.tiers };
	} catch (cause) {
		if (cause && typeof cause === 'object' && 'status' in cause) throw cause;
		error(503, 'Не удалось загрузить публикацию. Попробуйте позже.');
	}
};
