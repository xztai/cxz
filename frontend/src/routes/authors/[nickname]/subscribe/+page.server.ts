import { error, redirect } from '@sveltejs/kit';
import { serverApi } from '#lib/api/server.js';
import type { PageServerLoad } from './$types';
export const load: PageServerLoad = async ({ params, url, parent }) => {
	const { user } = await parent();
	if (!user) redirect(303, '/auth/login');
	try {
		const result = await serverApi('').GET('/authors/{nickname}', {
			params: { path: { nickname: params.nickname } }
		});
		if (!result.data) error(404, 'Автор не найден.');
		const tier = result.data.tiers.find((t) => t.id === url.searchParams.get('tier'));
		if (!tier) error(404, 'Уровень недоступен. Выберите другой на странице автора.');
		if (result.data.profile.id === user.id) error(409, 'Нельзя подписаться на себя.');
		return { profile: result.data.profile, tier };
	} catch (cause) {
		if (cause && typeof cause === 'object' && 'status' in cause) throw cause;
		error(503, 'Не удалось загрузить условия подписки. Попробуйте позже.');
	}
};
