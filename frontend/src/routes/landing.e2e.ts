import { expect, test, type Page } from '@playwright/test';

const csrf = 'a'.repeat(43);
const user = {
	id: '1',
	email: 'artist@example.com',
	nickname: 'artist',
	emailVerified: true,
	isAdmin: false
};

test('credentials cannot be submitted before hydration or with JavaScript disabled', async ({
	browser
}) => {
	const context = await browser.newContext({ javaScriptEnabled: false });
	const page = await context.newPage();
	for (const path of ['/', '/login']) {
		await page.goto(`http://127.0.0.1:4173${path}`);
		await expect(page.getByLabel('Пароль', { exact: true })).toBeDisabled();
		await expect(page.locator('form')).toHaveAttribute('method', 'POST');
		await expect(page.locator('form button[type="submit"]')).toBeDisabled();
	}
	await context.close();
});

async function mockCsrf(page: Page) {
	await page.route('**/api/v1/auth/csrf', (route) =>
		route.fulfill({
			json: { token: csrf },
			headers: { 'Set-Cookie': `csrf=${csrf}; HttpOnly; Path=/; SameSite=Lax` }
		})
	);
}
async function fillRegistration(page: Page, password = ' original password 🌿 ') {
	await page.getByLabel('Email', { exact: true }).fill('artist@example.com');
	await page.getByLabel('Никнейм', { exact: true }).fill('artist');
	await page.getByLabel('Пароль', { exact: true }).fill(password);
}

test('registration sends exact password with same-origin CSRF cookie and header; success and resend work', async ({
	page
}) => {
	await mockCsrf(page);
	await page.route('**/api/v1/auth/register', async (route) => {
		const request = route.request();
		expect(request.headers()['x-csrf-token']).toBe(csrf);
		expect(request.headers().cookie).toContain(`csrf=${csrf}`);
		expect(request.headers().origin).toBe('http://127.0.0.1:4173');
		expect(request.postDataJSON()).toEqual({
			email: user.email,
			nickname: user.nickname,
			password: ' original password 🌿 '
		});
		await route.fulfill({ status: 202, json: { message: 'accepted' } });
	});
	await page.route('**/api/v1/auth/resend-verification', (route) =>
		route.fulfill({ status: 202, json: { message: 'accepted' } })
	);
	await page.goto('/');
	await fillRegistration(page);
	await page.getByRole('button', { name: 'Показать пароль' }).click();
	await expect(page.getByLabel('Пароль', { exact: true })).toHaveAttribute('type', 'text');
	await page.getByRole('button', { name: 'Скрыть пароль' }).click();
	await page.getByRole('button', { name: 'Создать аккаунт', exact: true }).click();
	await expect(page.getByRole('heading', { name: 'Проверьте почту' })).toBeVisible();
	await expect(page.locator('.registration-success')).toBeFocused();
	await page.getByRole('button', { name: 'Отправить письмо ещё раз' }).click();
	await expect(
		page.getByText('Если адрес подходит для подтверждения', { exact: false })
	).toBeVisible();
});

test('invalid fields focus the first error and reject short Unicode passwords before requesting', async ({
	page
}) => {
	let requests = 0;
	await page.route('**/api/v1/auth/**', () => {
		requests++;
	});
	await page.goto('/');
	await page.getByRole('button', { name: 'Создать аккаунт', exact: true }).click();
	await expect(page.getByLabel('Email', { exact: true })).toBeFocused();
	await fillRegistration(page, '🌿'.repeat(11));
	await page.getByRole('button', { name: 'Создать аккаунт', exact: true }).click();
	await expect(page.getByText('Пароль должен содержать не менее 12 символов.')).toBeVisible();
	await expect(page.getByLabel('Пароль', { exact: true })).toBeFocused();
	expect(requests).toBe(0);
});

for (const [code, status, text] of [
	['email_exists', 409, 'Этот email уже зарегистрирован'],
	['nickname_exists', 409, 'Этот никнейм уже занят'],
	['rate_limit', 429, 'через 15 мин.'],
	['unavailable', 503, 'Сервис временно недоступен']
] as const) {
	test(`registration recovers from ${code} and preserves inputs`, async ({ page }) => {
		await mockCsrf(page);
		await page.route('**/api/v1/auth/register', (route) =>
			route.fulfill({
				status,
				headers: { 'Retry-After': '900' },
				json: { error: { code, message: 'ignored' } }
			})
		);
		await page.goto('/');
		await fillRegistration(page);
		await page.getByRole('button', { name: 'Создать аккаунт', exact: true }).click();
		await expect(page.getByRole('alert')).toContainText(text);
		await expect(page.getByLabel('Email', { exact: true })).toHaveValue(user.email);
		await expect(page.getByLabel('Никнейм', { exact: true })).toHaveValue(user.nickname);
		await page.route('**/api/v1/auth/register', (route) =>
			route.fulfill({ status: 202, json: { message: 'accepted' } })
		);
		await page.getByRole('button', { name: 'Создать аккаунт', exact: true }).click();
		await expect(page.getByRole('heading', { name: 'Проверьте почту' })).toBeVisible();
	});
}

