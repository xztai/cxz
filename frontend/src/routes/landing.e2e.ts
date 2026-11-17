import { expect, test, type Page } from '@playwright/test';

const csrf = 'a'.repeat(43);
const user = {
	id: '1',
	email: 'artist@example.com',
	nickname: 'artist',
	emailVerified: true,
	isAdmin: false
};

test.beforeEach(async ({ page }) => {
	await page.route('**/api/v1/me', (route) => route.fulfill({ json: { user: null } }));
});

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
	await page.clock.install();
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
	await expect(page).toHaveURL('/auth/check-email');
	await expect(page.getByRole('heading', { name: 'Проверьте почту' })).toBeVisible();
	await expect(page.getByText(user.email, { exact: true })).toBeVisible();
	const resend = page.getByRole('button', { name: 'Отправить письмо ещё раз' });
	await expect(resend).toBeDisabled();
	await expect(page.getByText(/Повторная отправка через \d+ сек\./)).toBeVisible();
	await page.reload();
	await expect(resend).toBeDisabled();
	await page.clock.fastForward(61_000);
	await expect(resend).toBeEnabled();
	await resend.click();
	await expect(page.getByRole('status')).toContainText('Письмо отправлено повторно');
	await expect(resend).toBeDisabled();
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
	await page.route('**/api/v1/auth/logout', (route) => route.fulfill({ json: { message: 'ok' } }));
	await page.route('**/api/v1/auth/login', async (route) => {
		expect(route.request().headers()['x-csrf-token']).toBe(csrf);
		expect(route.request().postDataJSON().password).toBe('original password');
		await route.fulfill({ json: { user } });
	});
	await page.goto('/login');
	await expect(page.getByRole('link', { name: 'Подтвердить email', exact: true })).toHaveCount(0);
	await page.getByLabel('Email', { exact: true }).fill(user.email);
	await page.getByLabel('Пароль', { exact: true }).fill('original password');
	await page.getByRole('button', { name: 'Войти', exact: true }).click();
	await expect(page.getByRole('heading', { name: 'Вы вошли в аккаунт' })).toBeVisible();
	await expect(page.locator('header')).toContainText('@artist');
	await expect(
		page.locator('header').getByRole('link', { name: 'Войти', exact: true })
	).toHaveCount(0);
	await expect(page.locator('main form')).toHaveCount(0);
	await page.getByRole('button', { name: 'Выйти', exact: true }).click();
	await expect(page.getByLabel('Email', { exact: true })).toBeEnabled();
	await expect(page.getByRole('link', { name: 'Забыли пароль?' })).toBeVisible();
});

test('restores the session on refresh and retries a failed logout without losing it', async ({
	page
}) => {
	let signedIn = true;
	let failLogout = true;
	await mockCsrf(page);
	await page.route('**/api/v1/me', (route) =>
		route.fulfill({ json: { user: signedIn ? user : null } })
	);
	await page.route('**/api/v1/auth/logout', (route) => {
		expect(route.request().method()).toBe('POST');
		expect(route.request().headers()['x-csrf-token']).toBe(csrf);
		if (failLogout) {
			failLogout = false;
			return route.fulfill({ status: 503, json: { error: { code: 'unavailable' } } });
		}
		signedIn = false;
		return route.fulfill({ json: { message: 'ok' } });
	});
	await page.goto('/login');
	await expect(page.getByRole('heading', { name: 'Вы вошли в аккаунт' })).toBeVisible();
	await expect(page.locator('main form')).toHaveCount(0);
	await page.reload();
	await expect(page.locator('header')).toContainText('@artist');
	await page.getByRole('button', { name: 'Выйти', exact: true }).click();
	await expect(page.locator('header').getByRole('alert')).toContainText(
		'Сервис временно недоступен'
	);
	await expect(page.locator('header')).toContainText('@artist');
	await page.getByRole('button', { name: 'Выйти', exact: true }).click();
	await expect(
		page.locator('header').getByRole('link', { name: 'Войти', exact: true })
	).toBeVisible();
	await expect(page.getByRole('heading', { name: 'С возвращением' })).toBeVisible();
	await expect(page.getByLabel('Email', { exact: true })).toBeEnabled();
});

test('a stale session response cannot overwrite a successful login', async ({ page }) => {
	let release = () => {};
	const hold = new Promise<void>((resolve) => {
		release = resolve;
	});
	await page.route('**/api/v1/me', async (route) => {
		await hold;
		await route.fulfill({ json: { user: null } });
	});
	await mockCsrf(page);
	await page.route('**/api/v1/auth/login', (route) => route.fulfill({ json: { user } }));
	await page.goto('/login');
	await page.getByLabel('Email', { exact: true }).fill(user.email);
	await page.getByLabel('Пароль', { exact: true }).fill('original password');
	await page.getByRole('button', { name: 'Войти', exact: true }).click();
	await expect(page.locator('header')).toContainText('@artist');
	const restored = page.waitForResponse('**/api/v1/me');
	release();
	await restored;
	await expect(page.getByRole('heading', { name: 'Вы вошли в аккаунт' })).toBeVisible();
	await expect(page.locator('header')).toContainText('@artist');
});

