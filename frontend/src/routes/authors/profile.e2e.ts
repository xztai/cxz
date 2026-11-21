import { expect, test, type Page } from '@playwright/test';
import type { Payment, PostListItem, Profile, Subscription, Tier, User } from '#lib/api/client.js';

const profile: Profile = {
	id: '42',
	nickname: 'quiet-garden',
	displayName: 'Тихий сад',
	description: 'Ботанические иллюстрации и заметки из мастерской.',
	cover: null,
	avatar: null
};
const routePath = `/authors/${profile.nickname}`;
const viewer: User = {
	id: '13',
	nickname: 'reader',
	email: 'reader@example.com',
	emailVerified: true,
	isAdmin: false
};
const tiers: Tier[] = [
	{ id: '1', name: 'Семена', priceKopecks: 20000, position: 1 },
	{ id: '2', name: 'Оранжерея', priceKopecks: 45050, position: 3 },
	{ id: '3', name: 'Гербарий', priceKopecks: 70000, position: 2 }
].map((tier) => ({ ...tier, description: 'Заметки и новые эскизы', used: true, archived: false }));

function post(id: string, title: string, minimumTierName: string | null): PostListItem {
	return {
		id,
		authorNickname: profile.nickname,
		title,
		teaser: `Анонс «${title}»`,
		publishedAt: '2026-11-15T10:00:00Z',
		minimumTierId: minimumTierName ? '20' : null,
		minimumTierName
	};
}

async function mockProfile(page: Page, user: User | null) {
	await page.route('**/api/v1/me', (route) => route.fulfill({ json: { user } }));
	await page.route(`**/api/v1${routePath}`, (route) => route.fulfill({ json: { profile, tiers } }));
	await page.route(`**/api/v1${routePath}/posts?*`, (route) =>
		route.fulfill({ json: { posts: [], hasMore: false } })
	);
}

test('author profile shows the current tier by position and appends public and locked posts', async ({
	page
}) => {
	await mockProfile(page, viewer);
	const subscription: Subscription = {
		id: '10',
		authorId: profile.id,
		nickname: profile.nickname,
		displayName: profile.displayName,
		tierId: '3',
		tierName: 'Гербарий',
		priceKopecks: 70000,
		autoRenew: false,
		paidUntil: '2026-12-01T00:00:00Z',
		nextTierId: null,
		nextPriceKopecks: null,
		nextTierName: '',
		stoppedReason: '',
		active: true
	};
	await page.route('**/api/v1/me/subscriptions?*', (route) => {
		const first = new URL(route.request().url()).searchParams.get('offset') === '0';
		return route.fulfill({
			json: {
				subscriptions: first ? [{ ...subscription, authorId: '4' }] : [subscription],
				hasMore: first,
				paymentsEnabled: true
			}
		});
	});
	let nextAttempts = 0;
	await page.route(`**/api/v1${routePath}/posts?*`, (route) => {
		const offset = new URL(route.request().url()).searchParams.get('offset');
		if (offset === '0') {
			return route.fulfill({
				json: { posts: [post('5', 'Зимние эскизы', 'Архивный клуб')], hasMore: true }
			});
		}
		expect(offset).toBe('1');
		if (nextAttempts++ === 0) return route.fulfill({ status: 503, json: {} });
		return route.fulfill({ json: { posts: [post('4', 'Начало сезона', null)], hasMore: false } });
	});

	await page.goto(routePath);
	const plans = page.getByRole('region', { name: 'Уровни подписки' });
	await expect(plans.getByRole('heading', { level: 3 })).toHaveText([
		'Оранжерея',
		'Гербарий',
		'Семена'
	]);
	const current = plans.getByRole('listitem').filter({ hasText: 'Гербарий' });
	await expect(current.getByText('Ваш уровень')).toBeVisible();
	await expect(
		plans.getByRole('button', { name: /Подписаться на уровень «Семена»/ })
	).toBeDisabled();
	const posts = page.getByRole('region', { name: 'Публикации' });
	await expect(posts.getByText('Архивный клуб')).toBeVisible();
	await posts.getByRole('button', { name: 'Показать ещё' }).click();
	await expect(posts.getByRole('alert')).toBeVisible();
	await posts.getByRole('button', { name: 'Попробовать ещё раз' }).click();
	await expect(posts.getByRole('heading', { level: 3 })).toHaveText([
		'Зимние эскизы',
		'Начало сезона'
	]);
	await expect(posts.getByText('Публично')).toBeVisible();
	await expect(posts.getByRole('button')).toHaveCount(0);
});

