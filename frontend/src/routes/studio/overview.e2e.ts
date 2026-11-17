import { expect, test, type Page, type Request } from '@playwright/test';
import type { Analytics, Post, Profile, Tier } from '#lib/api/client.js';

const csrf = 'a'.repeat(43);
const user = {
	id: '1',
	email: 'artist@example.com',
	nickname: 'artist',
	emailVerified: true,
	isAdmin: false
};
const profile: Profile = {
	id: '1',
	displayName: 'Мастерская',
	nickname: 'artist',
	description: '',
	avatar: null,
	cover: null
};
const tier: Tier = {
	id: '1',
	name: 'Мастерская',
	description: '',
	priceKopecks: 15050,
	position: 1,
	used: false,
	archived: false
};
const analytics: Analytics = {
	from: '2026-10-20',
	to: '2026-11-18',
	asOf: '2026-11-18T12:00:00Z',
	currency: 'RUB',
	timezone: 'UTC',
	test: true,
	paymentCount: 0,
	amountKopecks: 0,
	activeSubscriptions: 14,
	daily: [],
	tiers: []
};
const publishedAt = '2026-11-18T12:00:00Z';

function post(id: string, published: string | null = null): Post {
	return {
		id,
		authorNickname: profile.nickname,
		title: `Работа ${id}`,
		teaser: '',
		minimumTierId: null,
		canRead: true,
		bodyMarkdown: '',
		publishedAt: published
	};
}

async function mockAccount(page: Page) {
	await page.route('**/api/v1/me', (route) => route.fulfill({ json: { user } }));
	await page.route('**/api/v1/auth/csrf', (route) =>
		route.fulfill({
			json: { token: csrf },
			headers: { 'Set-Cookie': `csrf=${csrf}; HttpOnly; Path=/; SameSite=Lax` }
		})
	);
}

function expectCsrf(request: Request) {
	expect(request.headers().origin).toBe(new URL(request.url()).origin);
	expect(request.headers()['x-csrf-token']).toBe(csrf);
	expect(request.headers().cookie).toContain(`csrf=${csrf}`);
}

test('Studio guides setup and supports a public post before the first tier', async ({ page }) => {
	await mockAccount(page);
	let savedProfile: Profile | null = null;
	let savedPost: Post | null = null;
	let tiers: Tier[] = [{ ...tier, id: '2', archived: true }];
	await page.route('**/api/v1/me/profile', async (route) => {
		const request = route.request();
		if (request.method() === 'PUT') {
			expectCsrf(request);
			savedProfile = { ...profile, ...request.postDataJSON() };
		}
		await route.fulfill({ json: { profile: savedProfile } });
	});
	await page.route('**/api/v1/me/tiers', async (route) => {
		const request = route.request();
		if (request.method() === 'POST') {
			expectCsrf(request);
			expect(request.postDataJSON().priceKopecks).toBe(15050);
			tiers = [tier, ...tiers];
			await route.fulfill({ status: 201, json: { tier } });
			return;
		}
		await route.fulfill({ json: { tiers } });
	});
	await page.route('**/api/v1/me/analytics', async (route) => {
		expect(savedProfile).not.toBeNull();
		await route.fulfill({ json: analytics });
	});
	await page.route('**/api/v1/me/posts', async (route) => {
		const request = route.request();
		if (request.method() === 'POST') {
			expectCsrf(request);
			expect(request.postDataJSON().minimumTierId).toBeNull();
			savedPost = { ...post('7'), ...request.postDataJSON() };
			await route.fulfill({ status: 201, json: { post: savedPost } });
			return;
		}
		await route.fulfill({ json: { posts: savedPost ? [savedPost] : [], hasMore: false } });
	});
	await page.route('**/api/v1/me/posts/7/publish', async (route) => {
		expectCsrf(route.request());
		savedPost = { ...savedPost!, publishedAt };
		await route.fulfill({ json: { post: savedPost } });
	});
	await page.route('**/api/v1/me/posts/7', (route) => route.fulfill({ json: { post: savedPost } }));

	await page.goto('/studio');
	const main = page.getByRole('main');
	const setup = main.getByRole('region', { name: 'Настройте студию' });
	await expect(setup).toContainText('0 из 3');
	await expect(
		main.getByText('Аналитика станет доступна после сохранения профиля автора.')
	).toBeVisible();
	await expect(main.getByText('Тестовый режим')).toHaveCount(0);
	await setup.getByRole('link', { name: 'Создать профиль' }).click();
	const uploadCover = main.getByRole('button', { name: 'Загрузить обложку' });
	await expect(uploadCover).toBeDisabled();
	await page.getByLabel('Отображаемое имя').fill('Мастерская');
	await page.getByLabel('Ник', { exact: true }).fill('artist');
	await main.getByRole('button', { name: 'Сохранить', exact: true }).click();
	await expect(main.getByText('Сохранено', { exact: true })).toBeVisible();
	await expect(uploadCover).toBeEnabled();
	await page
		.getByRole('navigation', { name: 'Студия', exact: true })
		.getByRole('link', { name: 'Обзор' })
		.click();
	await expect(page).toHaveURL('/studio');
	await expect(setup).toContainText('1 из 3');
	await expect(main.getByRole('link', { name: 'Открыть профиль' })).toHaveAttribute(
		'href',
		'/authors/artist'
	);

	await main.getByRole('link', { name: 'Новый пост' }).click();
	await page.getByLabel('Заголовок').fill('Первый пост');
	await page.getByLabel('Текст поста').fill('Моя первая работа');
	await page.getByRole('button', { name: 'Опубликовать', exact: true }).click();
	await expect(page).toHaveURL('/studio/posts/7/edit');
	await page
		.getByRole('navigation', { name: 'Студия', exact: true })
		.getByRole('link', { name: 'Обзор' })
		.click();
	await expect(setup).toContainText('2 из 3');
	const recentPost = main.getByRole('link', { name: /Первый пост Опубликован/ });
	await expect(recentPost).toHaveAttribute('href', '/studio/posts/7/edit');
	await expect(recentPost.locator('time')).toHaveAttribute('datetime', publishedAt);

	await setup.getByRole('link', { name: 'Добавить уровень' }).click();
	await page.getByLabel('Название', { exact: true }).fill('Поддержка');
	await page.getByLabel('Стоимость в месяц, ₽').fill('150,50');
	await page.getByRole('button', { name: 'Создать уровень' }).click();
	await expect(page.getByText('Уровень создан.')).toBeVisible();
	await page
		.getByRole('navigation', { name: 'Студия', exact: true })
		.getByRole('link', { name: 'Обзор' })
		.click();
	await expect(setup).toHaveCount(0);
	await expect(main.getByRole('link', { name: 'Новый пост' })).toBeVisible();
});

