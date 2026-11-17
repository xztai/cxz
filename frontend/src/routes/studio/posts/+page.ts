import { api } from '#lib/api/client.js';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ parent, depends, fetch, url }) => {
	depends('studio:posts');
	const requestedOffset = Number(url.searchParams.get('offset') ?? 0);
	const offset =
		Number.isInteger(requestedOffset) && requestedOffset >= 0 && requestedOffset <= 100000
			? requestedOffset
			: 0;
	const { access } = await parent();
	if (access.state !== 'ready' || !access.profile) return { offset, listing: null };
	try {
		return { offset, listing: await api.ownPosts(offset, fetch) };
	} catch {
		return { offset, listing: null };
	}
};
