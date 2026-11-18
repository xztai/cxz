import { api } from '#lib/api/client.js';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ parent, depends, fetch }) => {
	depends('studio:overview');
	const { access } = await parent();
	if (access.state !== 'ready' || !access.profile) {
		return { latestPosts: null, progress: null, analytics: null };
	}

	const posts = api.ownPosts(access.ownerId, 0, fetch);
	const latestPosts = posts.then(({ posts }) => posts.slice(0, 3)).catch(() => null);
	const progress = Promise.all([api.ownTiers(access.ownerId, fetch), posts])
		.then(([{ tiers }, { posts, hasPublishedPost }]) => ({
			activeTier: tiers.some((tier) => !tier.archived),
			published: hasPublishedPost,
			draftId: hasPublishedPost ? undefined : posts[0]?.id
		}))
		.catch(() => null);
	const analytics = api.analytics(access.ownerId, fetch).catch(() => null);

	return { latestPosts, progress, analytics };
};
