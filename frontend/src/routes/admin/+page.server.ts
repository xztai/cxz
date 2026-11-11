import { error, redirect } from '@sveltejs/kit';
import { serverApi } from '#lib/api/server.js';
import type { PageServerLoad } from './$types';
export const load: PageServerLoad = async ({ parent, request }) => {
	const data = await parent();
	if (data.apiUnavailable) error(503, 'Сервис временно недоступен.');
	if (!data.user) redirect(303, '/auth/login');
	const result = await serverApi(request.headers.get('cookie') ?? '').GET('/admin/status');
	if (result.response.status === 403) error(403, 'Недостаточно прав.');
	if (result.error) error(503, 'Сервис временно недоступен.');
	return {};
};