test('checkout confirms the chosen tier price and renewal consent before showing payment status', async ({
	page
}) => {
	await mockProfile(page, viewer);
	await page.route('**/api/v1/me/subscriptions?*', (route) =>
		route.fulfill({ json: { subscriptions: [], hasMore: false, paymentsEnabled: true } })
	);
	await page.route('**/api/v1/auth/csrf', (route) =>
		route.fulfill({ json: { token: 'test-csrf' } })
	);
	const payment: Payment = {
		id: 'queued-payment',
		agreementId: '103',
		status: 'queued',
		amountKopecks: 45050,
		confirmationUrl: '',
		renewal: false,
		test: true,
		createdAt: '2026-11-15T10:00:00Z',
		periodStart: null,
		periodEnd: null,
		failureCode: ''
	};
	let currentPayment = payment;
	const checkoutRequests: unknown[] = [];
	await page.route('**/api/v1/billing/checkout', async (route) => {
		expect(route.request().headers()['x-csrf-token']).toBe('test-csrf');
		checkoutRequests.push(route.request().postDataJSON());
		if (checkoutRequests.length === 1) {
			return route.fulfill({ status: 409, json: { error: { code: 'price_changed' } } });
		}
		return route.fulfill({ status: 202, json: { payment } });
	});
	await page.route(`**/api/v1/me/payments/${payment.id}`, (route) =>
		route.fulfill({ json: { payment: currentPayment } })
	);

	await page.goto(routePath);
	const open = page.getByRole('button', { name: /Подписаться на уровень «Семена»/ });
	await open.click();
	const dialog = page.getByRole('dialog', { name: 'Оформить подписку' });
	await expect(dialog.getByLabel('Уровень подписки')).toHaveValue('1');
	await expect(dialog.getByRole('checkbox')).not.toBeChecked();
	await page.keyboard.press('Escape');
	await expect(dialog).toBeHidden();
	await expect(open).toBeFocused();
	await open.click();
	await dialog.getByRole('button', { name: 'Перейти к оплате' }).click();
	await expect(dialog.getByRole('alert')).toContainText('Цена изменилась');
	await dialog.getByLabel('Уровень подписки').selectOption('2');
	await expect(dialog.locator('output')).toHaveText('450,5 ₽');
	await dialog.getByRole('checkbox').check();
	await dialog.getByRole('button', { name: 'Перейти к оплате' }).click();
	await expect(page).toHaveURL(`/account/payments/${payment.id}`);
	await expect(page.getByText('Готовим ссылку на оплату')).toBeVisible();
	currentPayment = {
		...payment,
		status: 'pending',
		confirmationUrl: 'https://yoomoney.ru/checkout'
	};
	await page.getByRole('button', { name: 'Проверить статус' }).click();
	await expect(page.getByRole('link', { name: 'Продолжить оплату' })).toHaveAttribute(
		'href',
		currentPayment.confirmationUrl
	);
	currentPayment = { ...payment, status: 'succeeded' };
	await page.getByRole('button', { name: 'Проверить статус' }).click();
	await expect(page.getByText('Подписка оплачена')).toBeVisible();
	expect(checkoutRequests).toEqual([
		{ tierId: '1', priceKopecks: 20000, autoRenew: false },
		{ tierId: '2', priceKopecks: 45050, autoRenew: true }
	]);
});

test('a guest can return to the author after signing in from the subscription dialog', async ({
	page
}) => {
	await mockProfile(page, null);
	await page.route('**/api/v1/auth/csrf', (route) =>
		route.fulfill({ json: { token: 'test-csrf' } })
	);
	await page.route('**/api/v1/auth/login', (route) => route.fulfill({ json: { user: viewer } }));
	await page.route('**/api/v1/me/subscriptions?*', (route) =>
		route.fulfill({ json: { subscriptions: [], hasMore: false, paymentsEnabled: true } })
	);

	await page.goto(routePath);
	await page.getByRole('button', { name: /Подписаться на уровень «Семена»/ }).click();
	await page.getByRole('link', { name: 'Войти для оплаты' }).click();
	await page.getByLabel('Email').fill(viewer.email);
	await page.getByLabel('Пароль', { exact: true }).fill('test-password');
	await page.getByRole('button', { name: 'Войти', exact: true }).click();
	await page.getByRole('link', { name: 'Вернуться к автору' }).click();
	await expect(page).toHaveURL(routePath);
	await expect(page.getByRole('heading', { name: profile.displayName })).toBeVisible();
});
