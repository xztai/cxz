import { api, ApiError } from '#lib/api/client.js';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ depends, fetch, params, parent }) => {
	depends('studio:post-editor');
	const { access } = await parent();
	if (access.state !== 'ready' || !access.profile) {
		return { post: null, tiers: null, missing: false };
	}
	try {
		const [{ post }, tiers] = await Promise.all([
			api.ownPost(params.id, fetch),
			api.ownTiers(fetch).then(
				({ tiers }) => tiers,
				() => null
			)
		]);
		return { post, tiers, missing: false };
	} catch (error) {
		return { post: null, tiers: null, missing: error instanceof ApiError && error.status === 404 };
	}
};
