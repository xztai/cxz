import { api } from '#lib/api/client.js';
import type { PageLoad } from './$types';
import { analyticsPeriod } from './period.js';

export const load: PageLoad = async ({ parent, depends, fetch, url }) => {
	depends('studio:analytics');
	const { period, preset, invalidPeriod } = analyticsPeriod(url.searchParams);
	const { access } = await parent();
	return {
		period,
		preset,
		invalidPeriod,
		analytics:
			access.state === 'ready' && access.profile
				? api.analytics(access.ownerId, fetch, period).catch(() => null)
				: null
	};
};
