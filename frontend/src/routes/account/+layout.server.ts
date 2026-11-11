import { error, redirect } from '@sveltejs/kit';
import type { LayoutServerLoad } from './$types';
export const load: LayoutServerLoad = async ({ parent }) => {
	const data = await parent();
	if (data.apiUnavailable) error(503, 'Сервис временно недоступен. Попробуйте позже.');
	if (!data.user) redirect(303, '/auth/login');
	return { user: data.user };
};
