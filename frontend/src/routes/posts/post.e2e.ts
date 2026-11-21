import { expect, test, type Page } from '@playwright/test';
import type { Comment, Post, PostFile, Profile, User } from '#lib/api/client.js';

const profile: Profile = {
	id: '42',
	nickname: 'garden',
	displayName: 'Тихий сад',
	description: '',
	avatar: null,
	cover: null
};
const reader: User = {
	id: '13',
	nickname: 'reader',
	email: 'reader@example.com',
	emailVerified: true,
	isAdmin: false
};
const publishedAt = '2026-11-15T10:00:00Z';
const files: PostFile[] = [
	{ kind: 'image', filename: 'Эскиз.webp', contentType: 'image/webp' },
	{ kind: 'audio', filename: 'Запись.mp3', contentType: 'audio/mpeg' },
	{ kind: 'video', filename: 'Процесс.mp4', contentType: 'video/mp4' },
	{ kind: 'attachment', filename: 'Материалы.zip', contentType: 'application/octet-stream' }
].map(({ kind, filename, contentType }, index) => ({
	id: String(index + 1),
	kind: kind as PostFile['kind'],
	filename,
	contentType,
	sizeBytes: 1024,
	url: `/api/v1/media/${index + 1}/original`,
	thumbnailUrl: kind === 'image' ? '/api/v1/media/1/thumbnail' : null
}));
const post: Post = {
	id: '5',
	authorNickname: profile.nickname,
	title: 'Зимние эскизы',
	teaser: 'Публичный анонс',
	publishedAt,
	minimumTierId: '2',
	minimumTierName: 'Оранжерея',
	canRead: true,
	bodyHtml: '<p>Секретные материалы <strong>из мастерской</strong>.</p>',
	media: files
};
const path = `/posts/${post.id}`;
const apiPath = `/api/v1${path}`;

async function mockReader(page: Page, user: User | null, publishedPost: Post) {
	await page.route('**/api/v1/me', (route) => route.fulfill({ json: { user } }));
	await page.route(`**${apiPath}`, (route) => route.fulfill({ json: { post: publishedPost } }));
	await page.route(`**/api/v1/authors/${profile.nickname}`, (route) =>
		route.fulfill({ json: { profile, tiers: [] } })
	);
}

test('locked post displays the tier and links to the author without exposing body or media', async ({
	page
}) => {
	await mockReader(page, reader, { ...post, canRead: false });
	await page.route('**/api/v1/me/subscriptions?*', (route) =>
		route.fulfill({ json: { subscriptions: [], hasMore: false, paymentsEnabled: true } })
	);
	await page.route(`**/api/v1/authors/${profile.nickname}/posts?*`, (route) =>
		route.fulfill({ json: { posts: [post], hasMore: false } })
	);
	const restrictedRequests: string[] = [];
	page.on('request', (request) => {
		if (/\/api\/v1\/(media\/|posts\/5\/discussion)/.test(request.url()))
			restrictedRequests.push(request.url());
	});
	await page.goto(path);
	await expect(page.getByRole('link', { name: profile.displayName })).toBeVisible();
	await expect(page.getByRole('heading', { name: 'Нужен уровень «Оранжерея»' })).toBeVisible();
	await expect(page.getByText(post.teaser)).toBeVisible();
	await expect(page.getByText('Секретные материалы', { exact: false })).toHaveCount(0);
	await expect(page.locator('main video, main audio, main [download], main img')).toHaveCount(0);
	await expect(page.getByLabel('Ваш комментарий')).toHaveCount(0);
	expect(restrictedRequests).toEqual([]);

	await page.getByRole('link', { name: 'Подписаться' }).click();
	await expect(page.getByRole('heading', { name: 'Уровни подписки' })).toBeFocused();
	await page.getByRole('link', { name: post.title }).click();
	await expect(page.getByRole('heading', { name: post.title })).toBeVisible();
});

test('guest can read a public post and can return to the discussion after login', async ({
	page
}) => {
	await mockReader(page, null, { ...post, canRead: false });
	await page.goto(path);
	const discussion = page.getByRole('region', { name: 'Комментарии' });
	await expect(discussion).toContainText('Войдите в аккаунт с доступом к публикации');
	await page.route(`**${apiPath}`, (route) =>
		route.fulfill({
			json: { post: { ...post, minimumTierId: null, minimumTierName: null, media: [] } }
		})
	);
	await page.route('**/api/v1/auth/csrf', (route) =>
		route.fulfill({ json: { token: 'test-csrf' } })
	);
	await page.route('**/api/v1/auth/login', (route) => route.fulfill({ json: { user: reader } }));
	await page.reload();
	await expect(page.getByText('Публично', { exact: true })).toBeVisible();
	await expect(page.locator('main article strong')).toHaveText('из мастерской');
	await expect(page.getByLabel('Ваш комментарий')).toHaveCount(0);
	await discussion.getByRole('link', { name: 'Войдите' }).click();
	await page.getByLabel('Email', { exact: true }).fill(reader.email);
	await page.getByLabel('Пароль', { exact: true }).fill('example-password');
	await page.getByRole('button', { name: 'Войти', exact: true }).click();
	await expect(page.getByRole('link', { name: 'Вернуться к посту' })).toHaveAttribute(
		'href',
		`${path}#comments`
	);
});

