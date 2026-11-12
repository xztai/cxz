import { error } from '@sveltejs/kit';
import { serverApi } from '#lib/api/server.js';
import type { PageServerLoad } from './$types';
export const load: PageServerLoad = async ({ request }) => {
	try {
		const api = serverApi(request.headers.get('cookie') ?? '');
		const [tiers, profile] = await Promise.all([api.GET('/me/tiers'), api.GET('/me/profile')]);
		if (!tiers.data || !profile.data) error(503, 'Не удалось загрузить уровни. Попробуйте позже.');
		return { tiers: tiers.data.tiers, profile: profile.data.profile };
	} catch (cause) {
		if (cause && typeof cause === 'object' && 'status' in cause) throw cause;
		error(503, 'Не удалось загрузить уровни. Попробуйте позже.');
	}
};
