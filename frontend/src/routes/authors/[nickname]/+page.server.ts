import { error } from '@sveltejs/kit';
import { serverApi } from '#lib/api/server.js';
import type { PageServerLoad } from './$types';
export const load: PageServerLoad = async ({ params, url }) => {
	const offset = Number(url.searchParams.get('offset') ?? 0);
	if (!Number.isInteger(offset) || offset < 0 || offset > 100000) error(400, 'Неверная страница.');
	try {
		const result = await serverApi('').GET('/authors/{nickname}', {
			params: { path: { nickname: params.nickname } }
		});
		if (!result.data)
			error(result.response.status, result.error?.error.message ?? 'Автор не найден.');
		const posts = await serverApi('').GET('/authors/{nickname}/posts', {
			params: { path: { nickname: params.nickname }, query: { offset } }
		});
		if (!posts.data) error(503, 'Не удалось загрузить публикации. Попробуйте позже.');
		return { ...result.data, ...posts.data, offset };
	} catch (cause) {
		if (cause && typeof cause === 'object' && 'status' in cause) throw cause;
		error(503, 'Страница автора временно недоступна. Попробуйте позже.');
	}
};
