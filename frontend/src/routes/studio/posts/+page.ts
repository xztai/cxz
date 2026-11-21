import { api, type OwnPostStatus } from '#lib/api/client.js';
import type { PageLoad } from './$types';
import { maxPage, pageSize, postsDependency } from './listing.js';

export const load: PageLoad = async ({ parent, depends, fetch, url }) => {
	depends(postsDependency);
	const status: OwnPostStatus =
		url.searchParams.get('status') === 'published' ? 'published' : 'draft';
	const requestedPage = Number(
		url.searchParams.get('page') ?? Number(url.searchParams.get('offset') ?? 0) / pageSize + 1
	);
	const pageNumber =
		Number.isInteger(requestedPage) && requestedPage >= 1 && requestedPage <= maxPage
			? requestedPage
			: 1;
	const { access } = await parent();
	if (access.state !== 'ready' || !access.profile) return { status, pageNumber, listing: null };
	try {
		const offset = (pageNumber - 1) * pageSize;
		const listing = await api.ownPosts(access.ownerId, { offset, status, fetcher: fetch });
		return { status, pageNumber, listing };
	} catch {
		return { status, pageNumber, listing: null };
	}
};
