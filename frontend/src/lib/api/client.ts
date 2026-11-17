import type { components, operations } from './schema';

export type User = components['schemas']['User'];
export type Profile = components['schemas']['Profile'];
export type RegisterInput = components['schemas']['Register'];
type Result<K extends keyof operations, S extends number> = operations[K] extends {
	responses: Record<S, { content: { 'application/json': infer T } }>;
}
	? T
	: never;

export class ApiError extends Error {
	constructor(
		public code: string,
		public status = 0,
		public retryAfter = 0
	) {
		super(code);
	}
}

async function request<T>(path: string, body?: unknown, signal?: AbortSignal): Promise<T> {
	const headers = new Headers({ Accept: 'application/json' });
	if (body !== undefined) {
		const csrf = await request<Result<'csrf', 200>>('/auth/csrf');
		headers.set('X-CSRF-Token', csrf.token);
		headers.set('Content-Type', 'application/json');
	}
	let response: Response;
	try {
		response = await fetch(`/api/v1${path}`, {
			method: body === undefined ? 'GET' : 'POST',
			credentials: 'same-origin',
			cache: 'no-store',
			headers,
			body: body === undefined ? undefined : JSON.stringify(body),
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
		throw new ApiError(
			payload?.error?.code ?? 'unavailable',
			response.status,
			Number(response.headers.get('Retry-After')) || 0
		);
	}
	try {
		return await response.json();
	} catch {
		throw new ApiError('unavailable');
	}
}

export const api = {
	me: () => request<Result<'me', 200>>('/me'),
	logout: () => request<Result<'logout', 200>>('/auth/logout', {}),
	register: (input: RegisterInput) => request<Result<'register', 202>>('/auth/register', input),
	login: (input: { email: string; password: string }) =>
		request<Result<'login', 200>>('/auth/login', input),
	resend: (email: string) =>
		request<Result<'resendVerification', 202>>('/auth/resend-verification', { email }),
	forgot: (email: string) =>
		request<Result<'forgotPassword', 202>>('/auth/forgot-password', { email }),
	preview: (token: string) =>
		request<Result<'previewVerification', 200>>('/auth/verify-email/preview', { token }),
	verify: (token: string) => request<Result<'verifyEmail', 200>>('/auth/verify-email', { token }),
	reset: (token: string, password: string) =>
		request<Result<'resetPassword', 200>>('/auth/reset-password', { token, password }),
	authors: (q: string, offset: number, signal?: AbortSignal) =>
		request<Result<'listAuthors', 200>>(
			`/authors?${new URLSearchParams({ q, offset: String(offset) })}`,
			undefined,
			signal
		),
	author: (nickname: string, signal?: AbortSignal) =>
		request<Result<'publicAuthor', 200>>(
			`/authors/${encodeURIComponent(nickname)}`,
			undefined,
			signal
		)
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
		invalid_token: 'Ссылка недействительна или срок её действия истёк. Запросите новое письмо.',
		password_unchanged: 'Придумайте пароль, который отличается от текущего.',
		validation: 'Проверьте заполненные поля и попробуйте снова.',
		invalid_nickname: 'Никнейм: от 3 до 30 латинских букв, цифр, знаков _ или -.',
		csrf: 'Не удалось проверить запрос. Обновите страницу и повторите попытку.',
		network: 'Нет связи с сервером. Проверьте подключение и попробуйте ещё раз.'
	};
	return messages[error.code] ?? 'Сервис временно недоступен. Попробуйте ещё раз чуть позже.';
}
