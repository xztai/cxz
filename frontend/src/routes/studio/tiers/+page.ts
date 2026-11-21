import { api } from '#lib/api/client.js';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ parent, depends, fetch }) => {
	depends('studio:tiers');
	const { access } = await parent();
	if (access.state !== 'ready' || !access.profile) return { tiers: null };
	try {
		const { tiers } = await api.ownTiers(access.ownerId, fetch);
		return { tiers };
	} catch {
		return { tiers: null };
	}
};