test('verification clears the URL fragment, previews without consuming and requires confirmation', async ({
	page
}) => {
	await mockCsrf(page);
	let verified = false;
	await page.route('**/api/v1/me', (route) =>
		route.fulfill({ json: { user: verified ? user : null } })
	);
	await page.route('**/api/v1/authors?**', (route) =>
		route.fulfill({ json: { authors: [], hasMore: false } })
	);
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
	await expect(page.locator('header')).toContainText('@artist');
	await page.getByRole('link', { name: 'Найти авторов', exact: true }).last().click();
	await expect(page.locator('header')).toContainText('@artist');
	await page.reload();
	await expect(page.locator('header')).toContainText('@artist');
});

test('expired verification offers a working resend destination', async ({ page }) => {
	await mockCsrf(page);
	await page.route('**/api/v1/auth/verify-email/preview', (route) =>
		route.fulfill({ status: 400, json: { error: { code: 'invalid_token', message: 'expired' } } })
	);
	await page.goto(`/auth/verify#token=${csrf}`);
	await expect(page.getByRole('alert')).toContainText('Ссылка недействительна');
	await page.getByRole('link', { name: 'Запросить новое письмо' }).click();
	await expect(page).toHaveURL('/auth/check-email');
	await expect(page.getByRole('heading', { name: 'Проверьте почту' })).toBeVisible();
	await expect(page.locator('main form')).toHaveCount(0);
	await page.getByRole('link', { name: 'Перейти ко входу' }).click();
	await expect(page.getByRole('heading', { name: 'С возвращением' })).toBeVisible();
});

async function unverifiedLogin(page: Page) {
	await mockCsrf(page);
	await page.route('**/api/v1/auth/login', (route) =>
		route.fulfill({ status: 403, json: { error: { code: 'email_unverified' } } })
	);
	await page.goto('/login');
	await page.getByLabel('Email', { exact: true }).fill(user.email);
	await page.getByLabel('Пароль', { exact: true }).fill('original password');
	await page.getByRole('button', { name: 'Войти', exact: true }).click();
	await expect(page).toHaveURL('/auth/check-email');
}

test('unverified login opens the same check-email screen without signing in or sending automatically', async ({
	page
}) => {
	await page.clock.install();
	let sends = 0;
	await page.route('**/api/v1/auth/resend-verification', (route) => {
		sends++;
		expect(route.request().postDataJSON()).toEqual({ email: user.email });
		return route.fulfill({ status: 202, json: { message: 'accepted' } });
	});
	await unverifiedLogin(page);
	await expect(page.getByRole('heading', { name: 'Проверьте почту' })).toBeVisible();
	await expect(page.getByText(user.email, { exact: true })).toBeVisible();
	await expect(page.locator('header')).not.toContainText('@artist');
	await expect(page.getByLabel('Пароль', { exact: true })).toHaveCount(0);
	expect(sends).toBe(0);
	const resend = page.getByRole('button', { name: 'Отправить письмо ещё раз' });
	await resend.click();
	await expect(resend).toBeDisabled();
	expect(sends).toBe(1);
	await page.getByRole('link', { name: 'Перейти ко входу' }).click();
	await page.getByLabel('Email', { exact: true }).fill(user.email);
	await page.getByLabel('Пароль', { exact: true }).fill('original password');
	await page.getByRole('button', { name: 'Войти', exact: true }).click();
	await expect(resend).toBeDisabled();
	expect(sends).toBe(1);
	expect(await page.evaluate(() => sessionStorage.getItem('etheyl.verification'))).not.toContain(
		'original password'
	);
});

test('incorrect credentials stay on login and never open verification', async ({ page }) => {
	await mockCsrf(page);
	await page.route('**/api/v1/auth/login', (route) =>
		route.fulfill({ status: 401, json: { error: { code: 'credentials' } } })
	);
	await page.goto('/login');
	await page.getByLabel('Email', { exact: true }).fill(user.email);
	await page.getByLabel('Пароль', { exact: true }).fill('wrong password');
	await page.getByRole('button', { name: 'Войти', exact: true }).click();
	await expect(page.getByRole('alert')).toContainText('Неверный email или пароль');
	await expect(page).toHaveURL('/login');
	expect(await page.evaluate(() => sessionStorage.getItem('etheyl.verification'))).toBeNull();
});

