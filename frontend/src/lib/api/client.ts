import type { components, operations } from './schema';

export type User = components['schemas']['User'];
export type Profile = components['schemas']['Profile'];
export type ProfileInput = components['schemas']['ProfileInput'];
export type ProfileImageKind = operations['uploadProfileImage']['parameters']['path']['kind'];
export type Tier = components['schemas']['Tier'];
export type TierInput = components['schemas']['TierInput'];
export type Post = components['schemas']['Post'];
export type PostInput = components['schemas']['PostInput'];
export type PostSummary = components['schemas']['PostSummary'];
export type OwnPostSummary = components['schemas']['OwnPostSummary'];
export type OwnPostStatus = Required<
	NonNullable<operations['ownPosts']['parameters']['query']>
>['status'];
export type Analytics = components['schemas']['Analytics'];
export type RegisterInput = components['schemas']['Register'];
type Result<K extends keyof operations, S extends number> = operations[K] extends {
	responses: Record<S, { content: { 'application/json': infer T } }>;
}
	? T
	: never;

export type TierOrderConflictCode = Result<'reorderTiers', 409>['error']['code'];

export class ApiError extends Error {
	constructor(
		public code: string,
		public status = 0,
		public retryAfter = 0
	) {
		super(code);
	}
}

export const studioAccountFailure = 'studio-account-failure';

type RequestOptions = {
	// Studio bootstrap uses null until it has discovered the current owner.
	expectedUserId?: string | null;
	body?: unknown;
	signal?: AbortSignal;
	method?: 'GET' | 'POST' | 'PUT' | 'DELETE';
	fetcher?: typeof fetch;
};

async function request<T>(
	path: string,
	{
		body,
		expectedUserId,
		signal,
		method = body === undefined ? 'GET' : 'POST',
		fetcher = fetch
	}: RequestOptions = {}
): Promise<T> {
	const headers = new Headers({ Accept: 'application/json' });
	if (expectedUserId != null) headers.set('X-Expected-User-ID', expectedUserId);
	const rawBody = body instanceof Blob ? body : undefined;
	if (method !== 'GET') {
		const csrf = await request<Result<'csrf', 200>>('/auth/csrf', { fetcher });
		headers.set('X-CSRF-Token', csrf.token);
		if (body !== undefined) headers.set('Content-Type', rawBody?.type ?? 'application/json');
	}
	let response: Response;
	try {
		response = await fetcher(`/api/v1${path}`, {
			method,
			credentials: 'same-origin',
			cache: 'no-store',
			headers,
			body: rawBody ?? (body === undefined ? undefined : JSON.stringify(body)),
			signal: signal
				? AbortSignal.any([signal, AbortSignal.timeout(15000)])
				: AbortSignal.timeout(15000)
		});
	} catch (error) {
		if (signal?.aborted) throw error;
		throw new ApiError('network');
	}
	if (!response.ok) {
		const payload: components['schemas']['Error'] | null = await response.json().catch(() => null);
		const code = payload?.error?.code ?? 'unavailable';
		if (
			typeof window !== 'undefined' &&
			expectedUserId !== undefined &&
			(response.status === 401 || code === 'account_changed')
		) {
			window.dispatchEvent(new CustomEvent(studioAccountFailure, { detail: expectedUserId }));
		}
		throw new ApiError(code, response.status, Number(response.headers.get('Retry-After')) || 0);
	}
	try {
		return await response.json();
	} catch {
		throw new ApiError('unavailable');
	}
}

