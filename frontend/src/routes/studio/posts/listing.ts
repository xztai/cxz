import { resolve } from '$app/paths';
import type { OwnPostStatus } from '#lib/api/client.js';

export const postsDependency = 'studio:posts';
export const pageSize = 24;
export const maxPage = Math.floor(100000 / pageSize) + 1;

export function postsHref(status: OwnPostStatus, pageNumber: number) {
	return resolve(`/studio/posts?status=${status}&page=${pageNumber}`);
}
