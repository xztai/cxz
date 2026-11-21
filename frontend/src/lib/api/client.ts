import type { components, operations } from './schema';

export type User = components['schemas']['User'];
export type Profile = components['schemas']['Profile'];
export type ProfileInput = components['schemas']['ProfileInput'];
export type ProfileImageKind = operations['uploadProfileImage']['parameters']['path']['kind'];
export type Tier = components['schemas']['Tier'];
export type TierInput = components['schemas']['TierInput'];
export type Post = components['schemas']['Post'];
export type PostInput = components['schemas']['PostInput'];
export type PostFile = components['schemas']['PostFile'];
export type PostSummary = components['schemas']['PostSummary'];
export type PostListItem = components['schemas']['PostListItem'];
export type OwnPostSummary = components['schemas']['OwnPostSummary'];
export type Comment = components['schemas']['Comment'];
export type InteractionSummary = components['schemas']['InteractionSummary'];
export type Subscription = components['schemas']['Subscription'];
export type Payment = components['schemas']['Payment'];
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
	onUploadProgress?: (fraction: number) => void;
};

function upload(
	url: string,
	headers: Headers,
	file: Blob,
	onProgress: (fraction: number) => void,
	signal?: AbortSignal
) {
	return new Promise<Response>((resolve, reject) => {
		signal?.throwIfAborted();
		const xhr = new XMLHttpRequest();
		const abort = () => xhr.abort();
		xhr.open('POST', url);
		xhr.timeout = 10 * 60 * 1000;
		headers.forEach((value, name) => xhr.setRequestHeader(name, value));
		xhr.upload.onprogress = ({ loaded, total, lengthComputable }) => {
			if (lengthComputable && total > 0) onProgress(loaded / total);
		};
		xhr.upload.onload = () => onProgress(1);
		xhr.onload = () => {
			const retryAfter = xhr.getResponseHeader('Retry-After');
			resolve(
				new Response(xhr.responseText || null, {
					status: xhr.status,
					headers: retryAfter ? { 'Retry-After': retryAfter } : undefined
				})
			);
		};
		xhr.onerror = () => reject(new ApiError('network'));
		xhr.ontimeout = xhr.onerror;
		xhr.onabort = () => reject(signal?.reason);
		xhr.onloadend = () => signal?.removeEventListener('abort', abort);
		signal?.addEventListener('abort', abort, { once: true });
		xhr.send(file);
	});
}