test('Studio checks older publications, lists three drafts, and retries analytics on mobile', async ({
	page
}) => {
	await page.setViewportSize({ width: 360, height: 800 });
	await mockAccount(page);
	let failAnalytics = true;
	await page.route('**/api/v1/me/profile', (route) => route.fulfill({ json: { profile } }));
	await page.route('**/api/v1/me/tiers', (route) => route.fulfill({ json: { tiers: [tier] } }));
	await page.route('**/api/v1/me/posts', (route) =>
		route.fulfill({
			json: {
				posts: Array.from({ length: 24 }, (_, index) => post(String(25 - index))),
				hasMore: true
			}
		})
	);
	await page.route('**/api/v1/me/posts?offset=24', (route) =>
		route.fulfill({ json: { posts: [post('1', publishedAt)], hasMore: false } })
	);
	await page.route('**/api/v1/me/analytics', (route) =>
		failAnalytics ? route.fulfill({ status: 503, json: {} }) : route.fulfill({ json: analytics })
	);

	const olderPage = page.waitForResponse('**/api/v1/me/posts?offset=24');
	await page.goto('/studio');
	await olderPage;
	const posts = page.getByRole('region', { name: 'Последние посты' });
	const latest = posts.getByRole('link', { name: /Работа/ });
	await expect(latest).toHaveCount(3);
	await expect(latest).toHaveText([
		/Работа 25.*Черновик/,
		/Работа 24.*Черновик/,
		/Работа 23.*Черновик/
	]);
	await expect(page.getByRole('region', { name: 'Настройте студию' })).toHaveCount(0);
	const metrics = page.getByRole('region', { name: 'Аналитика' });
	await expect(metrics.getByText('Не удалось загрузить аналитику.')).toBeVisible();
	await expect(metrics.locator('dd')).toHaveCount(0);
	failAnalytics = false;
	await metrics.getByRole('button', { name: 'Повторить' }).click();
	await expect(metrics.getByText('Тестовый режим')).toBeVisible();
	await expect(metrics.locator('dd')).toHaveText([/^0\s*₽$/, '14']);

	await page.getByText('Меню студии', { exact: true }).click();
	const nav = page.getByRole('navigation', { name: 'Студия', exact: true });
	await expect(nav.getByRole('link', { name: 'Обзор' })).toHaveAttribute('aria-current', 'page');
	await nav.getByRole('link', { name: 'Аналитика' }).click();
	await expect(page).toHaveURL('/studio/analytics');
	await expect(page.getByRole('heading', { name: 'Аналитика', level: 1 })).toBeVisible();
	await expect(nav).toBeHidden();
});