export const api = {
	me: () => request<Result<'me', 200>>('/me'),
	logout: () => request<Result<'logout', 200>>('/auth/logout', { body: {} }),
	register: (input: RegisterInput) =>
		request<Result<'register', 202>>('/auth/register', { body: input }),
	login: (input: { email: string; password: string }) =>
		request<Result<'login', 200>>('/auth/login', { body: input }),
	resend: (email: string) =>
		request<Result<'resendVerification', 202>>('/auth/resend-verification', { body: { email } }),
	changeUnverifiedEmail: (
		input: operations['changeUnverifiedEmail']['requestBody']['content']['application/json']
	) =>
		request<Result<'changeUnverifiedEmail', 202>>('/auth/change-unverified-email', { body: input }),
	forgot: (email: string) =>
		request<Result<'forgotPassword', 202>>('/auth/forgot-password', { body: { email } }),
	preview: (token: string) =>
		request<Result<'previewVerification', 200>>('/auth/verify-email/preview', { body: { token } }),
	verify: (token: string) =>
		request<Result<'verifyEmail', 200>>('/auth/verify-email', { body: { token } }),
	reset: (token: string, password: string) =>
		request<Result<'resetPassword', 200>>('/auth/reset-password', { body: { token, password } }),
	ownProfile: (fetcher?: typeof fetch) =>
		request<Result<'ownProfile', 200>>('/me/profile', { expectedUserId: null, fetcher }),
	saveProfile: (expectedUserId: string, input: ProfileInput) =>
		request<Result<'saveProfile', 200>>('/me/profile', {
			expectedUserId,
			body: input,
			method: 'PUT'
		}),
	uploadProfileImage: (expectedUserId: string, kind: ProfileImageKind, image: Blob) =>
		request<Result<'uploadProfileImage', 200>>(`/me/profile/images/${kind}`, {
			expectedUserId,
			body: image,
			method: 'PUT'
		}),
	removeProfileImage: (expectedUserId: string, kind: ProfileImageKind) =>
		request<Result<'removeProfileImage', 200>>(`/me/profile/images/${kind}`, {
			expectedUserId,
			method: 'DELETE'
		}),
	ownTiers: (expectedUserId: string, fetcher?: typeof fetch) =>
		request<Result<'ownTiers', 200>>('/me/tiers', { expectedUserId, fetcher }),
	createTier: (expectedUserId: string, input: TierInput) =>
		request<Result<'createTier', 201>>('/me/tiers', { expectedUserId, body: input }),
	updateTier: (expectedUserId: string, id: string, input: TierInput) =>
		request<Result<'updateTier', 200>>(`/me/tiers/${encodeURIComponent(id)}`, {
			expectedUserId,
			body: input,
			method: 'PUT'
		}),
	archiveTier: (expectedUserId: string, id: string) =>
		request<Result<'archiveTier', 200>>(`/me/tiers/${encodeURIComponent(id)}/archive`, {
			expectedUserId,
			method: 'POST'
		}),
	deleteTier: (expectedUserId: string, id: string) =>
		request<Result<'deleteTier', 200>>(`/me/tiers/${encodeURIComponent(id)}`, {
			expectedUserId,
			method: 'DELETE'
		}),
	reorderTiers: (
		expectedUserId: string,
		body: operations['reorderTiers']['requestBody']['content']['application/json']
	) =>
		request<Result<'reorderTiers', 200>>('/me/tiers/order', {
			expectedUserId,
			body,
			method: 'PUT'
		}),
	ownPosts: (
		expectedUserId: string,
		{
			offset = 0,
			status,
			fetcher
		}: { offset?: number; status?: OwnPostStatus; fetcher?: typeof fetch } = {}
	) => {
		const query = new URLSearchParams();
		if (offset) query.set('offset', String(offset));
		if (status) query.set('status', status);
		return request<Result<'ownPosts', 200>>(query.size ? `/me/posts?${query}` : '/me/posts', {
			expectedUserId,
			fetcher
		});
	},
	ownPost: (expectedUserId: string, id: string, fetcher?: typeof fetch) =>
		request<Result<'ownPost', 200>>(`/me/posts/${encodeURIComponent(id)}`, {
			expectedUserId,
			fetcher
		}),
	createPost: (expectedUserId: string, input: PostInput) =>
		request<Result<'createPost', 201>>('/me/posts', { expectedUserId, body: input }),
	updatePost: (expectedUserId: string, id: string, input: PostInput) =>
		request<Result<'updatePost', 200>>(`/me/posts/${encodeURIComponent(id)}`, {
			expectedUserId,
			body: input,
			method: 'PUT'
		}),
	publishPost: (expectedUserId: string, id: string) =>
		request<Result<'publishPost', 200>>(`/me/posts/${encodeURIComponent(id)}/publish`, {
			expectedUserId,
			body: {}
		}),
	unpublishPost: (expectedUserId: string, id: string) =>
		request<Result<'unpublishPost', 200>>(`/me/posts/${encodeURIComponent(id)}/unpublish`, {
			expectedUserId,
			body: {}
		}),
	deletePost: (expectedUserId: string, id: string) =>
		request<Result<'deletePost', 200>>(`/me/posts/${encodeURIComponent(id)}`, {
			expectedUserId,
			method: 'DELETE'
		}),
	analytics: (expectedUserId: string, fetcher?: typeof fetch) =>
		request<Result<'analytics', 200>>('/me/analytics', { expectedUserId, fetcher }),
	authors: (q: string, offset: number, signal?: AbortSignal) =>
		request<Result<'listAuthors', 200>>(
			`/authors?${new URLSearchParams({ q, offset: String(offset) })}`,
			{ signal }
		),
	author: (nickname: string, signal?: AbortSignal) =>
		request<Result<'publicAuthor', 200>>(`/authors/${encodeURIComponent(nickname)}`, { signal })
};

export function errorMessage(error: unknown): string {
	if (!(error instanceof ApiError)) return 'Не удалось выполнить запрос. Попробуйте ещё раз.';
	if (error.status === 429)
		return `Слишком много попыток. Повторите примерно через ${Math.max(1, Math.ceil((error.retryAfter || 900) / 60))} мин.`;
	const messages: Record<string, string> = {
		email_exists: 'Этот email уже зарегистрирован. Войдите или укажите другой адрес.',
		nickname_exists: 'Этот никнейм уже занят. Попробуйте другой.',
		credentials: 'Неверный email или пароль. Проверьте данные и попробуйте снова.',
		email_unverified: 'Сначала подтвердите email. Если письмо не пришло, запросите новое.',
		email_already_verified: 'Этот email уже подтверждён. Войдите в аккаунт.',
		invalid_token: 'Ссылка недействительна или срок её действия истёк. Запросите новое письмо.',
		password_unchanged: 'Придумайте пароль, который отличается от текущего.',
		validation: 'Проверьте заполненные поля и попробуйте снова.',
		invalid_nickname: 'Никнейм: от 3 до 30 латинских букв, цифр, знаков _ или -.',
		tier_limit: 'Можно создать не более 50 активных уровней.',
		tier_order_locked: 'Позицию используемого уровня менять нельзя.',
		csrf: 'Не удалось проверить запрос. Обновите страницу и повторите попытку.',
		network: 'Нет связи с сервером. Проверьте подключение и попробуйте ещё раз.'
	};
	return messages[error.code] ?? 'Сервис временно недоступен. Попробуйте ещё раз чуть позже.';
}
