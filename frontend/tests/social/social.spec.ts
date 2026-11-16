import { expect, test, type BrowserContext } from '@playwright/test';
import { spawn } from 'node:child_process';

async function register(context: BrowserContext, nickname: string, origin: string) {
	const email = `${nickname}@example.com`;
	const { token } = await (await context.request.get('/api/v1/auth/csrf')).json();
	const headers = { Origin: origin, 'X-CSRF-Token': token };
	expect(
		(
			await context.request.post('/api/v1/auth/register', {
				headers,
				data: { email, nickname, password: 'Пароль тестов ленты 2026' }
			})
		).status()
	).toBe(202);
	let action = '';
	await expect
		.poll(
			async () => {
				const { messages } = await (
					await context.request.get(
						`${process.env.MAILPIT_URL ?? 'http://localhost:8025'}/api/v1/search`,
						{ params: { query: `to:${email}` } }
					)
				).json();
				if (!messages?.length) return false;
				const mail = await (
					await context.request.get(
						`${process.env.MAILPIT_URL ?? 'http://localhost:8025'}/api/v1/message/${messages[0].ID}`
					)
				).json();
				action = mail.Text.match(/#token=([A-Za-z0-9_-]+)/)?.[1] ?? '';
				return !!action;
			},
			{ timeout: 30000 }
		)
		.toBe(true);
	expect(
		(
			await context.request.post('/api/v1/auth/verify-email', { headers, data: { token: action } })
		).status()
	).toBe(200);
	return headers;
}
async function seed(
	project: string,
	author: string,
	reader: string,
	tierId: string,
	postId: string
) {
	// Test-only database fixture. No HTTP endpoint exists for granting paid access.
	// Never permit this helper to target the normal application project.
	if (!/^[a-z0-9-]+-social-tests$/.test(project))
		throw new Error('Use a dedicated Compose project ending in -social-tests');
	const sql = `
 BEGIN;
 INSERT INTO billing_agreements(subscriber_id,author_id,tier_id,price_kopecks,paid_until)
 SELECT r.id,a.id,:'tier'::bigint,19950,now()+interval '1 month' FROM users a,users r WHERE a.nickname=:'author' AND r.nickname=:'reader';
 INSERT INTO payments(id,agreement_id,tier_id,amount_kopecks,renewal,save_method,request_body,provider_id,status,completed_at)
 SELECT 'social-'||a.id,a.id,a.tier_id,19950,false,false,'{}','provider-social-'||a.id,'succeeded',now() FROM billing_agreements a JOIN users u ON u.id=a.author_id WHERE u.nickname=:'author';
 INSERT INTO subscriptions(subscriber_id,author_id,tier_id,starts_at,expires_at)
 SELECT a.subscriber_id,a.author_id,a.tier_id,now()-interval '1 minute',a.paid_until FROM billing_agreements a JOIN users u ON u.id=a.author_id WHERE u.nickname=:'author';
 INSERT INTO posts(author_id,title,teaser,body_markdown,minimum_tier_id,published_at)
 SELECT u.id,'Лента — история '||i,'Публичный анонс','Текст тестовой истории',:'tier'::bigint,now()-interval '1 day' FROM users u CROSS JOIN generate_series(1,25) i WHERE u.nickname=:'author';
 INSERT INTO post_comments(post_id,user_id,body) SELECT :'post'::bigint,u.id,'Предыдущий комментарий '||i FROM users u CROSS JOIN generate_series(1,25) i WHERE u.nickname=:'author';
 COMMIT;`;
	await new Promise<void>((resolve, reject) => {
		const child = spawn(
			'docker',
			[
				'compose',
				'-p',
				project,
				'exec',
				'-T',
				'postgres',
				'psql',
				'-U',
				'etheyl',
				'-d',
				'etheyl',
				'-v',
				'ON_ERROR_STOP=1',
				'-v',
				`author=${author}`,
				'-v',
				`reader=${reader}`,
				'-v',
				`tier=${tierId}`,
				'-v',
				`post=${postId}`
			],
			{ cwd: '..', stdio: ['pipe', 'pipe', 'pipe'] }
		);
		let failure = '';
		child.stderr.on('data', (b) => (failure += b));
		child.on('error', reject);
		child.on('close', (code) =>
			code === 0 ? resolve() : reject(new Error(`Test fixture failed: ${failure}`))
		);
		child.stdin.end(sql);
	});
}

test('оплаченная лента, лайки, плоские комментарии и аналитика автора', async ({
	page,
	context,
	browser
}, testInfo) => {
	test.setTimeout(120000);
	const project = process.env.E2E_SOCIAL_PROJECT;
	if (!project) throw new Error('E2E_SOCIAL_PROJECT must identify the dedicated social test stack');
	const origin = new URL(testInfo.project.use.baseURL as string).origin;
	const suffix = `${testInfo.project.name}_${Date.now()}`;
	const author = `sa_${suffix}`,
		reader = `sr_${suffix}`;
	const creator = await browser.newContext({
		baseURL: origin,
		viewport: testInfo.project.use.viewport
	});
	try {
		const headers = await register(creator, author, origin);
		expect(
			(
				await creator.request.put('/api/v1/me/profile', {
					headers,
					data: {
						nickname: author,
						displayName: 'Мастерская историй',
						description: 'Истории для своих'
					}
				})
			).status()
		).toBe(200);
		const { tier } = await (
			await creator.request.post('/api/v1/me/tiers', {
				headers,
				data: { name: 'Мастерская', description: 'Все истории', priceKopecks: 19950 }
			})
		).json();
		const { post } = await (
			await creator.request.post('/api/v1/me/posts', {
				headers,
				data: {
					title: 'История для подписчиков',
					teaser: 'Смотрим на новую работу',
					bodyMarkdown: '## Секрет мастерской\n\nОплаченная история.',
					minimumTierId: tier.id
				}
			})
		).json();
		expect(
			(await creator.request.post(`/api/v1/me/posts/${post.id}/publish`, { headers })).status()
		).toBe(200);
		await register(context, reader, origin);
		await page.goto('/account/feed');
		await expect(page.getByText('Пока нет новых историй', { exact: true })).toBeVisible();
		await page.goto(`/posts/${post.id}`);
		await expect(page.getByText('Пост для подписчиков', { exact: true })).toBeVisible();
		expect(await page.content()).not.toContain('Секрет мастерской');
		expect((await context.request.get(`/api/v1/posts/${post.id}/discussion`)).status()).toBe(403);
		await seed(project, author, reader, tier.id, post.id);
		await page.goto('/account/feed');
		await expect(page.getByRole('heading', { name: 'Моя лента' })).toBeVisible();
		await expect(page.getByRole('heading', { name: 'История для подписчиков' })).toBeVisible();
		await expect(
			page.getByRole('link', { name: 'Следующие публикации', exact: true })
		).toBeVisible();
		await page.getByRole('link', { name: 'Следующие публикации', exact: true }).click();
		await expect(page.locator('[data-slot=card]')).toHaveCount(2);
		await page.getByRole('link', { name: 'К новым публикациям', exact: true }).click();
		await page.getByRole('heading', { name: 'История для подписчиков' }).getByRole('link').click();
		await expect(page.getByRole('heading', { name: 'Секрет мастерской' })).toBeVisible();
		await page.getByRole('button', { name: 'Поставить лайк', exact: true }).click();
		await expect(page.getByRole('button', { name: 'Убрать лайк', exact: true })).toHaveAttribute(
			'aria-pressed',
			'true'
		);
		await page.getByRole('button', { name: 'Убрать лайк', exact: true }).click();
		await expect(page.getByRole('button', { name: 'Поставить лайк', exact: true })).toHaveAttribute(
			'aria-pressed',
			'false'
		);
		await page
			.getByLabel('Ваш комментарий', { exact: true })
			.fill('<script>window.socialInjected=true</script> Спасибо за историю!');
		await page.getByRole('button', { name: 'Отправить комментарий', exact: true }).click();
		await expect(page.getByText('Комментарий добавлен.', { exact: true })).toBeVisible();
		await expect(
			page.getByText('<script>window.socialInjected=true</script> Спасибо за историю!', {
				exact: true
			})
		).toBeVisible();
		expect(await page.evaluate(() => Object.hasOwn(window, 'socialInjected'))).toBe(false);
		await page.getByRole('button', { name: 'Ещё комментарии', exact: true }).click();
		await expect(page.getByText('Предыдущий комментарий 1', { exact: true })).toBeVisible();
		await page.getByRole('button', { name: 'Удалить свой комментарий', exact: true }).click();
		await expect(page.getByText('Комментарий удалён.', { exact: true })).toBeVisible();
		const creatorPage = await creator.newPage();
		const browserErrors: string[] = [];
		creatorPage.on('pageerror', (e) => browserErrors.push(e.message));
		await creatorPage.goto('/account/analytics');
		await expect(creatorPage.getByTestId('payment-count')).toHaveText('1');
		await expect(creatorPage.getByTestId('payment-amount')).toContainText('199,50');
		await expect(creatorPage.locator('[data-slot=chart] svg').first()).toBeVisible({
			timeout: 30000
		});
		await creatorPage.screenshot({ path: testInfo.outputPath('analytics.png'), fullPage: true });
		expect(
			await creatorPage.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)
		).toBe(true);
		await creatorPage.getByText('Данные по дням · доступная таблица', { exact: true }).click();
		await expect(
			creatorPage.getByRole('table', { name: 'Подтверждённые тестовые платежи, UTC' })
		).toBeVisible();
		await creatorPage.getByLabel('С даты · UTC', { exact: true }).fill('2025-01-01');
		await creatorPage.getByLabel('По дату · UTC, включительно', { exact: true }).fill('2025-01-02');
		await creatorPage.getByRole('button', { name: 'Показать период', exact: true }).click();
		await expect(creatorPage.getByTestId('payment-count')).toHaveText('0');
		await expect(
			creatorPage.getByText('В этом периоде платежей нет', { exact: true })
		).toBeVisible();
		expect(browserErrors).toEqual([]);
	} finally {
		await creator.close();
	}
});
