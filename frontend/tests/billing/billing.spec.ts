import { expect, test, type BrowserContext } from '@playwright/test';

async function register(context: BrowserContext, nickname: string, origin: string) {
	const email = `${nickname}@example.com`;
	const csrf = await context.request.get('/api/v1/auth/csrf');
	const { token } = await csrf.json();
	const headers = { Origin: origin, 'X-CSRF-Token': token };
	expect(
		(
			await context.request.post('/api/v1/auth/register', {
				headers,
				data: { email, nickname, password: 'Пароль тестовых подписок 2026' }
			})
		).status()
	).toBe(202);
	let actionToken = '';
	await expect
		.poll(
			async () => {
				const listing = await context.request.get(
					`${process.env.MAILPIT_URL ?? 'http://localhost:8025'}/api/v1/search`,
					{ params: { query: `to:${email}` } }
				);
				const { messages } = await listing.json();
				if (!messages?.length) return false;
				const mail = await context.request.get(
					`${process.env.MAILPIT_URL ?? 'http://localhost:8025'}/api/v1/message/${messages[0].ID}`
				);
				actionToken = (await mail.json()).Text.match(/#token=([A-Za-z0-9_-]+)/)?.[1] ?? '';
				return !!actionToken;
			},
			{ timeout: 30000 }
		)
		.toBe(true);
	expect(
		(
			await context.request.post('/api/v1/auth/verify-email', {
				headers,
				data: { token: actionToken }
			})
		).status()
	).toBe(200);
	return headers;
}

test('условия подписки, явное согласие, ошибка оплаты и пустые состояния', async ({
	page,
	context,
	browser
}, testInfo) => {
	test.setTimeout(90000);
	const origin = new URL(testInfo.project.use.baseURL as string).origin;
	const suffix = `${testInfo.project.name}_${Date.now()}`;
	const author = `ba_${suffix}`;
	const reader = `br_${suffix}`;
	const authorContext = await browser.newContext({ baseURL: origin });
	try {
		const authorHeaders = await register(authorContext, author, origin);
		expect(
			(
				await authorContext.request.put('/api/v1/me/profile', {
					headers: authorHeaders,
					data: {
						nickname: author,
						displayName: 'Автор теста подписок',
						description: 'Творческий дневник'
					}
				})
			).status()
		).toBe(200);
		const response = await authorContext.request.post('/api/v1/me/tiers', {
			headers: authorHeaders,
			data: { name: 'Мастерская', description: 'Все закрытые истории', priceKopecks: 19950 }
		});
		const { tier } = await response.json();
		await register(context, reader, origin);
		const errors: string[] = [];
		page.on('pageerror', (error) => errors.push(error.message));
		await page.goto(`/authors/${author}`);
		await page.getByRole('link', { name: 'Оформить подписку', exact: true }).click();
		await expect(page.getByRole('heading', { name: 'Поддержать автора' })).toBeVisible();
		await expect(page.getByLabel('Согласен на автопродление', { exact: true })).not.toBeChecked();
		await page.getByLabel('Согласен на автопродление', { exact: true }).check();
		await expect(page.getByLabel('Согласен на автопродление', { exact: true })).toBeChecked();
		let checkoutBody: unknown;
		await page.route('**/api/v1/billing/checkout', async (route) => {
			checkoutBody = route.request().postDataJSON();
			await route.fulfill({
				status: 503,
				contentType: 'application/json',
				body: JSON.stringify({
					error: { code: 'payments_disabled', message: 'Оплата пока не настроена.' }
				})
			});
		});
		await page.getByRole('button', { name: /Перейти к оплате/ }).click();
		await expect(
			page.getByRole('alert').filter({ hasText: 'Не удалось оформить подписку' })
		).toContainText('Оплата пока не настроена.');
		expect(checkoutBody).toEqual({ tierId: tier.id, priceKopecks: 19950, autoRenew: true });
		await expect(page.getByLabel('Согласен на автопродление', { exact: true })).toBeChecked();
		await page.screenshot({ path: testInfo.outputPath('checkout.png'), fullPage: true });
		expect(
			await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)
		).toBe(true);
		await page.goto('/account/subscriptions');
		await expect(page.getByText('Выберите своего автора', { exact: true })).toBeVisible();
		await page.getByRole('link', { name: 'История платежей', exact: true }).click();
		await expect(page.getByText('Платежей пока нет', { exact: true })).toBeVisible();
		await page.goto('/account');
		await expect(page.getByRole('link', { name: 'Мои подписки', exact: true })).toBeVisible();
		expect(errors).toEqual([]);
	} finally {
		await authorContext.close();
	}
});
