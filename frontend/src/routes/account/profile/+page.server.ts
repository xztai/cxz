import { error } from '@sveltejs/kit';
import { serverApi } from '#lib/api/server.js';
import type { PageServerLoad } from './$types';
export const load: PageServerLoad = async ({ request }) => {
	try {
		const result = await serverApi(request.headers.get('cookie') ?? '').GET('/me/profile');
		if (!result.data)
			error(result.response.status, result.error?.error.message ?? 'Не удалось загрузить профиль.');
		return { profile: result.data.profile };
	} catch (cause) {
		if (cause && typeof cause === 'object' && 'status' in cause) throw cause;
		error(503, 'Не удалось загрузить профиль. Попробуйте позже.');
	}
};
