import { expect, test, type APIRequestContext } from '@playwright/test';

const mailpit = process.env.MAILPIT_URL ?? 'http://localhost:8025';

async function emailLink(
	request: APIRequestContext,
	email: string,
	kind: 'verify' | 'reset',
	minMessages = 1
) {
	let link = '';
	await expect
		.poll(
			async () => {
				const response = await request.get(`${mailpit}/api/v1/search`, {
					params: { query: `to:${email}` }
				});
				expect(response.ok()).toBeTruthy();
				const listing = (await response.json()) as { messages: { ID: string }[] };
				if (listing.messages.length < minMessages) return false;
				for (const item of listing.messages) {
					const message = await request.get(`${mailpit}/api/v1/message/${item.ID}`);
					const body = (await message.json()) as { Text: string };
					const match = body.Text.match(
						new RegExp(`https?://[^\\s]+/auth/${kind}#token=[A-Za-z0-9_-]+`)
					);
					if (match) {
						link = match[0];
						return true;
					}
				}
				return false;
			},
			{ timeout: 30_000, message: 'worker должен доставить письмо в Mailpit' }
		)
		.toBe(true);
	return link;
}

test('регистрация, письмо, вход, восстановление и выход через интерфейс', async ({
	page,
	request,
	browser,
	context
}, testInfo) => {
	const email = `e2e-${testInfo.project.name}-${Date.now()}@example.com`;
	const nickname = `maker_${testInfo.project.name}_${Date.now()}`;
	const password = 'Первоначальный пароль 2026';
	const replacement = 'Новый длинный пароль 2026';
	const pageErrors: string[] = [];
	page.on('pageerror', (error) => pageErrors.push(error.message));
	page.on('console', (message) => {
		if (message.text().includes('Content Security Policy')) pageErrors.push('CSP violation');
	});

	await page.goto('/auth/register');
	await expect(page.getByRole('button', { name: 'Отправить письмо ещё раз' })).toHaveCount(0);
	await page.getByLabel('Никнейм', { exact: true }).fill('bad.name');
	expect(
		await page
			.getByLabel('Никнейм', { exact: true })
			.evaluate((node: HTMLInputElement) => node.validity.patternMismatch)
	).toBe(true);
	await page.getByLabel('Никнейм', { exact: true }).fill(nickname.toUpperCase());
	await page.getByLabel('Email', { exact: true }).fill(email);
	await page.getByLabel('Пароль', { exact: true }).fill(password);
	await page.getByLabel('Повторите пароль').fill(password);
	await page.getByRole('button', { name: 'Создать аккаунт' }).click();
	await expect(page.getByRole('heading', { name: 'Проверьте почту.' })).toBeVisible();
	await expect(page.getByRole('button', { name: 'Отправить письмо ещё раз' })).toBeVisible();

	await page.goto('/auth/login');
	await expect(page.getByRole('button', { name: 'Отправить письмо ещё раз' })).toHaveCount(0);
	await expect(page.getByRole('link', { name: /письм/i })).toHaveCount(0);
	await page.getByLabel('Email', { exact: true }).fill(email);
	await page.getByLabel('Пароль', { exact: true }).fill(password);
	await page.getByRole('button', { name: 'Войти', exact: true }).click();
	await expect(page.getByRole('heading', { name: 'Проверьте почту.' })).toBeVisible();
	await page.getByRole('button', { name: 'Вернуться ко входу', exact: true }).click();
	await expect(page.getByLabel('Пароль', { exact: true })).toBeVisible();
	await expect(page.getByRole('button', { name: 'Отправить письмо ещё раз' })).toHaveCount(0);
	await page.getByLabel('Пароль', { exact: true }).fill(password);
	await page.getByRole('button', { name: 'Войти', exact: true }).click();
	await expect(page.getByRole('heading', { name: 'Проверьте почту.' })).toBeVisible();
	await page.getByRole('button', { name: 'Отправить письмо ещё раз' }).click();
	await expect(page.getByText('Новое письмо отправлено', { exact: true })).toBeVisible();
	const verificationLink = await emailLink(request, email, 'verify', 2);
	await page.goto(verificationLink);
	await expect(page.getByTestId('verification-email')).toHaveText(email);
	await page.screenshot({ path: testInfo.outputPath('verify-email.png'), fullPage: true });
	await page.getByRole('button', { name: 'Подтвердить email', exact: true }).click();
	await expect(page).toHaveURL(/\/account$/);
	await expect(page.getByRole('heading', { name: `Здравствуйте, ${nickname}.` })).toBeVisible();
	await page.getByRole('button', { name: 'Выйти', exact: true }).click();
	await expect(page).toHaveURL(/\/auth\/login$/);
	await page.goto('/auth/register');
	await page.getByLabel('Никнейм', { exact: true }).fill(nickname);
	await page.getByLabel('Email', { exact: true }).fill(email);
	await page.getByLabel('Пароль', { exact: true }).fill(password);
	await page.getByLabel('Повторите пароль').fill(password);
	await page.getByRole('button', { name: 'Создать аккаунт' }).click();
	await expect(page.getByRole('alert')).toContainText('Этот email уже зарегистрирован.');
	await expect(page.getByRole('button', { name: 'Отправить письмо ещё раз' })).toHaveCount(0);
	await expect(page.getByRole('link', { name: /письм/i })).toHaveCount(0);
	await expect(page.getByRole('heading', { name: 'Проверьте почту.' })).toHaveCount(0);
	await page.goto('/auth/login');
	await page.getByLabel('Email', { exact: true }).fill(email);
	await page.getByLabel('Пароль', { exact: true }).fill('wrong password');
	await page.getByRole('button', { name: 'Войти', exact: true }).click();
	await expect(page.getByRole('alert')).toContainText('Неверный email или пароль.');
	await page.getByLabel('Пароль', { exact: true }).fill(password);
	await page.getByRole('button', { name: 'Войти', exact: true }).click();
	await expect(page).toHaveURL(/\/account$/);
	await expect(page.getByRole('heading', { name: `Здравствуйте, ${nickname}.` })).toBeVisible();
	await page.reload();
	await expect(page.getByText('Email подтверждён', { exact: true })).toBeVisible();
	await page.getByRole('tab', { name: 'Я автор' }).click();
	await expect(page.getByText('Место для вашего голоса')).toBeVisible();
	await expect(page.locator('body')).toHaveJSProperty(
		'scrollWidth',
		await page.evaluate(() => document.body.clientWidth)
	);
	await page.screenshot({ path: testInfo.outputPath('account.png'), fullPage: true });

	const originalCookies = await context.cookies();
	expect(originalCookies.find((cookie) => cookie.name === 'etheyl_session')?.httpOnly).toBe(true);
	expect(await page.evaluate(() => document.cookie)).not.toContain('etheyl_session');
	const otherSession = await browser.newContext();
	await otherSession.addCookies(originalCookies);

	await page.getByRole('link', { name: 'Изменить пароль через почту' }).click();
	await page.getByLabel('Email', { exact: true }).fill(email);
	await page.getByRole('button', { name: 'Отправить ссылку' }).click();
	await expect(page.getByRole('heading', { name: 'Проверьте почту.' })).toBeVisible();
	await page.goto(await emailLink(request, email, 'reset'));
	await page.getByLabel('Новый пароль', { exact: true }).fill(password);
	await page.getByLabel('Повторите пароль').fill(password);
	await page.getByRole('button', { name: 'Сохранить пароль' }).click();
	await expect(page.getByRole('alert')).toContainText(
		'Новый пароль должен отличаться от текущего.'
	);
	await page.getByLabel('Новый пароль', { exact: true }).fill(replacement);
	await page.getByLabel('Повторите пароль').fill(replacement);
	await page.getByRole('button', { name: 'Сохранить пароль' }).click();
	await expect(page.getByText('Пароль изменён. Все предыдущие сеансы завершены.')).toBeVisible();

	const stale = await otherSession.newPage();
	await stale.goto(`${testInfo.project.use.baseURL ?? 'http://localhost:8080'}/account`);
	await expect(stale).toHaveURL(/\/auth\/login$/);
	await otherSession.close();

	await page.getByRole('link', { name: 'Перейти ко входу' }).click();
	await page.getByLabel('Email', { exact: true }).fill(email);
	await page.getByLabel('Пароль', { exact: true }).fill(replacement);
	await page.getByRole('button', { name: 'Войти', exact: true }).click();
	await expect(page).toHaveURL(/\/account$/);
	const denied = await page.goto('/admin');
	expect(denied?.status()).toBe(403);
	await expect(page.getByText('Недостаточно прав.', { exact: true })).toBeVisible();
	await page.goto('/account');
	await page.getByRole('button', { name: 'Выйти', exact: true }).click();
	await expect(page).toHaveURL(/\/auth\/login$/);
	await page.goto('/account');
	await expect(page).toHaveURL(/\/auth\/login$/);
	expect(pageErrors).toEqual([]);
});

