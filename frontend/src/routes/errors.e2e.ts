import { expect, test } from '@playwright/test';

test.beforeEach(async ({ page }) => {
	await page.route('**/api/v1/me', (route) => route.fulfill({ json: { user: null } }));
});

const clientErrors = [
	{ path: '/403', status: 403, description: 'Доступ запрещён' },
	{ path: '/404', status: 404, description: 'Страница не найдена' },
	{ path: '/missing/unknown-page', status: 404, description: 'Страница не найдена' }
];

for (const { path, status, description } of clientErrors) {
	test(`${path} serves the ${status} page`, async ({ page }) => {
		const response = await page.goto(path);
		expect(response?.status()).toBe(status);
		await expect(page.getByRole('heading', { name: `${status} ${description}` })).toBeVisible();
		await expect(page.getByRole('link', { name: 'На главную' })).toHaveAttribute('href', '/');
		await expect(page.getByRole('link', { name: 'Повторить' })).toHaveCount(0);
	});
}

test('the server error page retries the original request', async ({ page }) => {
	const path = '/error?attempt=1';
	const response = await page.goto(path);
	expect(response?.status()).toBe(500);
	await expect(page.getByRole('heading', { name: '500 Ошибка сервера' })).toBeVisible();

	const [retry] = await Promise.all([
		page.waitForResponse((response) => response.request().isNavigationRequest()),
		page.getByRole('link', { name: 'Повторить' }).click()
	]);
	expect(retry.status()).toBe(500);
	expect(retry.request().method()).toBe('GET');
	await expect(page).toHaveURL(path);

	await page.getByRole('link', { name: 'На главную' }).click();
	await expect(page).toHaveURL('/');
});