async function request<T>(
	path: string,
	{
		body,
		expectedUserId,
		signal,
		method = body === undefined ? 'GET' : 'POST',
		fetcher = fetch,
		onUploadProgress
	}: RequestOptions = {}
): Promise<T> {
	const headers = new Headers({ Accept: 'application/json' });
	if (expectedUserId != null) headers.set('X-Expected-User-ID', expectedUserId);
	const rawBody = body instanceof Blob ? body : undefined;
	if (method !== 'GET') {
		const csrf = await request<Result<'csrf', 200>>('/auth/csrf', { fetcher });
		headers.set('X-CSRF-Token', csrf.token);
		if (body !== undefined) {
			headers.set(
				'Content-Type',
				onUploadProgress ? 'application/octet-stream' : (rawBody?.type ?? 'application/json')
			);
		}
	}
	let response: Response;
	try {
		const url = `/api/v1${path}`;
		response =
			rawBody && onUploadProgress
				? await upload(url, headers, rawBody, onUploadProgress, signal)
				: await fetcher(url, {
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
	uploadPostMedia: (
		expectedUserId: string,
		id: string,
		kind: PostFile['kind'],
		file: File,
		onUploadProgress: (fraction: number) => void,
		signal: AbortSignal
	) =>
		request<Result<'uploadPostMedia', 201>>(
			`/me/posts/${encodeURIComponent(id)}/media?${new URLSearchParams({ kind, filename: file.name })}`,
			{ expectedUserId, body: file, onUploadProgress, signal }
		),
	detachPostMedia: (expectedUserId: string, id: string, mediaId: string) =>
		request<Result<'detachPostMedia', 200>>(
			`/me/posts/${encodeURIComponent(id)}/media/${encodeURIComponent(mediaId)}`,
			{ expectedUserId, method: 'DELETE' }
		),
	deletePost: (expectedUserId: string, id: string) =>
		request<Result<'deletePost', 200>>(`/me/posts/${encodeURIComponent(id)}`, {
			expectedUserId,
			method: 'DELETE'
		}),
	analytics: (
		expectedUserId: string,
		fetcher?: typeof fetch,
		period?: operations['analytics']['parameters']['query']
	) =>
		request<Result<'analytics', 200>>(
			period ? `/me/analytics?${new URLSearchParams(period)}` : '/me/analytics',
			{ expectedUserId, fetcher }
		),
	authors: (q: string, offset: number, signal?: AbortSignal) =>
		request<Result<'listAuthors', 200>>(
			`/authors?${new URLSearchParams({ q, offset: String(offset) })}`,
			{ signal }
		),
	author: (nickname: string, signal?: AbortSignal) =>
		request<Result<'publicAuthor', 200>>(`/authors/${encodeURIComponent(nickname)}`, { signal }),
	authorPosts: (nickname: string, offset: number, signal?: AbortSignal) =>
		request<Result<'authorPosts', 200>>(
			`/authors/${encodeURIComponent(nickname)}/posts?${new URLSearchParams({ offset: String(offset) })}`,
			{ signal }
		),
	post: (id: string, signal?: AbortSignal) =>
		request<Result<'getPost', 200>>(`/posts/${encodeURIComponent(id)}`, { signal }),
	discussion: (id: string, signal: AbortSignal, before?: string) =>
		request<Result<'discussion', 200>>(
			`/posts/${encodeURIComponent(id)}/discussion${before ? `?${new URLSearchParams({ before })}` : ''}`,
			{ signal }
		),
	setPostLike: (id: string, liked: boolean, signal: AbortSignal) =>
		request<Result<'likePost', 200> | Result<'unlikePost', 200>>(
			`/posts/${encodeURIComponent(id)}/like`,
			{ method: liked ? 'PUT' : 'DELETE', signal }
		),
	createComment: (id: string, body: string, signal: AbortSignal) =>
		request<Result<'createComment', 201>>(`/posts/${encodeURIComponent(id)}/comments`, {
			body: { body },
			signal
		}),
	deleteComment: (id: string, commentId: string, signal: AbortSignal) =>
		request<Result<'deleteComment', 200>>(
			`/posts/${encodeURIComponent(id)}/comments/${encodeURIComponent(commentId)}`,
			{ method: 'DELETE', signal }
		),
	subscriptions: (offset: number, signal?: AbortSignal) =>
		request<Result<'subscriptions', 200>>(
			`/me/subscriptions?${new URLSearchParams({ offset: String(offset) })}`,
			{ signal }
		),
	checkout: (body: operations['checkout']['requestBody']['content']['application/json']) =>
		request<Result<'checkout', 202>>('/billing/checkout', { body }),
	payment: (id: string, signal?: AbortSignal) =>
		request<Result<'payment', 200>>(`/me/payments/${encodeURIComponent(id)}`, { signal })
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
		payments_disabled: 'Оплата пока недоступна.',
		price_changed: 'Цена изменилась. Обновите страницу, чтобы увидеть новую сумму.',
		tier_unavailable: 'Уровень больше недоступен. Обновите страницу.',
		checkout_terms_conflict: 'У вас есть незавершённый платёж с другими условиями.',
		subscription_active: 'У вас уже есть действующая подписка на этого автора.',
		invalid_price: 'Минимальная сумма оплаты — 1 ₽.',
		own_subscription: 'Вы смотрите свой профиль.',
		file_too_large: 'Размер файла не должен превышать 100 МБ.',
		invalid_file: 'Проверьте формат файла или загрузите его с типом «Вложение».',
		storage_unavailable: 'Хранилище временно недоступно. Попробуйте позже.',
		csrf: 'Не удалось проверить запрос. Обновите страницу и повторите попытку.',
		network: 'Нет связи с сервером. Проверьте подключение и попробуйте ещё раз.'
	};
	return messages[error.code] ?? 'Сервис временно недоступен. Попробуйте ещё раз чуть позже.';
}
