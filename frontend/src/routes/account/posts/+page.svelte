<script lang="ts">
	import { goto } from '$app/navigation';
	import { api, checkError, errorMessage } from '#lib/api/client.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Spinner } from '#lib/components/ui/spinner/index.js';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import * as Empty from '#lib/components/ui/empty/index.js';
	import type { PageProps } from './$types';
	let { data }: PageProps = $props();
	let pending = $state(false);
	let failure = $state('');
	async function create() {
		if (pending) return;
		pending = true;
		failure = '';
		try {
			const result = await api.POST('/me/posts', {
				body: { title: 'Без названия', teaser: '', bodyMarkdown: '', minimumTierId: null }
			});
			checkError(result.error);
			if (result.data) await goto(`/account/posts/${result.data.post.id}`);
		} catch (cause) {
			failure = errorMessage(cause);
		} finally {
			pending = false;
		}
	}
</script>

<svelte:head><title>Публикации — etheyl</title><meta name="robots" content="noindex" /></svelte:head
>
<section class="creator-workspace page-inset">
	<a href="/account" class="back-link">← Личный кабинет</a>
	<div class="workspace-heading">
		<div>
			<h1>Публикации</h1>
			<p>От первой заметки до истории для подписчиков.</p>
		</div>
		{#if data.profile}<Button onclick={create} disabled={pending}
				>{#if pending}<Spinner data-icon="inline-start" />{/if}Новый пост</Button
			>{/if}
	</div>
	<nav class="creator-nav" aria-label="Настройки автора">
		<a href="/account/profile">Профиль</a><a href="/account/tiers">Уровни подписки</a><a
			href="/account/posts"
			aria-current="page">Публикации</a
		>
	</nav>
	{#if failure}<Alert.Root variant="destructive" role="alert"
			><Alert.Title>Не удалось создать пост</Alert.Title><Alert.Description
				>{failure}</Alert.Description
			></Alert.Root
		>{/if}
	{#if !data.profile}
		<Empty.Root
			><Empty.Header
				><Empty.Title>Начните с профиля</Empty.Title><Empty.Description
					>Сохраните имя и описание автора, чтобы создавать публикации.</Empty.Description
				></Empty.Header
			><Empty.Content><Button href="/account/profile">Оформить профиль</Button></Empty.Content
			></Empty.Root
		>
	{:else if !data.posts.length}
		<Empty.Root
			><Empty.Header
				><Empty.Title>Первая история ещё впереди</Empty.Title><Empty.Description
					>Создайте черновик. Добавьте текст, изображения, аудио, видео или файлы и выберите, кому
					они доступны.</Empty.Description
				></Empty.Header
			></Empty.Root
		>
	{:else}
		<div class="post-list">
			{#each data.posts as post (post.id)}
				<article class="post-list-row">
					<div class="flex flex-wrap gap-2">
						<Badge variant="outline">{post.publishedAt ? 'Опубликован' : 'Черновик'}</Badge><span
							class="text-sm text-muted-foreground"
							>{post.minimumTierId ? 'По подписке' : 'Открытый доступ'}</span
						>
					</div>
					<h2><a href={`/account/posts/${post.id}`}>{post.title}</a></h2>
					{#if post.teaser}<p class="post-teaser">{post.teaser}</p>{/if}
					<Button href={`/account/posts/${post.id}`} variant="outline">Редактировать</Button>
				</article>
			{/each}
		</div>
	{/if}
	<nav class="post-pagination" aria-label="Страницы публикаций">
		{#if data.offset > 0}<Button href={`?offset=${Math.max(0, data.offset - 24)}`} variant="outline"
				>Назад</Button
			>{/if}
		{#if data.hasMore}<Button href={`?offset=${data.offset + 24}`} variant="outline">Дальше</Button
			>{/if}
	</nav>
</section>
