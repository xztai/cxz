import { api, ApiError, type Profile } from '#lib/api/client.js';
import type { LayoutLoad } from './$types';

export const ssr = false;

type Access =
	| { state: 'ready'; ownerId: string; profile: Profile | null }
	| { state: 'unauthorized' }
	| { state: 'error' };

export const load: LayoutLoad = async ({ depends, fetch }): Promise<{ access: Access }> => {
	depends('studio:profile');
	try {
		const { ownerId, profile } = await api.ownProfile(fetch);
		if (
			typeof ownerId !== 'string' ||
			!/^[1-9][0-9]*$/.test(ownerId) ||
			(profile && profile.id !== ownerId)
		) {
			return { access: { state: 'error' } };
		}
		return { access: { state: 'ready', ownerId, profile } };
	} catch (error) {
		return {
			access: {
				state: error instanceof ApiError && error.status === 401 ? 'unauthorized' : 'error'
			}
		};
	}
};
