import { error } from '@sveltejs/kit';
import { serverApi } from '#lib/api/server.js';
import type { PageServerLoad } from './$types';
export const load: PageServerLoad = async ({ request, params }) => {
	try {
		const result = await serverApi(request.headers.get('cookie') ?? '').GET('/me/payments/{id}', {
			params: { path: { id: params.id } }
		});
		if (!result.data)
			error(result.response.status, result.error?.error.message ?? 'Платёж не найден.');
		return result.data;
	} catch (cause) {
		if (cause && typeof cause === 'object' && 'status' in cause) throw cause;
		error(503, 'Не удалось загрузить результат оплаты. Попробуйте позже.');
	}
};
