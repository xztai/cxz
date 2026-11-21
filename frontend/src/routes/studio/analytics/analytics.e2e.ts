import { expect, test } from '@playwright/test';
import type { Analytics } from '#lib/api/client.js';

test.use({
	timezoneId: 'America/Los_Angeles',
	viewport: { width: 360, height: 800 },
	hasTouch: true
});

test('creator analytics uses inclusive UTC dates and keeps the selected range on retry', async ({
	page
}) => {
	await page.clock.setFixedTime(new Date('2026-11-21T00:05:00Z'));
	const asOf = '2026-11-21T00:04:15Z';
	const requests: string[][] = [];
	let fail = false;
	await page.route('**/api/v1/me', (route) =>
		route.fulfill({
			json: {
				user: {
					id: '1',
					email: 'artist@example.com',
					nickname: 'artist',
					emailVerified: true,
					isAdmin: false
				}
			}
		})
	);
	await page.route('**/api/v1/me/profile', (route) =>
		route.fulfill({
			json: {
				ownerId: '1',
				profile: {
					id: '1',
					nickname: 'artist',
					displayName: 'Мастерская',
					description: '',
					avatar: null,
					cover: null
				}
			}
		})
	);
	await page.route('**/api/v1/me/analytics?*', (route) => {
		const { searchParams } = new URL(route.request().url());
		const from = searchParams.get('from') ?? '';
		const to = searchParams.get('to') ?? '';
		requests.push([from, to]);
		expect(route.request().headers()['x-expected-user-id']).toBe('1');
		if (fail) return route.fulfill({ status: 503, json: {} });
		const empty = from === to;
		const dayMs = 86_400_000;
		const first = Date.parse(from);
		const days = (Date.parse(to) - first) / dayMs + 1;
		const daily = Array.from({ length: days }, (_, index) => {
			const date = new Date(first + index * dayMs).toISOString().slice(0, 10);
			const paymentCount = empty ? 0 : date === from ? 1 : date === to ? 2 : 0;
			return { date, paymentCount, amountKopecks: paymentCount * 15000 };
		});
		return route.fulfill({
			json: {
				from,
				to,
				asOf,
				currency: 'RUB',
				timezone: 'UTC',
				test: true,
				paymentCount: empty ? 0 : 3,
				amountKopecks: empty ? 0 : 45000,
				activeSubscriptions: 5,
				daily,
				tiers: [{ id: '1', name: 'Первые подписчики', archived: true, activeSubscriptions: 5 }]
			} satisfies Analytics
		});
	});

	await page.goto('/studio/analytics');
	const presets = page.getByRole('navigation', { name: 'Быстрый выбор периода' });
	const from = page.getByLabel('Дата начала');
	const to = page.getByLabel('Дата окончания');
	const apply = page.getByRole('button', { name: 'Применить' });
	const table = page.getByRole('table', { name: 'Подписчики по уровням' });
	await expect(presets.getByRole('link', { name: '30 дней' })).toHaveAttribute(
		'aria-current',
		'true'
	);
	await expect(table.getByRole('row', { name: /Первые подписчики В архиве 5/ })).toBeVisible();
	await expect(page.locator(`time[datetime="${asOf}"]`)).toHaveText(
		/21 нояб\. 2026 г\., 00:04 UTC/
	);
	expect(requests.at(-1)).toEqual(['2026-10-23', '2026-11-21']);

	for (const [days, start] of [
		[7, '2026-11-15'],
		[90, '2026-08-24']
	] as const) {
		await presets.getByRole('link', { name: `${days} дней` }).click();
		await expect(page).toHaveURL(`/studio/analytics?days=${days}`);
		await expect(from).toHaveValue(start);
		await expect(to).toHaveValue('2026-11-21');
		await expect(table).toBeVisible();
		expect(requests.at(-1)).toEqual([start, '2026-11-21']);
	}

	const chart = page.getByRole('group', { name: 'Дневные платежи, UTC' });
	const previousDay = page.getByRole('button', { name: 'Предыдущий день' });
	const nextDay = page.getByRole('button', { name: 'Следующий день' });
	await expect(chart.getByRole('button')).toHaveCount(90);
	await expect(nextDay).toBeDisabled();
	await previousDay.tap();
	await expect(page.locator('time[datetime="2026-11-20"]')).toBeVisible();
	await nextDay.tap();
	await expect(page.locator('time[datetime="2026-11-21"]')).toBeVisible();
	await expect(nextDay).toBeDisabled();

	await chart.getByRole('button', { name: /21 ноября/ }).focus();
	await page.keyboard.press('ArrowLeft');
	await expect(chart.getByRole('button', { name: /20 ноября.*Платежей 0/ })).toBeFocused();
	await page.keyboard.press('Home');
	await expect(chart.getByRole('button', { name: /24 августа.*Платежей 1/ })).toBeFocused();
	await expect(previousDay).toBeDisabled();

	await from.fill('2026-11-22');
	await apply.click();
	await expect(page.getByRole('alert')).toHaveText(
		'Дата начала должна быть не позже даты окончания.'
	);
	expect(requests).toHaveLength(3);

	fail = true;
	await from.fill('2026-11-17');
	await to.fill('2026-11-17');
	await apply.click();
	await expect(page).toHaveURL('/studio/analytics?from=2026-11-17&to=2026-11-17');
	await expect(page.getByText('Не удалось загрузить аналитику.')).toBeVisible();
	fail = false;
	await page.getByRole('button', { name: 'Повторить' }).click();
	await expect(page.getByText('За выбранный период платежей нет.')).toBeVisible();
	await expect(table.getByRole('row', { name: /Первые подписчики В архиве 5/ })).toBeVisible();
	expect(requests.slice(-2)).toEqual([
		['2026-11-17', '2026-11-17'],
		['2026-11-17', '2026-11-17']
	]);

	await page.goBack();
	await expect(from).toHaveValue('2026-08-24');
	await expect(chart).toBeVisible();
});