test('email correction reauthenticates, handles errors and persists the new address and cooldown', async ({
	page
}) => {
	const correctedEmail = 'corrected@example.com';
	let attempts = 0;
	await page.route('**/api/v1/auth/change-unverified-email', (route) => {
		attempts++;
		expect(route.request().headers()['x-csrf-token']).toBe(csrf);
		expect(route.request().headers().cookie).toContain(`csrf=${csrf}`);
		expect(route.request().headers().origin).toBe('http://127.0.0.1:4173');
		expect(route.request().postDataJSON()).toEqual({
			email: user.email,
			newEmail: correctedEmail,
			password: attempts === 1 ? 'wrong password' : 'original password'
		});
		if (attempts < 3) {
			return route.fulfill({
				status: attempts === 1 ? 401 : 409,
				json: { error: { code: attempts === 1 ? 'credentials' : 'email_exists' } }
			});
		}
		return route.fulfill({ status: 202, json: { message: 'accepted' } });
	});
	await unverifiedLogin(page);
	await page.getByRole('button', { name: 'Изменить email' }).click();
	await expect(page.getByLabel('Новый email')).toBeFocused();
	await page.getByLabel('Новый email').fill(correctedEmail);
	for (const [index, password] of [
		'wrong password',
		'original password',
		'original password'
	].entries()) {
		await page.getByLabel('Пароль', { exact: true }).fill(password);
		await page.getByRole('button', { name: 'Сохранить и отправить письмо' }).click();
		if (index === 0) {
			await expect(page.getByText('Неверный пароль. Попробуйте ещё раз.')).toBeVisible();
			await expect(page.getByLabel('Пароль', { exact: true })).toHaveValue('');
		} else if (index === 1) {
			await expect(page.getByLabel('Новый email')).toHaveAttribute('aria-invalid', 'true');
		}
	}
	await expect(page.getByRole('heading', { name: 'Проверьте почту' })).toBeVisible();
	await expect(page.getByRole('status')).toContainText('Email изменён');
	await expect(page.getByText(correctedEmail, { exact: true })).toBeVisible();
	await expect(page.getByRole('button', { name: 'Отправить письмо ещё раз' })).toBeDisabled();
	await expect(page.locator('header')).not.toContainText('@artist');
	await page.reload();
	await expect(page.getByText(correctedEmail, { exact: true })).toBeVisible();
	await expect(page.getByRole('button', { name: 'Отправить письмо ещё раз' })).toBeDisabled();
	await page.getByRole('button', { name: 'Изменить email' }).click();
	await page.getByLabel('Пароль', { exact: true }).fill('discard me');
	await page.getByRole('button', { name: 'Отмена', exact: true }).click();
	await page.getByRole('button', { name: 'Изменить email' }).click();
	await expect(page.getByLabel('Пароль', { exact: true })).toHaveValue('');
});

test('resend recovers from network failure and respects server Retry-After', async ({ page }) => {
	await page.clock.install();
	await unverifiedLogin(page);
	const resend = page.getByRole('button', { name: 'Отправить письмо ещё раз' });
	await page.route('**/api/v1/auth/resend-verification', (route) => route.abort('failed'));
	await resend.click();
	await expect(page.getByRole('alert')).toContainText('Нет связи с сервером');
	await expect(resend).toBeEnabled();
	await page.route('**/api/v1/auth/resend-verification', (route) =>
		route.fulfill({
			status: 429,
			headers: { 'Retry-After': '120' },
			json: { error: { code: 'rate_limit' } }
		})
	);
	await resend.click();
	await expect(page.getByRole('alert')).toContainText('через 2 мин.');
	await page.clock.fastForward(61_000);
	await expect(resend).toBeDisabled();
	await page.clock.fastForward(60_000);
	await expect(resend).toBeEnabled();
});

test('legacy resend route redirects to the check-email screen without a separate email form', async ({
	page
}) => {
	await page.goto('/auth/resend');
	await expect(page).toHaveURL('/auth/check-email');
	await expect(page.locator('main form')).toHaveCount(0);
});

test('check-email and email correction fit narrow screens with long addresses', async ({
	page
}) => {
	await page.addInitScript(() => {
		sessionStorage.setItem(
			'etheyl.verification',
			JSON.stringify({
				email: 'very.long.email.address.for.a.creator@very-long-domain.example.com',
				resendAt: Date.now() + 60_000,
				sent: true
			})
		);
	});
	for (const width of [320, 390, 768, 1440]) {
		await page.setViewportSize({ width, height: 900 });
		await page.goto('/auth/check-email');
		await expect(page.getByRole('heading', { name: 'Проверьте почту' })).toBeVisible();
		await expect(page.getByRole('button', { name: 'Изменить email' })).toBeVisible();
		expect(
			await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)
		).toBe(true);
		await page.getByRole('button', { name: 'Изменить email' }).click();
		await expect(page.getByLabel('Новый email')).toBeFocused();
		expect(
			await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)
		).toBe(true);
	}
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
