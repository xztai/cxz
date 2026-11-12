import { expect, type Page, type Browser, type TestInfo } from '@playwright/test';

export async function creatorExperience(
	page: Page,
	browser: Browser,
	testInfo: TestInfo,
	nickname: string
) {
	const displayName = `Мастерская ${testInfo.project.name}`;
	const noOverflow = async (target = page) => {
		expect(await target.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(
			true
		);
		await target.locator('#main').focus();
		await target.evaluate(() => window.scrollTo({ top: 0, behavior: 'instant' }));
	};
	await page.getByRole('link', { name: 'Оформить профиль', exact: true }).click();
	await expect(page.getByLabel('Аватар', { exact: true })).toBeDisabled();
	await page.getByLabel('Имя автора').fill(displayName);
	await page
		.getByLabel('Описание', { exact: true })
		.fill('Керамика, заметки из мастерской и новые работы.\nЗдесь рождаются вещи с историей.');
	await page.getByRole('button', { name: 'Сохранить профиль', exact: true }).click();
	await expect(page.getByRole('status')).toHaveText('Профиль сохранён и доступен в каталоге.');
	await page.getByLabel('Аватар', { exact: true }).scrollIntoViewIfNeeded();
	await page.getByLabel('Аватар', { exact: true }).setInputFiles({
		name: 'bad.svg',
		mimeType: 'image/svg+xml',
		buffer: Buffer.from('<svg xmlns="http://www.w3.org/2000/svg"/>')
	});
	await expect(page.getByRole('alert')).toContainText('Выберите исправное изображение');
	await expect(page.getByRole('alert')).toBeInViewport();
	await page.screenshot({ path: testInfo.outputPath('creator-upload-error-viewport.png') });
	const makePng = async (width: number, height: number) =>
		Buffer.from(
			await page.evaluate(
				({ width, height }) => {
					const canvas = document.createElement('canvas');
					canvas.width = width;
					canvas.height = height;
					const ctx = canvas.getContext('2d')!;
					ctx.fillStyle = '#ddd3b6';
					ctx.fillRect(0, 0, width, height);
					ctx.fillStyle = '#2448e5';
					ctx.beginPath();
					ctx.arc(width / 2, height / 2, height / 3, 0, Math.PI * 2);
					ctx.fill();
					return canvas.toDataURL('image/png').split(',')[1];
				},
				{ width, height }
			),
			'base64'
		);
	await page
		.getByLabel('Аватар', { exact: true })
		.setInputFiles({ name: 'avatar.png', mimeType: 'image/png', buffer: await makePng(160, 160) });
	await expect(page.getByAltText('Ваш аватар')).toBeVisible();
	await page
		.getByLabel('Обложка', { exact: true })
		.setInputFiles({ name: 'cover.png', mimeType: 'image/png', buffer: await makePng(900, 300) });
	await expect(page.getByAltText('Ваша обложка')).toBeVisible();
	await page.reload();
	await expect(page.getByLabel('Имя автора')).toHaveValue(displayName);
	await expect(page.getByAltText('Ваш аватар')).toBeVisible();
	await noOverflow();
	await page.screenshot({ path: testInfo.outputPath('creator-profile.png'), fullPage: true });
	await page.getByRole('link', { name: 'Уровни подписки', exact: true }).click();
	await page.getByLabel('Название уровня').fill('Читатель');
	await page.getByLabel('Что входит в уровень').fill('Новости и заметки из мастерской.');
	await page.getByLabel('Цена, ₽ в месяц').fill('99');
	await page.getByRole('button', { name: 'Создать уровень', exact: true }).click();
	await expect(page.getByRole('status')).toHaveText('Уровень сохранён.');
	await page.getByLabel('Название уровня').fill('Близкий круг');
	await page
		.getByLabel('Что входит в уровень')
		.fill('Процесс работы и ранний взгляд на новые коллекции.');
	await page.getByLabel('Цена, ₽ в месяц').fill('299,50');
	await page.getByRole('button', { name: 'Создать уровень', exact: true }).click();
	await expect(page.locator('.tier-list .tier-row').first()).toContainText('Близкий круг');
	const result = await page.request.get('/api/v1/me/tiers');
	const { tiers } = (await result.json()) as {
		tiers: { id: string; name: string; priceKopecks: number }[];
	};
	expect(tiers[0].priceKopecks).toBe(29950);
	await page.getByRole('button', { name: 'Поднять уровень «Читатель»', exact: true }).click();
	await expect(page.getByRole('status')).toHaveText('Порядок сохранён.');
	await expect(page.locator('.tier-list .tier-row').first()).toContainText('Читатель');
	await page.getByRole('button', { name: 'Поднять уровень «Близкий круг»', exact: true }).click();
	await expect(page.getByRole('status')).toHaveText('Порядок сохранён.');
	await expect(page.locator('.tier-list .tier-row').first()).toContainText('Близкий круг');
	await page
		.locator('.tier-row')
		.filter({ has: page.getByRole('heading', { name: 'Близкий круг', exact: true }) })
		.getByRole('button', { name: 'Редактировать', exact: true })
		.click();
	await expect(page.getByLabel('Название уровня')).toHaveValue('Близкий круг');
	await expect(page.getByLabel('Название уровня')).toBeFocused();
	await expect(page.getByLabel('Название уровня')).toBeInViewport();
	await page.screenshot({ path: testInfo.outputPath('creator-edit-focused-viewport.png') });
	await page.getByLabel('Цена, ₽ в месяц').fill('499,01');
	await page.getByRole('button', { name: 'Сохранить изменения', exact: true }).click();
	await expect(page.locator('.tier-row').first()).toContainText('499,01');
	await noOverflow();
	await page.screenshot({ path: testInfo.outputPath('creator-tiers.png'), fullPage: true });
	await page.getByRole('link', { name: 'Открыть страницу', exact: true }).click();
	await expect(page.getByRole('heading', { name: displayName, exact: true })).toBeVisible();
	await expect(page.getByRole('link', { name: 'Редактировать профиль' })).toBeVisible();
	await noOverflow();
	await page.screenshot({ path: testInfo.outputPath('creator-public-owner.png'), fullPage: true });
	const guest = await browser.newContext({ viewport: page.viewportSize() });
	try {
		const visitor = await guest.newPage();
		const errors: string[] = [];
		visitor.on('pageerror', (error) => errors.push(error.message));
		await visitor.goto(new URL('/authors', page.url()).href);
		await visitor.getByLabel('Поиск авторов').fill(nickname.toUpperCase());
		await visitor.getByRole('button', { name: 'Найти', exact: true }).click();
		await expect(visitor.getByRole('heading', { name: displayName, exact: true })).toBeVisible();
		await noOverflow(visitor);
		await visitor.screenshot({ path: testInfo.outputPath('creator-catalog.png'), fullPage: true });
		await visitor.getByRole('link', { name: new RegExp(displayName) }).click();
		await expect(visitor.getByRole('heading', { name: displayName, exact: true })).toBeVisible();
		await expect(visitor.getByRole('link', { name: 'Редактировать профиль' })).toHaveCount(0);
		await expect(visitor.getByRole('link', { name: 'Настроить уровни' })).toHaveCount(0);
		await expect(visitor.getByRole('img', { name: `Аватар автора ${displayName}` })).toBeVisible();
		await expect(visitor.getByText('Также включает: Читатель.')).toBeVisible();
		await noOverflow(visitor);
		await visitor.screenshot({ path: testInfo.outputPath('creator-public.png'), fullPage: true });
		const csrf = await guest.request.get(new URL('/api/v1/auth/csrf', page.url()).href);
		const { token } = await csrf.json();
		const denied = await guest.request.put(
			new URL(`/api/v1/me/tiers/${tiers[0].id}`, page.url()).href,
			{
				headers: { Origin: new URL(page.url()).origin, 'X-CSRF-Token': token },
				data: { name: 'Взлом', description: '', priceKopecks: 0 }
			}
		);
		expect(denied.status()).toBe(401);
		expect(errors).toEqual([]);
	} finally {
		await guest.close();
	}
	await page.goto('/account/tiers');
	await page
		.locator('.tier-row')
		.first()
		.getByRole('button', { name: 'Убрать уровень', exact: true })
		.click();
	await page.getByRole('button', { name: 'Архивировать', exact: true }).click();
	await expect(page.locator('.tier-archive')).toContainText('Близкий круг');
	await page.goto(`/authors/${nickname}`);
	await expect(page.getByRole('heading', { name: 'Близкий круг', exact: true })).toHaveCount(0);
	await page.goto('/account/profile');
	await page.getByRole('button', { name: 'Удалить аватар', exact: true }).click();
	await expect(page.getByAltText('Ваш аватар')).toHaveCount(0);
	await page.goto('/account');
}