test('Profile previews draft text and applies each image crop independently', async ({ page }) => {
	await mockAccount(page);
	let savedProfile = { ...profile };
	let rejectCover = true;
	let rejectRemoval = true;
	let uploadCount = 0;
	await page.route('**/api/v1/me/profile', async (route) => {
		if (route.request().method() === 'PUT') {
			expectCsrf(route.request());
			await route.fulfill({ status: 409, json: { error: { code: 'nickname_exists' } } });
		} else {
			await route.fulfill({ json: { profile: savedProfile } });
		}
	});
	await page.route('**/api/v1/me/profile/images/*', async (route) => {
		const request = route.request();
		expectCsrf(request);
		const kind = new URL(request.url()).pathname.endsWith('/cover') ? 'cover' : 'avatar';
		if (request.method() === 'DELETE') {
			if (rejectRemoval) {
				rejectRemoval = false;
				await route.fulfill({ status: 503, json: { error: { code: 'unavailable' } } });
				return;
			}
			savedProfile = { ...savedProfile, [kind]: null };
		} else {
			expect(request.method()).toBe('PUT');
			expect(request.headers()['content-type']).toBe('image/png');
			const bytes = request.postDataBuffer()!;
			const width = bytes.readUInt32BE(16);
			const height = bytes.readUInt32BE(20);
			expect(width).toBe(height * (kind === 'cover' ? 4 : 1));
			uploadCount += 1;
			if (kind === 'cover' && rejectCover) {
				rejectCover = false;
				await route.fulfill({ status: 503, json: { error: { code: 'storage_unavailable' } } });
				return;
			}
			savedProfile = {
				...savedProfile,
				[kind]: {
					url: '/artwork/garden.webp',
					contentType: 'image/png',
					width,
					height,
					sizeBytes: bytes.length
				}
			};
		}
		await route.fulfill({ json: { profile: savedProfile } });
	});

	await page.goto('/studio/profile');
	await page.getByLabel('Ник', { exact: true }).fill('New_Artist');
	await page.getByLabel('Отображаемое имя').fill('Новая мастерская');
	await page.getByLabel('Описание').fill('Описание с текущими правками.');
	await page.getByRole('link', { name: 'Как это видят читатели' }).click();
	await expect(page.getByRole('heading', { name: 'Новая мастерская' })).toBeVisible();
	await expect(page.getByText('@new_artist', { exact: true })).toBeVisible();
	await expect(page.getByText('Описание с текущими правками.')).toBeVisible();
	await page.goBack();
	await expect(page.getByLabel('Описание')).toHaveValue('Описание с текущими правками.');

	await page.getByRole('button', { name: 'Сохранить', exact: true }).click();
	await expect(page.getByRole('alert')).toHaveText('Этот ник уже занят. Попробуйте другой.');
	await expect(page.getByLabel('Ник', { exact: true })).toBeFocused();
	await expect(page.getByLabel('Отображаемое имя')).toHaveValue('Новая мастерская');

	const png = await page.evaluate(() => {
		const canvas = document.createElement('canvas');
		canvas.width = 400;
		canvas.height = 240;
		const drawing = canvas.getContext('2d')!;
		const gradient = drawing.createLinearGradient(0, 0, 400, 240);
		gradient.addColorStop(0, '#315de8');
		gradient.addColorStop(1, '#118961');
		drawing.fillStyle = gradient;
		drawing.fillRect(0, 0, 400, 240);
		return canvas.toDataURL().split(',')[1];
	});
	const file = { name: 'picture.png', mimeType: 'image/png', buffer: Buffer.from(png, 'base64') };
	const cover = page.getByRole('region', { name: 'Обложка' });
	const avatar = page.getByRole('region', { name: 'Аватар' });
	const crop = page.getByRole('dialog');
	const apply = crop.getByRole('button', { name: 'Применить и загрузить' });

	await cover.locator('input[type=file]').setInputFiles(file);
	await expect(crop.getByRole('application', { name: 'Положение изображения' })).toBeFocused();
	await crop.getByRole('slider', { name: 'Масштаб' }).press('ArrowRight');
	await apply.click();
	await expect(crop.getByText('Хранилище изображений недоступно.')).toBeVisible();
	await expect(cover.getByRole('img', { name: 'Обложка профиля' })).toHaveCount(0);
	await apply.click();
	await expect(cover.getByRole('status')).toHaveText('Обложка обновлена.');
	await expect(crop).toHaveCount(0);

	await avatar.locator('input[type=file]').setInputFiles(file);
	await crop.getByRole('button', { name: 'Отмена' }).click();
	await expect(crop).toHaveCount(0);
	expect(uploadCount).toBe(2);
	await avatar.locator('input[type=file]').setInputFiles(file);
	await apply.click();
	await expect(avatar.getByRole('status')).toHaveText('Аватар обновлён.');

	await page.getByRole('link', { name: 'Как это видят читатели' }).click();
	await expect(page.getByRole('img', { name: 'Обложка страницы автора' })).toBeVisible();
	await page.getByRole('link', { name: 'Вернуться к редактированию' }).click();
	await expect(page.getByLabel('Отображаемое имя')).toHaveValue('Новая мастерская');
	const remove = cover.getByRole('button', { name: 'Удалить обложку' });
	await remove.press('Enter');
	await expect(cover.getByRole('status')).toContainText('Сервис временно недоступен.');
	await expect(remove).toBeFocused();
	await remove.press('Enter');
	await expect(cover.getByRole('status')).toHaveText('Обложка удалена.');
	await expect(cover.getByRole('button', { name: 'Загрузить обложку' })).toBeFocused();
	await expect(avatar.getByRole('img', { name: 'Аватар профиля' })).toBeVisible();
});
