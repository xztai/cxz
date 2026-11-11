import createClient from 'openapi-fetch';
import type { components, paths } from './schema.js';
import { requestWithTimeout } from './request.js';

export type User = components['schemas']['User'];
export class ApiError extends Error {
	constructor(
		public code: string,
		message: string
	) {
		super(message);
	}
}
export function checkError(error: components['schemas']['Error'] | undefined) {
	if (error) throw new ApiError(error.error.code, error.error.message);
}
export function errorMessage(error: unknown): string {
	return error instanceof ApiError
		? error.message
		: 'Не удалось связаться с сервисом. Проверьте подключение и попробуйте снова.';
}

// Stateless client: no shared session or token state during SSR.
export const api = createClient<paths>({
	baseUrl: '/api/v1',
	credentials: 'same-origin',
	fetch: requestWithTimeout
});
api.use({
	async onRequest({ request }) {
		if (request.method !== 'GET' && request.method !== 'HEAD') {
			const { data, error } = await api.GET('/auth/csrf');
			checkError(error);
			if (!data) throw new ApiError('csrf', 'Обновите страницу и попробуйте снова.');
			request.headers.set('X-CSRF-Token', data.token);
		}
		return request;
	}
});
