import { api } from '#lib/api/client.js';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ parent, depends, fetch }) => {
	depends('studio:analytics');
	const { access } = await parent();
	return {
		analytics:
			access.state === 'ready' && access.profile
				? api.analytics(access.ownerId, fetch).catch(() => null)
				: null
	};
};
