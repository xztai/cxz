import { expect, test } from '@playwright/test';
import { readFile } from 'node:fs/promises';

test('черновик, все типы медиа, публикация и запрет прямого доступа', async ({
	page,
	context,
	browser
}, testInfo) => {
	test.setTimeout(90_000);
	const nickname = `post_${testInfo.project.name}_${Date.now()}`;
	const email = `${nickname}@example.com`;
	const origin = new URL(testInfo.project.use.baseURL as string).origin;
	const csrf = await context.request.get('/api/v1/auth/csrf');
	const { token } = await csrf.json();
	const headers = { Origin: origin, 'X-CSRF-Token': token };
	expect(
		(
			await context.request.post('/api/v1/auth/register', {
				headers,
				data: { email, nickname, password: 'Пароль автора публикаций 2026' }
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
				const message = await context.request.get(
					`${process.env.MAILPIT_URL ?? 'http://localhost:8025'}/api/v1/message/${messages[0].ID}`
				);
				actionToken = (await message.json()).Text.match(/#token=([A-Za-z0-9_-]+)/)?.[1] ?? '';
				return !!actionToken;
			},
			{ timeout: 30_000 }
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
	expect(
		(
			await context.request.put('/api/v1/me/profile', {
				headers,
				data: { nickname, displayName: 'Истории мастерской', description: '' }
			})
		).status()
	).toBe(200);
	const tierResponse = await context.request.post('/api/v1/me/tiers', {
		headers,
		data: { name: 'Читатель', description: '', priceKopecks: 10000 }
	});
	const { tier } = await tierResponse.json();
	const errors: string[] = [];
	page.on('pageerror', (cause) => errors.push(cause.message));
	await page.goto('/account/posts');
	await expect(page.getByText('Первая история ещё впереди')).toBeVisible();
	await page.getByRole('button', { name: 'Новый пост', exact: true }).click();
	await expect(page).toHaveURL(/\/account\/posts\/\d+$/);
	const id = page.url().split('/').at(-1)!;
	await page.getByLabel('Заголовок', { exact: true }).fill('Как рождается новая работа');
	await page.getByLabel('Публичный анонс', { exact: true }).fill('Заглянем в мастерскую.');
	await page
		.getByLabel('Текст поста · Markdown', { exact: true })
		.fill(
			'## Секрет мастерской\n\n**Настоящая история** и [ссылка](https://example.com).\n\n<script>window.injected = true</script>'
		);
	await page.getByLabel('Кому доступен пост', { exact: true }).selectOption(tier.id);
	await page.getByRole('button', { name: 'Сохранить', exact: true }).click();
	await expect(page.locator('.save-message')).toHaveText('Изменения сохранены.');
	const png = await page.evaluate(() => {
		const canvas = document.createElement('canvas');
		canvas.width = 800;
		canvas.height = 400;
		const drawing = canvas.getContext('2d')!;
		drawing.fillStyle = '#2448e5';
		drawing.fillRect(0, 0, 800, 400);
		return canvas.toDataURL('image/png').split(',')[1];
	});
	const wav = Buffer.alloc(44 + 4000);
	wav.write('RIFF');
	wav.writeUInt32LE(wav.length - 8, 4);
	wav.write('WAVEfmt ', 8);
	wav.writeUInt32LE(16, 16);
	wav.writeUInt16LE(1, 20);
	wav.writeUInt16LE(1, 22);
	wav.writeUInt32LE(8000, 24);
	wav.writeUInt32LE(16000, 28);
	wav.writeUInt16LE(2, 32);
	wav.writeUInt16LE(16, 34);
	wav.write('data', 36);
	wav.writeUInt32LE(4000, 40);
	const video = await readFile(new URL('../fixtures/sample.mp4', import.meta.url));
	const samples = [
		{
			kind: 'image',
			name: 'picture.png',
			mimeType: 'image/png',
			buffer: Buffer.from(png, 'base64')
		},
		{ kind: 'audio', name: 'voice.wav', mimeType: 'audio/wav', buffer: wav },
		{ kind: 'video', name: 'film.mp4', mimeType: 'video/mp4', buffer: video },
		{
			kind: 'attachment',
			name: 'notes.html',
			mimeType: 'text/html',
			buffer: Buffer.from('<h1>Только скачивание</h1>')
		}
	];
	for (const sample of samples) {
		await page.getByLabel('Тип файла', { exact: true }).selectOption(sample.kind);
		await page.getByLabel('Добавить файлы', { exact: true }).setInputFiles(sample);
		await expect(page.locator('.save-message')).toHaveText('Файлы добавлены к посту.');
		await expect(page.getByRole('link', { name: sample.name, exact: true })).toBeVisible();
	}
	await page.getByLabel('Заголовок', { exact: true }).fill('   ');
	await page.getByRole('button', { name: 'Сохранить', exact: true }).click();
	await expect(page.getByRole('alert')).toContainText('Проверьте заполненные поля.');
	await expect(page.getByRole('alert')).toBeInViewport();
	await page.screenshot({ path: testInfo.outputPath('post-validation-error.png') });
	await page.getByLabel('Заголовок', { exact: true }).fill('Как рождается новая работа');
	await page.getByRole('button', { name: 'Сохранить', exact: true }).click();
	await expect(page.locator('.save-message')).toHaveText('Изменения сохранены.');
	await page.locator('#main').focus();
	await page.evaluate(() => window.scrollTo({ top: 0, behavior: 'instant' }));
	expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
	await page.screenshot({ path: testInfo.outputPath('post-editor.png'), fullPage: true });
	const draft = (await (await context.request.get(`/api/v1/me/posts/${id}`)).json()).post;
	const guest = await browser.newContext({
		baseURL: origin,
		viewport: testInfo.project.use.viewport
	});
	try {
		expect((await guest.request.get(`/api/v1/posts/${id}`)).status()).toBe(404);
		await page.getByRole('button', { name: 'Опубликовать', exact: true }).click();
		await expect(page.locator('.save-message')).toHaveText('Пост опубликован.');
		const visitor = await guest.newPage();
		await visitor.goto(`/posts/${id}`);
		await expect(
			visitor.getByRole('heading', { name: 'Как рождается новая работа' })
		).toBeVisible();
		await expect(visitor.getByText('Заглянем в мастерскую.', { exact: true })).toBeVisible();
		await expect(visitor.getByText('Пост для подписчиков', { exact: true })).toBeVisible();
		expect(await visitor.content()).not.toContain('Секрет мастерской');
		expect(await visitor.content()).not.toContain('picture.png');
		await visitor.screenshot({ path: testInfo.outputPath('post-locked.png'), fullPage: true });
		for (const file of draft.media) {
			const denied = await guest.request.get(file.url, {
				headers: { Range: 'bytes=0-10', 'If-None-Match': '*' }
			});
			expect(denied.status()).toBe(403);
			expect(denied.headers()['cache-control']).toBe('no-store');
			expect((await guest.request.head(file.url)).status()).toBe(403);
			if (file.thumbnailUrl)
				expect((await guest.request.get(file.thumbnailUrl)).status()).toBe(403);
		}
		await page.getByLabel('Кому доступен пост', { exact: true }).selectOption('');
		await page.getByRole('button', { name: 'Сохранить', exact: true }).click();
		await expect(page.locator('.save-message')).toHaveText('Изменения сохранены.');
		await visitor.reload();
		await expect(
			visitor.getByRole('heading', { name: 'Секрет мастерской', exact: true })
		).toBeVisible();
		expect(await visitor.evaluate(() => 'injected' in window)).toBe(false);
		await expect(visitor.locator('.post-image')).toBeVisible();
		await expect
			.poll(() =>
				visitor.locator('audio').evaluate((element: HTMLAudioElement) => element.readyState)
			)
			.toBeGreaterThanOrEqual(1);
		for (const selector of ['audio', 'video']) {
			await visitor.locator(selector).evaluate(async (element: HTMLMediaElement) => {
				element.muted = true;
				await element.play();
			});
			await expect
				.poll(() =>
					visitor.locator(selector).evaluate((element: HTMLMediaElement) => element.currentTime)
				)
				.toBeGreaterThan(0);
			await visitor.locator(selector).evaluate((element: HTMLMediaElement) => element.pause());
		}
		await expect
			.poll(() =>
				visitor.locator('video').evaluate((element: HTMLVideoElement) => element.readyState)
			)
			.toBeGreaterThanOrEqual(1);
		for (const sample of samples) {
			const file = draft.media.find((item: { filename: string }) => item.filename === sample.name);
			const downloaded = await guest.request.get(file.url);
			expect(await downloaded.body()).toEqual(sample.buffer);
		}
		const download = visitor.waitForEvent('download');
		await visitor.getByRole('link', { name: 'Скачать · notes.html', exact: true }).click();
		expect((await download).suggestedFilename()).toBe('notes.html');
		await visitor.locator('#main').focus();
		await visitor.evaluate(() => window.scrollTo({ top: 0, behavior: 'instant' }));
		expect(await visitor.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(
			true
		);
		await visitor.screenshot({ path: testInfo.outputPath('post-public.png'), fullPage: true });
		await page.getByRole('button', { name: 'Снять с публикации', exact: true }).click();
		await expect(page.locator('.save-message')).toHaveText('Пост возвращён в черновики.');
		expect((await guest.request.get(`/api/v1/posts/${id}`)).status()).toBe(404);
		expect((await guest.request.get(draft.media[0].url)).status()).toBe(404);
		await page.getByRole('button', { name: 'Удалить пост', exact: true }).click();
		await page.getByRole('button', { name: 'Удалить навсегда', exact: true }).click();
		await expect(page).toHaveURL(/\/account\/posts$/);
		expect((await context.request.get(`/api/v1/posts/${id}`)).status()).toBe(404);
		expect(errors).toEqual([]);
	} finally {
		await guest.close();
	}
});
