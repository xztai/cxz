import { api } from '#lib/api/client.js';
import type { PageLoad } from './$types';

async function hasPublishedPost(
	ownerId: string,
	firstPage: Awaited<ReturnType<typeof api.ownPosts>>,
	fetcher: typeof fetch
) {
	let page = firstPage;
	let offset = 0;
	while (!page.posts.some((post) => post.publishedAt !== null)) {
		if (!page.hasMore || page.posts.length === 0) return false;
		offset += page.posts.length;
		// The API caps the maximum offset at 100,000.
		if (offset > 100_000) throw new Error('Post offset limit reached');
		page = await api.ownPosts(ownerId, offset, fetcher);
	}
	return true;
}

export const load: PageLoad = async ({ parent, depends, fetch }) => {
	depends('studio:overview');
	const { access } = await parent();
	if (access.state !== 'ready' || !access.profile) {
		return { latestPosts: null, progress: null, analytics: null };
	}

	const posts = api.ownPosts(access.ownerId, 0, fetch);
	const latestPosts = posts.then(({ posts }) => posts.slice(0, 3)).catch(() => null);
	const progress = Promise.all([
		api.ownTiers(access.ownerId, fetch),
		posts,
		posts.then((firstPage) => hasPublishedPost(access.ownerId, firstPage, fetch))
	])
		.then(([{ tiers }, { posts }, published]) => ({
			activeTier: tiers.some((tier) => !tier.archived),
			published,
			draftId: published ? undefined : posts[0]?.id
		}))
		.catch(() => null);
	const analytics = api.analytics(access.ownerId, fetch).catch(() => null);

	return { latestPosts, progress, analytics };
};
