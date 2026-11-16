<script lang="ts">
	import { navigating } from '$app/state';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Empty from '#lib/components/ui/empty/index.js';
	import type { PageProps } from './$types';
	let { data }: PageProps = $props();
	function date(value: string | null) {
		return value
			? new Intl.DateTimeFormat('ru-RU', { dateStyle: 'long' }).format(new Date(value))
			: '';
	}
</script>

<svelte:head><title>Моя лента — etheyl</title><meta name="robots" content="noindex" /></svelte:head>
<section class="workspace-page page-inset">
	<a href="/account" class="back-link">Личный кабинет</a>
	<div class="workspace-heading">
		<div>
			<h1>Моя лента</h1>
			<p>Новые истории авторов, которых вы поддерживаете.</p>
		</div>
		<Button href="/account/subscriptions" variant="outline">Мои подписки</Button>
	</div>
	{#if navigating.to?.url.pathname === '/account/feed'}<div
			aria-label="Загрузка ленты"
			role="status"
			class="flex flex-col gap-4"
		>
			<Skeleton class="h-8 w-2/3" /><Skeleton class="h-32 w-full" />
		</div>
	{:else if data.posts.length}<div class="flex max-w-4xl flex-col gap-6">
			{#each data.posts as post (post.id)}
				<Card.Root
					><Card.Header
						><Card.Description
							><a href={`/authors/${post.authorNickname}`}>{post.authorName}</a> ·
							<time datetime={post.publishedAt ?? undefined}>{date(post.publishedAt)}</time
							></Card.Description
						><Card.Title><h2><a href={`/posts/${post.id}`}>{post.title}</a></h2></Card.Title
						><Card.Action
							><Badge variant="outline"
								>{!post.canRead
									? 'Нужен старший уровень'
									: post.minimumTierId
										? 'По подписке'
										: 'Открытый пост'}</Badge
							></Card.Action
						></Card.Header
					>
					<Card.Content
						><p class="whitespace-pre-wrap break-words">
							{post.teaser || 'Автор приглашает прочитать новую публикацию.'}
						</p></Card.Content
					>
					<Card.Footer class="flex flex-wrap gap-3"
						><Button href={`/posts/${post.id}`} variant="outline"
							>{post.canRead ? 'Читать публикацию' : 'Открыть анонс'}</Button
						>{#if !post.canRead}<Button href={`/authors/${post.authorNickname}`} variant="link"
								>Выбрать уровень</Button
							>{/if}</Card.Footer
					>
				</Card.Root>{/each}
		</div>
	{:else}<Empty.Root
			><Empty.Header
				><Empty.Title
					>{data.cursor ? 'Вы прочитали всю ленту' : 'Пока нет новых историй'}</Empty.Title
				><Empty.Description
					>{data.cursor
						? 'Вернитесь к новым публикациям.'
						: 'Здесь появятся опубликованные истории авторов с действующей оплаченной подпиской. Выберите автора в каталоге или проверьте свои подписки.'}</Empty.Description
				></Empty.Header
			><Empty.Content><Button href="/authors" variant="outline">Найти автора</Button></Empty.Content
			></Empty.Root
		>{/if}
	<nav class="post-pagination" aria-label="Страницы ленты">
		{#if data.cursor}<Button href="/account/feed" variant="outline">К новым публикациям</Button
			>{/if}{#if data.nextCursor}<Button
				href={`?cursor=${encodeURIComponent(data.nextCursor)}`}
				variant="outline">Следующие публикации</Button
			>{/if}
	</nav>
</section>
