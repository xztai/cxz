import { error } from '@sveltejs/kit';
import { serverApi } from '#lib/api/server.js';
import type { PageServerLoad } from './$types';
export const load: PageServerLoad = async ({ params }) => {
	try {
		const result = await serverApi('').GET('/authors/{nickname}', {
			params: { path: { nickname: params.nickname } }
		});
		if (!result.data)
			error(result.response.status, result.error?.error.message ?? 'Автор не найден.');
		return result.data;
	} catch (cause) {
		if (cause && typeof cause === 'object' && 'status' in cause) throw cause;
		error(503, 'Страница автора временно недоступна. Попробуйте позже.');
	}
};
