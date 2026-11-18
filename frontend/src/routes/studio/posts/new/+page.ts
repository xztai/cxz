import { api } from '#lib/api/client.js';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ depends, fetch, parent }) => {
	depends('studio:post-editor');
	const { access } = await parent();
	if (access.state !== 'ready' || !access.profile) return { tiers: null };
	return {
		tiers: await api.ownTiers(access.ownerId, fetch).then(
			({ tiers }) => tiers,
			() => null
		)
	};
};