test('reader can view media, toggle a like, page comments and delete only their own', async ({
	page
}) => {
	await mockReader(page, reader, post);
	await page.route('**/api/v1/media/*/*', (route) => {
		if (route.request().url().endsWith('thumbnail'))
			return route.fulfill({ path: 'static/artwork/garden.webp' });
		return route.fulfill({ status: 204 });
	});
	await page.route('**/api/v1/auth/csrf', (route) =>
		route.fulfill({ json: { token: 'test-csrf' } })
	);
	const initial: Comment = {
		id: '10',
		postId: post.id,
		nickname: 'writer',
		body: 'Первый отклик',
		createdAt: publishedAt,
		own: false
	};
	const summary = { liked: false, likeCount: 9, commentCount: 2 };
	await page.route(`**${apiPath}/discussion*`, (route) => {
		const before = new URL(route.request().url()).searchParams.get('before');
		return route.fulfill({
			json: before
				? {
						summary: { ...summary, liked: true, likeCount: 10 },
						comments: [{ ...initial, id: '8', body: 'Более ранний отклик' }],
						nextCursor: null
					}
				: { summary, comments: [initial], nextCursor: '10' }
		});
	});
	let hasAccess = true;
	await page.route(`**${apiPath}/like`, (route) => {
		expect(route.request().headers()['x-csrf-token']).toBe('test-csrf');
		if (!hasAccess)
			return route.fulfill({ status: 403, json: { error: { code: 'post_access_required' } } });
		const liked = route.request().method() === 'PUT';
		return route.fulfill({ json: { summary: { ...summary, liked, likeCount: liked ? 10 : 9 } } });
	});
	const ownText = '<b>Мой комментарий</b>';
	await page.route(`**${apiPath}/comments`, (route) => {
		expect(route.request().postDataJSON()).toEqual({ body: ownText });
		return route.fulfill({
			status: 201,
			json: {
				comment: { ...initial, id: '11', own: true, nickname: reader.nickname, body: ownText },
				summary: { ...summary, commentCount: 3 }
			}
		});
	});
	await page.route(`**${apiPath}/comments/11`, (route) => {
		expect(route.request().method()).toBe('DELETE');
		expect(route.request().headers()['x-csrf-token']).toBe('test-csrf');
		return route.fulfill({ json: { message: 'Удалён' } });
	});

	await page.goto(path);
	await expect(page.getByRole('img', { name: 'Эскиз.webp' })).toBeVisible();
	await expect(page.locator('audio[controls]')).toHaveAttribute('src', files[1].url);
	await expect(page.locator('video[controls]')).toHaveAttribute('src', files[2].url);
	const download = page.getByRole('link', { name: /Скачать Материалы.zip/ });
	await expect(download).toHaveAttribute('href', files[3].url);
	await expect(download).toHaveAttribute('download', files[3].filename);

	const like = page.getByRole('button', { name: 'Нравится' });
	await expect(like).toContainText('9');
	await like.click();
	await expect(like).toHaveAttribute('aria-pressed', 'true');
	await expect(like).toContainText('10');
	await page.getByRole('button', { name: 'Загрузить ранее' }).click();
	await expect(page.getByText('Более ранний отклик')).toBeVisible();
	await expect(page.locator('#comment-8')).toBeFocused();
	await like.click();
	await expect(like).toHaveAttribute('aria-pressed', 'false');
	await expect(page.getByRole('button', { name: /Удалить/ })).toHaveCount(0);

	const field = page.getByLabel('Ваш комментарий');
	const submit = page.getByRole('button', { name: 'Отправить' });
	await field.fill('я'.repeat(2001));
	await expect(submit).toBeDisabled();
	await field.fill('🌱'.repeat(2000));
	await expect(submit).toBeEnabled();
	await field.fill(ownText);
	await submit.click();
	await expect(page.getByText(ownText, { exact: true })).toBeVisible();
	await expect(page.getByRole('link', { name: 'Комментарии 3' })).toBeVisible();
	await page.getByRole('button', { name: 'Удалить свой комментарий' }).click();
	await expect(page.getByText(ownText, { exact: true })).toHaveCount(0);
	await expect(page.getByRole('link', { name: 'Комментарии 2' })).toBeVisible();

	hasAccess = false;
	await like.click();
	await expect(page.getByRole('heading', { name: 'Нужен уровень «Оранжерея»' })).toBeVisible();
	await expect(page.getByText('Более ранний отклик')).toHaveCount(0);
	await expect(page.locator('main video, main audio, main [download]')).toHaveCount(0);
});