test('адаптивные экраны, клавиатура и недействительная ссылка', async ({ page }, testInfo) => {
	await page.goto('/');
	await page.keyboard.press('Tab');
	await expect(page.getByRole('link', { name: 'Перейти к содержимому' })).toBeFocused();
	await page.keyboard.press('Enter');
	await expect(page.locator('#main')).toBeFocused();
	const cards = page.locator('[data-slot="card"]');
	for (const card of await cards.all()) {
		const bounds = await card.boundingBox();
		const footer = await card.locator('[data-slot="card-footer"]').boundingBox();
		expect(bounds).not.toBeNull();
		expect(footer).not.toBeNull();
		expect(Math.abs(bounds!.y + bounds!.height - footer!.y - footer!.height)).toBeLessThan(2);
	}
	await page.screenshot({ path: testInfo.outputPath('home.png'), fullPage: true });
	for (const path of [
		'/auth/register',
		'/auth/login',
		'/auth/forgot',
		'/auth/resend',
		'/auth/reset',
		'/auth/verify'
	]) {
		await page.goto(path);
		expect(
			await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)
		).toBe(true);
	}
	await expect(page.getByText('В ссылке нет кода')).toBeVisible();
	await expect(page.getByRole('button', { name: 'Подтвердить email', exact: true })).toBeDisabled();
	await page.goto('/auth/verify#token=bad');
	await expect(page.getByRole('alert')).toContainText('Ссылка недействительна');
	await expect(page.getByRole('button', { name: 'Подтвердить email', exact: true })).toBeDisabled();
});
