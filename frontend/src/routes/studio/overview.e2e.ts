import { expect, test, type BrowserContext, type Page, type Request } from '@playwright/test';
import type {
	Analytics,
	OwnPostSummary,
	Post,
	PostFile,
	Profile,
	Tier,
	User
} from '#lib/api/client.js';

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
	page.on('request', (request) => {
		const path = new URL(request.url()).pathname;
		if (request.method() === 'GET' && /^\/api\/v1\/me\/(tiers|posts|analytics)/.test(path)) {
			expect(request.headers()['x-expected-user-id']).toBe(user.id);
		}
	});
	await page.route('**/api/v1/me', (route) => route.fulfill({ json: { user } }));
	await page.route('**/api/v1/auth/csrf', (route) =>
		route.fulfill({
			json: { token: csrf },
			headers: { 'Set-Cookie': `csrf=${csrf}; HttpOnly; Path=/; SameSite=Lax` }
		})
	);
}

function expectStudioMutation(request: Request) {
	expect(request.headers()['x-expected-user-id']).toBe(user.id);
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
			expectStudioMutation(request);
			savedProfile = { ...profile, ...request.postDataJSON() };
		}
		await route.fulfill({ json: { ownerId: user.id, profile: savedProfile } });
	});
	await page.route('**/api/v1/me/tiers', async (route) => {
		const request = route.request();
		if (request.method() === 'POST') {
			expectStudioMutation(request);
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
			expectStudioMutation(request);
			expect(request.postDataJSON().minimumTierId).toBeNull();
			savedPost = { ...post('7'), ...request.postDataJSON() };
			await route.fulfill({ status: 201, json: { post: savedPost } });
			return;
		}
		await route.fulfill({
			json: {
				posts: savedPost ? [savedPost] : [],
				hasMore: false,
				hasPublishedPost: savedPost?.publishedAt != null
			}
		});
	});
	await page.route('**/api/v1/me/posts/7/publish', async (route) => {
		expectStudioMutation(route.request());
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
	await page.getByRole('button', { name: 'Новый уровень' }).click();
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

test('Post editor previews Markdown and unlocks attachments after saving a draft', async ({
	page
}) => {
	await mockAccount(page);
	await page.route('**/api/v1/me/profile', (route) =>
		route.fulfill({ json: { ownerId: user.id, profile } })
	);
	await page.route('**/api/v1/me/tiers', (route) => route.fulfill({ json: { tiers: [tier] } }));

	let savedPost = post('7');
	let publishAttempts = 0;
	await page.route('**/api/v1/me/posts', async (route) => {
		expectStudioMutation(route.request());
		savedPost = { ...savedPost, ...route.request().postDataJSON() };
		await route.fulfill({ status: 201, json: { post: savedPost } });
	});
	await page.route('**/api/v1/me/posts/7', async (route) => {
		if (route.request().method() === 'PUT') {
			expectStudioMutation(route.request());
			savedPost = { ...savedPost, ...route.request().postDataJSON() };
		}
		await route.fulfill({ json: { post: savedPost } });
	});
	await page.route('**/api/v1/me/posts/7/publish', async (route) => {
		expectStudioMutation(route.request());
		if (++publishAttempts === 1) {
			await route.fulfill({ status: 503, json: { error: { code: 'unavailable' } } });
			return;
		}
		savedPost = { ...savedPost, publishedAt };
		await route.fulfill({ json: { post: savedPost } });
	});
	await page.route('**/api/v1/me/posts/7/unpublish', async (route) => {
		expectStudioMutation(route.request());
		savedPost = { ...savedPost, publishedAt: null };
		await route.fulfill({ json: { post: savedPost } });
	});

	const file: PostFile = {
		id: '12',
		kind: 'image',
		filename: 'эскиз.png',
		contentType: 'image/png',
		sizeBytes: 1024,
		url: '/api/v1/media/12/original',
		thumbnailUrl: '/api/v1/media/12/thumbnail'
	};
	let allowUpload: () => void;
	const waitForUpload = new Promise<void>((resolve) => (allowUpload = resolve));
	await page.route('**/api/v1/me/posts/7/media?*', async (route) => {
		const request = route.request();
		expectStudioMutation(request);
		expect(request.headers()['content-type']).toBe('application/octet-stream');
		expect(request.postDataBuffer()?.length).toBe(file.sizeBytes);
		expect(Object.fromEntries(new URL(request.url()).searchParams)).toEqual({
			kind: file.kind,
			filename: file.filename
		});
		await waitForUpload;
		savedPost = { ...savedPost, media: [file] };
		await route.fulfill({ status: 201, json: { media: file } });
	});
	await page.route('**/api/v1/me/posts/7/media/12', async (route) => {
		expectStudioMutation(route.request());
		expect(route.request().method()).toBe('DELETE');
		savedPost = { ...savedPost, media: [] };
		await route.fulfill({ json: { message: 'Файл удалён из публикации.' } });
	});
	await page.route('**/api/v1/media/12/thumbnail', (route) =>
		route.fulfill({
			contentType: 'image/png',
			body: Buffer.from(
				'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jD1sAAAAASUVORK5CYII=',
				'base64'
			)
		})
	);

	await page.goto('/studio/posts/new');
	const files = page.getByRole('region', { name: 'Файлы' });
	const picker = files.getByLabel('Выбрать файлы');
	await expect(page.getByRole('link', { name: 'Назад' })).toHaveAttribute('href', '/studio/posts');
	await expect(picker).toBeDisabled();
	await expect(files.getByLabel('Тип файла')).toBeDisabled();
	await expect(page.getByText('Виден всем', { exact: true })).toBeVisible();
	await page.getByLabel('Заголовок').fill('Новый эскиз');
	await page.getByLabel('Тизер').fill('Заметки из мастерской');
	await page.getByLabel('Доступ').selectOption({ label: tier.name });
	const markdown = '## Эскиз\n\n**Подпись** <mark>текст</mark>';
	await page.getByLabel('Текст поста').fill(markdown);
	await page.getByRole('button', { name: 'Предпросмотр' }).click();
	await expect(page.getByRole('heading', { name: 'Эскиз' })).toBeVisible();
	await expect(page.locator('#post-preview strong')).toHaveText('Подпись');
	await expect(page.locator('#post-preview mark')).toHaveCount(0);
	await page.getByRole('button', { name: 'Редактор' }).click();
	await expect(page.getByLabel('Текст поста')).toHaveValue(markdown);

	await page.getByRole('button', { name: 'Опубликовать' }).click();
	await expect(page).toHaveURL('/studio/posts/7/edit');
	await expect(page.getByRole('alert')).toContainText('Черновик сохранён.');
	await expect(page.getByText('Черновик', { exact: true })).toBeVisible();
	await expect(picker).toBeEnabled();
	await expect(page.getByLabel('Доступ')).toHaveValue(tier.id);

	await files.getByLabel('Тип файла').selectOption('image');
	try {
		await picker.setInputFiles({
			name: file.filename,
			mimeType: file.contentType,
			buffer: Buffer.alloc(file.sizeBytes)
		});
		await expect(
			files.getByRole('progressbar', { name: `Загрузка ${file.filename}` })
		).toBeVisible();
		await expect(page.getByRole('button', { name: 'Сохранить' })).toBeDisabled();
	} finally {
		allowUpload!();
	}
	await expect(files.getByText('Изображение · 1 КБ')).toBeVisible();
	await expect(files.locator('img')).toHaveAttribute('src', file.thumbnailUrl!);
	await files.getByRole('button', { name: `Открепить ${file.filename}` }).click();
	await expect(files.getByText(file.filename)).toHaveCount(0);

	const dataTransfer = await page.evaluateHandle(({ filename, sizeBytes, contentType }) => {
		const transfer = new DataTransfer();
		transfer.items.add(new File([new Uint8Array(sizeBytes)], filename, { type: contentType }));
		return transfer;
	}, file);
	await files.getByText('или перетащить сюда').dispatchEvent('drop', { dataTransfer });
	await expect(files.getByText(file.filename)).toBeVisible();
	await expect(files.getByRole('button', { name: `Открепить ${file.filename}` })).toBeEnabled();
	await dataTransfer.dispose();

	await page.getByRole('button', { name: 'Опубликовать' }).click();
	await expect(page.getByText('Опубликован', { exact: true })).toBeVisible();
	await page.getByRole('button', { name: 'Снять', exact: true }).click();
	await expect(page.getByText('Черновик', { exact: true })).toBeVisible();
});

test('Studio confines tier moves to their section and reloads the order after a conflict', async ({
	page
}) => {
	await mockAccount(page);
	await page.route('**/api/v1/me/profile', (route) =>
		route.fulfill({ json: { ownerId: user.id, profile } })
	);
	let tiers: Tier[] = [
		{ ...tier, id: '1', name: 'Первая', position: 6 },
		{ ...tier, id: '2', name: 'Вторая', position: 5 },
		{ ...tier, id: '3', name: 'Используемая', position: 4, used: true },
		{ ...tier, id: '4', name: 'Четвёртая', position: 3 },
		{ ...tier, id: '5', name: 'Архивная', position: 2, used: true, archived: true },
		{ ...tier, id: '6', name: 'Последняя', position: 1 }
	];
	let reorderRequests = 0;
	await page.route('**/api/v1/me/tiers', (route) => route.fulfill({ json: { tiers } }));
	await page.route('**/api/v1/me/tiers/order', async (route) => {
		expectStudioMutation(route.request());
		reorderRequests++;
		if (reorderRequests === 1) {
			expect(route.request().postDataJSON()).toEqual({ ids: ['2', '1', '3', '4', '6'] });
			tiers = [{ ...tier, id: '7', name: 'Добавленная', position: 7 }, ...tiers];
			await route.fulfill({ status: 409, json: { error: { code: 'tier_order_locked' } } });
			return;
		}
		expect(route.request().postDataJSON()).toEqual({ ids: ['7', '2', '1', '3', '4', '6'] });
		const positions = { '1': 5, '2': 6 } as Record<string, number | undefined>;
		tiers = tiers.map((current) => ({
			...current,
			position: positions[current.id] ?? current.position
		}));
		await route.fulfill({ json: { tiers } });
	});

	await page.goto('/studio/tiers');
	const active = page.getByRole('list', { name: 'Активные уровни' });
	const row = (name: string) =>
		active.getByRole('listitem').filter({
			has: page.getByRole('heading', { name, exact: true })
		});
	const saveOrder = page.getByRole('button', { name: 'Сохранить порядок' });
	await expect(saveOrder).toBeDisabled();
	await expect(row('Вторая').getByRole('button', { name: 'Ниже' })).toBeDisabled();
	await expect(row('Четвёртая').getByRole('button', { name: 'Ниже' })).toBeDisabled();
	await expect(row('Последняя').getByRole('button', { name: 'Выше' })).toBeDisabled();
	await row('Первая').getByRole('button', { name: 'Ниже' }).press('Enter');
	await expect(active.getByRole('heading').first()).toHaveText('Вторая');
	await expect(saveOrder).toBeEnabled();
	await row('Первая').getByRole('button', { name: 'Выше' }).press('Enter');
	await expect(saveOrder).toBeDisabled();
	await row('Первая').locator('[draggable=true]').dragTo(row('Используемая'));
	await expect(saveOrder).toBeDisabled();
	await row('Первая').locator('[draggable=true]').dragTo(row('Вторая'));
	await expect(active.getByRole('heading').first()).toHaveText('Вторая');
	expect(reorderRequests).toBe(0);
	await saveOrder.click();
	await expect(page.getByText('Порядок изменился, обновляем список')).toBeVisible();
	await expect(active.getByRole('heading')).toHaveText([
		'Добавленная',
		'Первая',
		'Вторая',
		'Используемая',
		'Четвёртая',
		'Последняя'
	]);
	await expect(saveOrder).toBeDisabled();
	await row('Вторая').getByRole('button', { name: 'Выше' }).click();
	await saveOrder.click();
	await expect(page.getByText('Порядок уровней сохранён.')).toBeVisible();
	await expect(active.getByRole('heading').nth(1)).toHaveText('Вторая');
	await expect(saveOrder).toBeDisabled();
	await page.getByText('Архив', { exact: true }).first().click();
	const archive = page.getByRole('list', { name: 'Архивные уровни' });
	await expect(archive.getByText('Используется')).toBeVisible();
	await expect(archive).toContainText('150,5');
});

test('Studio explains tier prices and reloads a deletion that becomes an archive', async ({
	page
}) => {
	await mockAccount(page);
	await page.route('**/api/v1/me/profile', (route) =>
		route.fulfill({ json: { ownerId: user.id, profile } })
	);
	let tiers = [
		{ ...tier, name: 'Эскизы', position: 3, priceKopecks: 99 },
		{ ...tier, id: '2', name: 'Мастерская', position: 2, priceKopecks: 100, used: true },
		{ ...tier, id: '3', name: 'Чертежи', position: 1 }
	];
	await page.route('**/api/v1/me/tiers', (route) => route.fulfill({ json: { tiers } }));
	await page.route('**/api/v1/me/tiers/1', async (route) => {
		expectStudioMutation(route.request());
		expect(route.request().method()).toBe('DELETE');
		tiers = tiers.map((current) =>
			current.id === '1' ? { ...current, used: true, archived: true } : current
		);
		await route.fulfill({ json: { message: 'Уровень убран с публичной страницы.' } });
	});
	await page.route('**/api/v1/me/tiers/2/archive', async (route) => {
		expectStudioMutation(route.request());
		expect(route.request().method()).toBe('POST');
		tiers = tiers.map((current) => (current.id === '2' ? { ...current, archived: true } : current));
		await route.fulfill({ json: { message: 'Уровень убран с публичной страницы.' } });
	});

	await page.goto('/studio/tiers');
	const active = page.getByRole('list', { name: 'Активные уровни' });
	const sketches = active.getByRole('listitem').filter({ hasText: 'Эскизы' });
	const workshop = active.getByRole('listitem').filter({ hasText: 'Мастерская' });
	const modal = page.getByRole('dialog');
	const paymentWarning = 'Этот уровень нельзя оформить как подписку, минимальный платёж 1 ₽';
	await expect(sketches).toContainText('Дороже по порядку, но дешевле по цене');
	await expect(sketches).toContainText(paymentWarning);
	await expect(workshop).not.toContainText(paymentWarning);
	await workshop.getByRole('button', { name: 'Изменить' }).click();
	await expect(modal.getByLabel('Стоимость в месяц, ₽')).toHaveValue('1');
	await expect(modal).toContainText(
		'Новая цена применится к новым подписчикам, действующие продлеваются по прежней'
	);
	await modal.getByRole('button', { name: 'Отмена' }).click();
	await expect(workshop.getByRole('button', { name: 'Изменить' })).toBeFocused();

	await sketches.getByRole('button', { name: 'Удалить' }).click();
	await expect(modal).toContainText('«Эскизы»');
	await modal.getByRole('button', { name: 'Удалить безвозвратно' }).click();
	await expect(sketches).toHaveCount(0);
	await expect(page.getByText('Уровень перемещён в архив.')).toBeVisible();
	await workshop.getByRole('button', { name: 'Архивировать' }).click();
	await expect(modal).toContainText('Доступ у действующих подписчиков сохранится.');
	await modal.getByRole('button', { name: 'Архивировать' }).click();
	await expect(workshop).toHaveCount(0);
	await page.getByText('Архив', { exact: true }).first().click();
	await expect(page.getByRole('list', { name: 'Архивные уровни' }).getByRole('heading')).toHaveText(
		['Эскизы', 'Мастерская']
	);
});

test('Studio detects older publications in one request and retries analytics on mobile', async ({
	page
}) => {
	await page.setViewportSize({ width: 360, height: 800 });
	await mockAccount(page);
	const postRequests: { offset: string | null; ownerId: string | undefined }[] = [];
	page.on('request', (request) => {
		const url = new URL(request.url());
		if (request.method() === 'GET' && url.pathname === '/api/v1/me/posts') {
			postRequests.push({
				offset: url.searchParams.get('offset'),
				ownerId: request.headers()['x-expected-user-id']
			});
		}
	});
	let failAnalytics = true;
	await page.route('**/api/v1/me/profile', (route) =>
		route.fulfill({ json: { ownerId: user.id, profile } })
	);
	await page.route('**/api/v1/me/tiers', (route) => route.fulfill({ json: { tiers: [tier] } }));
	await page.route('**/api/v1/me/posts', (route) =>
		route.fulfill({
			json: {
				posts: Array.from({ length: 24 }, (_, index) => post(String(25 - index))),
				hasMore: true,
				hasPublishedPost: true
			}
		})
	);
	await page.route(/\/api\/v1\/me\/analytics(?:\?|$)/, (route) =>
		failAnalytics ? route.fulfill({ status: 503, json: {} }) : route.fulfill({ json: analytics })
	);

	await page.goto('/studio');
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
	expect(postRequests).toEqual([{ offset: null, ownerId: user.id }]);

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
	await expect(page.getByRole('main').getByText('Тестовый режим')).toBeVisible();
	await expect(nav).toBeHidden();
});

test('Studio links to the newest draft when the author has no publications', async ({ page }) => {
	await mockAccount(page);
	await page.route('**/api/v1/me/profile', (route) =>
		route.fulfill({ json: { ownerId: user.id, profile } })
	);
	await page.route('**/api/v1/me/tiers', (route) => route.fulfill({ json: { tiers: [tier] } }));
	await page.route('**/api/v1/me/posts', (route) =>
		route.fulfill({
			json: { posts: [post('2'), post('1')], hasMore: false, hasPublishedPost: false }
		})
	);
	await page.route('**/api/v1/me/analytics', (route) => route.fulfill({ json: analytics }));

	await page.goto('/studio');
	const setup = page.getByRole('region', { name: 'Настройте студию' });
	await expect(setup).toContainText('2 из 3');
	await expect(setup.getByRole('link', { name: 'Открыть черновик' })).toHaveAttribute(
		'href',
		'/studio/posts/2/edit'
	);
});

test('Profile previews draft text and applies each image crop independently', async ({ page }) => {
	await mockAccount(page);
	let savedProfile = { ...profile };
	let rejectCover = true;
	let rejectRemoval = true;
	let uploadCount = 0;
	await page.route('**/api/v1/me/profile', async (route) => {
		if (route.request().method() === 'PUT') {
			expectStudioMutation(route.request());
			await route.fulfill({ status: 409, json: { error: { code: 'nickname_exists' } } });
		} else {
			await route.fulfill({ json: { ownerId: user.id, profile: savedProfile } });
		}
	});
	await page.route('**/api/v1/me/profile/images/*', async (route) => {
		const request = route.request();
		expectStudioMutation(request);
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
		await route.fulfill({ json: { ownerId: user.id, profile: savedProfile } });
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

const secondUser: User = { ...user, id: '2', email: 'potter@example.com', nickname: 'potter' };
const firstProfile: Profile = {
	...profile,
	description: 'Заметки первой мастерской',
	avatar: {
		url: '/artwork/garden.webp',
		contentType: 'image/webp',
		sizeBytes: 100,
		width: 80,
		height: 80
	}
};
const secondProfile: Profile = {
	...profile,
	id: secondUser.id,
	nickname: secondUser.nickname,
	displayName: 'Гончарная студия',
	description: 'Заметки второй мастерской'
};

async function mockSharedAccounts(context: BrowserContext) {
	let current: User | null = user;
	await context.route('**/api/v1/me', (route) => route.fulfill({ json: { user: current } }));
	await context.route('**/api/v1/auth/csrf', (route) =>
		route.fulfill({
			json: { token: csrf },
			headers: { 'Set-Cookie': `csrf=${csrf}; HttpOnly; Path=/; SameSite=Lax` }
		})
	);
	await context.route('**/api/v1/auth/login', (route) => {
		current = secondUser;
		return route.fulfill({ json: { user: current } });
	});
	await context.route('**/api/v1/auth/logout', (route) => {
		current = null;
		return route.fulfill({ json: { message: 'ok' } });
	});
	return () => current;
}

async function signInAsSecondUser(page: Page) {
	await page.getByRole('button', { name: 'Выйти', exact: true }).click();
	await page.getByLabel('Email', { exact: true }).fill(secondUser.email);
	await page.getByLabel('Пароль', { exact: true }).fill('test-password');
	await page.getByRole('main').getByRole('button', { name: 'Войти', exact: true }).click();
	await expect(page.getByText('@potter', { exact: true })).toBeVisible();
}

for (const scenario of ['second creator', 'second account without a profile', 'guest'] as const) {
	test(`Studio replaces the previous editor after session refresh as ${scenario}`, async ({
		page,
		context
	}) => {
		const currentAccount = await mockSharedAccounts(context);
		const enteredReload = Promise.withResolvers<void>();
		const reload = Promise.withResolvers<void>();
		await context.route('**/api/v1/me/profile', async (route) => {
			const current = currentAccount();
			if (current?.id === user.id) {
				await route.fulfill({ json: { ownerId: user.id, profile: firstProfile } });
				return;
			}
			enteredReload.resolve();
			await reload.promise;
			if (current) {
				await route.fulfill({
					json: {
						ownerId: current.id,
						profile: scenario === 'second creator' ? secondProfile : null
					}
				});
			} else {
				await route.fulfill({ status: 401, json: { error: { code: 'unauthorized' } } });
			}
		});

		const otherTab = await context.newPage();
		await otherTab.goto('/login');
		await page.goto('/studio/profile');
		const main = page.getByRole('main');
		const name = main.getByLabel('Отображаемое имя');
		const description = main.getByLabel('Описание');
		const deleteAvatar = main.getByRole('button', { name: 'Удалить аватар' });
		await expect(name).toHaveValue(firstProfile.displayName);
		await expect(description).toHaveValue(firstProfile.description);
		await expect(deleteAvatar).toBeVisible();
		await description.fill('Несохранённый черновик первого автора');

		const enteredRefresh = Promise.withResolvers<void>();
		const refresh = Promise.withResolvers<void>();
		await page.route(
			'**/api/v1/me',
			async (route) => {
				enteredRefresh.resolve();
				await refresh.promise;
				await route.fulfill({ json: { user: currentAccount() } });
			},
			{ times: 1 }
		);
		const sameAccountRefresh = page.waitForResponse('**/api/v1/me');
		try {
			await page.evaluate(() => window.dispatchEvent(new Event('focus')));
			await enteredRefresh.promise;
			await expect(description).toHaveValue('Несохранённый черновик первого автора');
		} finally {
			refresh.resolve();
		}
		await sameAccountRefresh;
		await expect(description).toHaveValue('Несохранённый черновик первого автора');

		if (scenario === 'guest') {
			await otherTab.getByRole('button', { name: 'Выйти', exact: true }).click();
			await expect(otherTab.getByRole('banner').getByText('@artist', { exact: true })).toHaveCount(
				0
			);
		} else {
			await signInAsSecondUser(otherTab);
		}

		try {
			if (scenario === 'second account without a profile') {
				await page.evaluate(() =>
					window.dispatchEvent(new PageTransitionEvent('pageshow', { persisted: true }))
				);
			} else {
				await page.evaluate(() => window.dispatchEvent(new Event('focus')));
			}
			await enteredReload.promise;
			await expect(name).toHaveCount(0);
			await expect(description).toHaveCount(0);
			await expect(deleteAvatar).toHaveCount(0);
			await expect(main.getByRole('img', { name: 'Аватар профиля' })).toHaveCount(0);
			if (scenario === 'guest') {
				await expect(main.getByRole('link', { name: 'Войти', exact: true })).toBeVisible();
			} else {
				await expect(main.getByRole('status')).toHaveText('Загружаем…');
			}
		} finally {
			reload.resolve();
		}

		if (scenario === 'guest') {
			await expect(main.getByRole('link', { name: 'Войти', exact: true })).toBeVisible();
		} else if (scenario === 'second account without a profile') {
			await expect(main.getByRole('heading', { name: 'Создать профиль' })).toBeVisible();
			await expect(name).toHaveValue('');
			await expect(description).toHaveValue('');
			await expect(main.getByLabel('Ник', { exact: true })).toHaveValue(secondUser.nickname);
		} else {
			await expect(name).toHaveValue(secondProfile.displayName);
			await expect(description).toHaveValue(secondProfile.description);
		}
		if (scenario !== 'guest') await expect(deleteAvatar).toBeDisabled();
	});
}

test('An image mutation delayed by CSRF retains its original account and is never replayed', async ({
	page,
	context
}) => {
	const currentAccount = await mockSharedAccounts(context);
	const enteredCSRF = Promise.withResolvers<void>();
	const releaseCSRF = Promise.withResolvers<void>();
	await context.route('**/api/v1/auth/csrf', async (route) => {
		if (route.request().frame().page() !== page) return route.fallback();
		enteredCSRF.resolve();
		await releaseCSRF.promise;
		await route.fallback();
	});
	await context.route('**/api/v1/me/profile', (route) => {
		const current = currentAccount()!;
		return route.fulfill({
			json: { ownerId: current.id, profile: current.id === user.id ? firstProfile : secondProfile }
		});
	});
	const mutationOwners: string[] = [];
	await context.route('**/api/v1/me/profile/images/avatar', (route) => {
		const request = route.request();
		expectStudioMutation(request);
		expect(request.method()).toBe('DELETE');
		mutationOwners.push(request.headers()['x-expected-user-id']);
		expect(currentAccount()?.id).toBe(secondUser.id);
		return route.fulfill({ status: 412, json: { error: { code: 'account_changed' } } });
	});

	const otherTab = await context.newPage();
	await otherTab.goto('/login');
	await page.goto('/studio/profile');
	await page.getByRole('button', { name: 'Удалить аватар' }).click();
	await enteredCSRF.promise;
	try {
		await signInAsSecondUser(otherTab);
	} finally {
		releaseCSRF.resolve();
	}
	await expect(page.getByLabel('Отображаемое имя')).toHaveValue(secondProfile.displayName);
	await expect(page.getByRole('button', { name: 'Удалить аватар' })).toBeDisabled();
	expect(mutationOwners).toEqual([user.id]);
});

test('A private read can restart session verification during an account reload', async ({
	page,
	context
}) => {
	const currentAccount = await mockSharedAccounts(context);
	const enteredTiers = Promise.withResolvers<void>();
	const releaseTiers = Promise.withResolvers<void>();
	await context.route('**/api/v1/me/profile', (route) => {
		const current = currentAccount();
		return current
			? route.fulfill({
					json: {
						ownerId: current.id,
						profile: current.id === user.id ? firstProfile : secondProfile
					}
				})
			: route.fulfill({ status: 401, json: { error: { code: 'unauthorized' } } });
	});
	await context.route('**/api/v1/me/tiers', async (route) => {
		const ownerId = route.request().headers()['x-expected-user-id'];
		if (ownerId === secondUser.id) {
			enteredTiers.resolve();
			await releaseTiers.promise;
		}
		const current = currentAccount();
		if (!current) {
			await route.fulfill({ status: 401, json: { error: { code: 'unauthorized' } } });
		} else {
			expect(ownerId).toBe(current.id);
			await route.fulfill({ json: { tiers: [tier] } });
		}
	});

	const otherTab = await context.newPage();
	await otherTab.goto('/login');
	await page.goto('/studio/tiers');
	const main = page.getByRole('main');
	await expect(main.getByRole('heading', { name: tier.name })).toBeVisible();

	try {
		await signInAsSecondUser(otherTab);
		await page.evaluate(() => window.dispatchEvent(new Event('focus')));
		await enteredTiers.promise;
		await expect(main.getByRole('status')).toHaveText('Загружаем…');
		await otherTab.getByRole('button', { name: 'Выйти', exact: true }).click();
		await expect(otherTab.getByRole('banner').getByText('@potter', { exact: true })).toHaveCount(0);
	} finally {
		releaseTiers.resolve();
	}
	await expect(main.getByRole('link', { name: 'Войти', exact: true })).toBeVisible();
	await expect(main.getByRole('heading', { name: tier.name })).toHaveCount(0);
});

test('Studio keeps write controls hidden until the bootstrap response identifies its owner', async ({
	page
}) => {
	await mockAccount(page);
	let hasOwner = false;
	await page.route('**/api/v1/me/profile', (route) =>
		route.fulfill({ json: hasOwner ? { ownerId: user.id, profile } : { profile } })
	);

	await page.goto('/studio/profile');
	const main = page.getByRole('main');
	const retry = main.getByRole('button', { name: 'Повторить' });
	await expect(retry).toBeVisible();
	await expect(main.getByRole('button', { name: 'Сохранить', exact: true })).toHaveCount(0);
	hasOwner = true;
	await retry.click();
	await expect(main.getByLabel('Отображаемое имя')).toHaveValue(profile.displayName);
});

test('Studio reuses the initial overview read after verifying its matching account', async ({
	page
}) => {
	const releaseSession = Promise.withResolvers<void>();
	const sessionStarted = Promise.withResolvers<void>();
	let firstPageRequests = 0;
	await page.route('**/api/v1/me', async (route) => {
		sessionStarted.resolve();
		await releaseSession.promise;
		await route.fulfill({ json: { user } });
	});
	await page.route('**/api/v1/me/profile', (route) =>
		route.fulfill({ json: { ownerId: user.id, profile } })
	);
	await page.route('**/api/v1/me/tiers', (route) => route.fulfill({ json: { tiers: [tier] } }));
	await page.route('**/api/v1/me/analytics', (route) => route.fulfill({ json: analytics }));
	await page.route('**/api/v1/me/posts', (route) => {
		firstPageRequests++;
		expect(route.request().headers()['x-expected-user-id']).toBe(user.id);
		return route.fulfill({
			json: { posts: [post('1', publishedAt)], hasMore: false, hasPublishedPost: true }
		});
	});

	const firstPage = page.waitForResponse('**/api/v1/me/posts');
	try {
		await page.goto('/studio');
		await Promise.all([sessionStarted.promise, firstPage]);
		await expect(page.getByRole('main').getByRole('status')).toHaveText('Загружаем…');
		expect(firstPageRequests).toBe(1);
	} finally {
		releaseSession.resolve();
	}

	await expect(page.getByRole('link', { name: /Работа 1/ })).toBeVisible();
	expect(firstPageRequests).toBe(1);
});

for (const initialUser of [secondUser, null] as const) {
	test(`Studio discards initial profile data after verifying ${initialUser?.nickname ?? 'a guest'}`, async ({
		page
	}) => {
		const reloadStarted = Promise.withResolvers<void>();
		const releaseReload = Promise.withResolvers<void>();
		let reads = 0;
		await page.route('**/api/v1/me', (route) => route.fulfill({ json: { user: initialUser } }));
		await page.route('**/api/v1/me/profile', async (route) => {
			if (++reads === 1) {
				return route.fulfill({ json: { ownerId: user.id, profile: firstProfile } });
			}
			reloadStarted.resolve();
			await releaseReload.promise;
			if (initialUser) {
				return route.fulfill({ json: { ownerId: initialUser.id, profile: secondProfile } });
			}
			return route.fulfill({ status: 401, json: { error: { code: 'unauthorized' } } });
		});

		await page.goto('/studio/profile');
		const main = page.getByRole('main');
		const editor = main.getByLabel('Отображаемое имя');
		try {
			await reloadStarted.promise;
			await expect(editor).toHaveCount(0);
			if (initialUser) {
				await expect(main.getByRole('status')).toHaveText('Загружаем…');
			} else {
				await expect(main.getByRole('link', { name: 'Войти', exact: true })).toBeVisible();
			}
		} finally {
			releaseReload.resolve();
		}
		if (initialUser) await expect(editor).toHaveValue(secondProfile.displayName);
		else await expect(editor).toHaveCount(0);
	});
}

test('Studio keeps the initial editor hidden when session verification fails', async ({ page }) => {
	let verifySession = false;
	await page.route('**/api/v1/me', (route) =>
		verifySession ? route.fulfill({ json: { user } }) : route.fulfill({ status: 503, json: {} })
	);
	await page.route('**/api/v1/me/profile', (route) =>
		route.fulfill({ json: { ownerId: user.id, profile: firstProfile } })
	);

	await page.goto('/studio/profile');
	const main = page.getByRole('main');
	const editor = main.getByLabel('Отображаемое имя');
	await expect(main.getByRole('status')).toHaveText('Не удалось загрузить студию.');
	await expect(editor).toHaveCount(0);
	verifySession = true;
	await main.getByRole('button', { name: 'Повторить' }).click();
	await expect(editor).toHaveValue(firstProfile.displayName);
});

for (const refreshFails of [false, true]) {
	test(`Studio keeps the current post tab and returns from an emptied page after publishing${refreshFails ? ' and retrying the list' : ''}`, async ({
		page
	}) => {
		await mockAccount(page);
		await page.route('**/api/v1/me/profile', (route) =>
			route.fulfill({ json: { ownerId: user.id, profile } })
		);
		const updatedAt = '2026-11-19T12:00:00Z';
		const posts: OwnPostSummary[] = Array.from({ length: 25 }, (_, index) => ({
			...post(String(25 - index)),
			updatedAt,
			minimumTierName: null
		}));
		const lastDraft = posts[24];
		lastDraft.minimumTierId = tier.id;
		lastDraft.minimumTierName = tier.name;
		let failNextRefresh = refreshFails;
		await page.route(/\/api\/v1\/me\/posts\?/, (route) => {
			const params = new URL(route.request().url()).searchParams;
			const status = params.get('status');
			expect(['draft', 'published']).toContain(status);
			const offset = Number(params.get('offset') ?? 0);
			if (failNextRefresh && status === 'draft' && offset === 24 && lastDraft.publishedAt) {
				failNextRefresh = false;
				return route.fulfill({ status: 503, json: {} });
			}
			const filtered = posts.filter((entry) => !!entry.publishedAt === (status === 'published'));
			return route.fulfill({
				json: {
					posts: filtered.slice(offset, offset + 24),
					hasMore: filtered.length > offset + 24,
					hasPublishedPost: lastDraft.publishedAt !== null
				}
			});
		});
		await page.route('**/api/v1/me/posts/1/publish', (route) => {
			expectStudioMutation(route.request());
			lastDraft.publishedAt = publishedAt;
			return route.fulfill({ json: { post: lastDraft } });
		});
		await page.route('**/api/v1/me/posts/1/unpublish', (route) => {
			expectStudioMutation(route.request());
			lastDraft.publishedAt = null;
			return route.fulfill({ json: { post: lastDraft } });
		});

		await page.goto('/studio/posts');
		const main = page.getByRole('main');
		await expect(main.getByRole('link', { name: 'Черновики' })).toHaveAttribute(
			'aria-current',
			'page'
		);
		await expect(main.getByText('Работа 25', { exact: true })).toBeVisible();
		await expect(main.getByText('Работа 1', { exact: true })).toHaveCount(0);
		await expect(main.getByText('Для всех').first()).toBeVisible();
		await expect(main.getByRole('link', { name: 'Новый пост' })).toHaveAttribute(
			'href',
			'/studio/posts/new'
		);
		await main.getByRole('link', { name: 'Следующая' }).click();
		await expect(page).toHaveURL('/studio/posts?status=draft&page=2');
		await page.reload();
		await expect(main.getByText('Работа 1', { exact: true })).toBeVisible();
		await expect(main.getByText(tier.name, { exact: true })).toBeVisible();
		await expect(main.locator('time')).toHaveAttribute('datetime', updatedAt);
		await expect(main.getByRole('link', { name: 'Редактировать' })).toHaveAttribute(
			'href',
			'/studio/posts/1/edit'
		);
		await main.getByRole('button', { name: 'Опубликовать' }).click();
		if (refreshFails) await main.getByRole('button', { name: 'Повторить' }).click();
		await expect(page).toHaveURL('/studio/posts?status=draft&page=1');
		await expect(main.getByText('Пост опубликован.', { exact: true })).toBeVisible();
		await expect(main.getByText('Работа 1', { exact: true })).toHaveCount(0);
		await expect(main.getByRole('link', { name: 'Следующая' })).toHaveCount(0);

		await main.getByRole('link', { name: 'Опубликованные' }).click();
		await expect(page).toHaveURL('/studio/posts?status=published&page=1');
		await expect(main.getByText('Работа 1', { exact: true })).toBeVisible();
		await expect(main.locator('time')).toHaveAttribute('datetime', publishedAt);
		await main.getByRole('button', { name: 'Снять', exact: true }).click();
		await expect(page).toHaveURL('/studio/posts?status=published&page=1');
		await expect(main.getByText('Пост снят с публикации.', { exact: true })).toBeVisible();
		await expect(main.getByText(/Опубликованных постов пока нет/)).toBeVisible();
	});
}

test('Studio retries loading posts and keeps a row when deletion fails', async ({ page }) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await mockAccount(page);
	await page.route('**/api/v1/me/profile', (route) =>
		route.fulfill({ json: { ownerId: user.id, profile } })
	);
	const draft: OwnPostSummary = { ...post('7'), updatedAt: publishedAt, minimumTierName: null };
	let failLoad = true;
	let failDelete = true;
	let deleted = false;
	await page.route(/\/api\/v1\/me\/posts\?/, (route) =>
		failLoad
			? route.fulfill({ status: 503, json: {} })
			: route.fulfill({
					json: { posts: deleted ? [] : [draft], hasMore: false, hasPublishedPost: false }
				})
	);
	await page.route('**/api/v1/me/tiers', (route) => route.fulfill({ json: { tiers: [] } }));
	await page.route('**/api/v1/me/posts/7', (route) => {
		if (route.request().method() === 'GET') return route.fulfill({ json: { post: draft } });
		expect(route.request().method()).toBe('DELETE');
		expectStudioMutation(route.request());
		if (failDelete) return route.fulfill({ status: 503, json: {} });
		deleted = true;
		return route.fulfill({ json: { message: 'Публикация удалена.' } });
	});

	await page.goto('/studio/posts');
	const main = page.getByRole('main');
	const retry = main.getByRole('button', { name: 'Повторить' });
	await expect(retry).toBeVisible();
	failLoad = false;
	await retry.click();
	const remove = main.getByRole('button', { name: 'Удалить', exact: true });
	await remove.click();
	const dialog = page.getByRole('dialog');
	await expect(dialog).toContainText(draft.title);
	await expect(dialog.getByRole('button', { name: 'Отмена' })).toBeFocused();
	await page.keyboard.press('Escape');
	await expect(dialog).toBeHidden();
	await expect(remove).toBeFocused();
	await remove.click();
	await dialog.getByRole('button', { name: 'Удалить', exact: true }).click();
	await expect(main.getByRole('alert')).toBeVisible();
	await expect(main.getByText(draft.title, { exact: true })).toBeVisible();
	failDelete = false;
	await remove.click();
	await dialog.getByRole('button', { name: 'Удалить', exact: true }).click();
	await expect(main.getByText('Черновиков пока нет')).toBeVisible();
});
