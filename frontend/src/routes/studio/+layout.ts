import { api, ApiError, type Profile } from '#lib/api/client.js';
import type { LayoutLoad } from './$types';

export const ssr = false;

type Access =
	{ state: 'ready'; profile: Profile | null } | { state: 'unauthorized' } | { state: 'error' };

export const load: LayoutLoad = async ({ depends, fetch }): Promise<{ access: Access }> => {
	depends('studio:profile');
	try {
		const { profile } = await api.ownProfile(fetch);
		return { access: { state: 'ready', profile } };
	} catch (error) {
		return {
			access: {
				state: error instanceof ApiError && error.status === 401 ? 'unauthorized' : 'error'
			}
		};
	}
};