test('network failure allows retry and submitting blocks duplicate requests', async ({ page }) => {
	await mockCsrf(page);
	await page.route('**/api/v1/auth/register', (route) => route.abort('failed'));
	await page.goto('/');
	await fillRegistration(page);
	await page.getByRole('button', { name: 'Создать аккаунт', exact: true }).click();
	await expect(page.getByRole('alert')).toContainText('Нет связи с сервером');
	let release: () => void = () => {};
	const hold = new Promise<void>((resolve) => {
		release = resolve;
	});
	await page.route('**/api/v1/auth/register', async (route) => {
		await hold;
		await route.fulfill({ status: 202, json: { message: 'accepted' } });
	});
	await page.getByRole('button', { name: 'Создать аккаунт', exact: true }).click();
	await expect(page.getByRole('button', { name: 'Создаём аккаунт…' })).toBeDisabled();
	release();
	await expect(page.getByRole('heading', { name: 'Проверьте почту' })).toBeVisible();
});

test('login uses CSRF and opens an authenticated success state', async ({ page }) => {
	await mockCsrf(page);
	await page.route('**/api/v1/auth/login', async (route) => {
		expect(route.request().headers()['x-csrf-token']).toBe(csrf);
		expect(route.request().postDataJSON().password).toBe('original password');
		await route.fulfill({ json: { user } });
	});
	await page.goto('/login');
	await page.getByLabel('Email', { exact: true }).fill(user.email);
	await page.getByLabel('Пароль', { exact: true }).fill('original password');
	await page.getByRole('button', { name: 'Войти', exact: true }).click();
	await expect(page.getByRole('heading', { name: 'Вы вошли в аккаунт' })).toBeVisible();
});

test('verification clears the URL fragment, previews without consuming and requires confirmation', async ({
	page
}) => {
	await mockCsrf(page);
	let verified = false;
	await page.route('**/api/v1/auth/verify-email/preview', async (route) => {
		expect(route.request().postDataJSON()).toEqual({ token: csrf });
		await route.fulfill({ json: { email: user.email } });
	});
	await page.route('**/api/v1/auth/verify-email', async (route) => {
		verified = true;
		expect(route.request().postDataJSON()).toEqual({ token: csrf });
		await route.fulfill({ json: { user } });
	});
	await page.goto(`/auth/verify#token=${csrf}`);
	await expect(page.getByText(`Подтвердите, что ${user.email} — ваш адрес.`)).toBeVisible();
	expect(page.url()).not.toContain('token');
	expect(verified).toBe(false);
	await page.getByRole('button', { name: 'Подтвердить email и войти' }).click();
	await expect(page.getByRole('heading', { name: 'Email подтверждён' })).toBeVisible();
});

test('expired verification offers a working resend destination', async ({ page }) => {
	await mockCsrf(page);
	await page.route('**/api/v1/auth/verify-email/preview', (route) =>
		route.fulfill({ status: 400, json: { error: { code: 'invalid_token', message: 'expired' } } })
	);
	await page.goto(`/auth/verify#token=${csrf}`);
	await expect(page.getByRole('alert')).toContainText('Ссылка недействительна');
	await page.getByRole('link', { name: 'Запросить новое письмо' }).click();
	await expect(page.getByRole('heading', { name: 'Новое письмо' })).toBeVisible();
});

test('discovery searches real API results and opens the public profile', async ({ page }) => {
	const profile = {
		id: '2',
		nickname: 'studio',
		displayName: 'Мастерская',
		description: 'Иллюстрации и 3D',
		avatar: null,
		cover: null
	};
	await page.route('**/api/v1/authors?**', (route) => {
		const query = new URL(route.request().url()).searchParams.get('q');
		return route.fulfill({ json: { authors: query ? [profile] : [], hasMore: false } });
	});
	await page.route('**/api/v1/authors/studio', (route) =>
		route.fulfill({ json: { profile, tiers: [] } })
	);
	await page.goto('/authors');
	await expect(page.getByRole('heading', { name: 'Здесь появятся авторы' })).toBeVisible();
	await page.getByLabel('Поиск авторов').fill('Иллюстрации');
	await page.getByRole('button', { name: 'Найти', exact: true }).click();
	await page.getByRole('link', { name: 'Мастерская', exact: false }).click();
	await expect(page.getByRole('heading', { name: 'Мастерская' })).toBeVisible();
});

test('mobile and zoom keep registration usable; closing invitation focuses the existing form', async ({
	page
}) => {
	for (const width of [320, 390, 768, 1440]) {
		await page.setViewportSize({ width, height: 900 });
		await page.goto('/');
		await expect(
			page.getByRole('heading', { name: 'Создавайте. Делитесь. Получайте поддержку.' })
		).toBeVisible();
		expect(
			await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)
		).toBe(true);
	}
	await page.getByRole('link', { name: 'Создать аккаунт', exact: true }).click();
	await expect(page.getByLabel('Email', { exact: true })).toBeFocused();
	await page.locator('summary').filter({ hasText: 'Что нужно, чтобы начать?' }).click();
	await expect(page.getByText('Создайте аккаунт: укажите email', { exact: false })).toBeVisible();
});
