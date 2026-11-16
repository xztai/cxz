<script lang="ts">
	import PostDiscussion from '#lib/components/social/PostDiscussion.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { fileSize } from '#lib/posts/files.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import type { PageProps } from './$types';
	let { data }: PageProps = $props();
	let post = $derived(data.post);
	let owner = $derived(data.user?.nickname === post.authorNickname);
</script>

<svelte:head
	><title>{post.title} — etheyl</title><meta
		name="description"
		content={post.teaser.slice(0, 160)}
	/></svelte:head
>
<article class="post-reader page-inset">
	<a class="back-link" href={`/authors/${post.authorNickname}`}>← @{post.authorNickname}</a>
	<header class="post-reader-heading">
		<div class="flex flex-wrap gap-2">
			{#if !post.publishedAt}<Badge variant="outline">Черновик</Badge>{/if}<Badge variant="outline"
				>{post.minimumTierId ? 'По подписке' : 'Открытый доступ'}</Badge
			>
		</div>
		<h1>{post.title}</h1>
		{#if post.teaser}<p class="post-teaser">{post.teaser}</p>{/if}
		{#if owner}<Button href={`/account/posts/${post.id}`} variant="outline"
				>Редактировать пост</Button
			>{/if}
	</header>
	{#if post.canRead}
		<div class="post-prose">
			<!-- eslint-disable-next-line svelte/no-at-html-tags -- Go CommonMark renderer disables raw HTML and unsafe link schemes; covered by API and browser XSS tests. -->
			{@html post.bodyHtml ?? ''}
		</div>
		<div class="post-media-stack">
			{#each post.media ?? [] as file (file.id)}
				<figure>
					{#if file.kind === 'image'}
						<a href={file.url}
							><img
								class="post-image"
								src={file.thumbnailUrl ?? file.url}
								alt={file.filename}
								loading="lazy"
							/></a
						>
					{:else if file.kind === 'audio'}
						<audio controls preload="metadata" src={file.url} aria-label={file.filename}
							><a href={file.url}>Скачать аудио</a></audio
						>
					{:else if file.kind === 'video'}
						<!-- svelte-ignore a11y_media_has_caption (original user media has no generated captions) -->
						<video controls playsinline preload="metadata" src={file.url} aria-label={file.filename}
							><a href={file.url}>Скачать видео</a></video
						>
					{/if}
					<figcaption>
						<a href={file.url}
							>{file.kind === 'attachment' ? 'Скачать' : 'Открыть оригинал'} · {file.filename}</a
						><span>{fileSize(file.sizeBytes)}</span>
					</figcaption>
				</figure>
			{/each}
		</div>
	{:else}
		<Alert.Root
			><Alert.Title>Пост для подписчиков</Alert.Title><Alert.Description
				>Текст и файлы доступны с минимального уровня, выбранного автором, и на старших уровнях.
				Выберите подходящий уровень на странице автора.</Alert.Description
			></Alert.Root
		>
		<div class="post-actions">
			<Button href={`/authors/${post.authorNickname}`}>Выбрать подписку</Button>
		</div>
		{#if !data.user}<div class="post-actions">
				<Button href="/auth/login">Войти в аккаунт</Button>
			</div>{/if}
	{/if}
	{#if post.canRead && post.publishedAt}
		{#if data.user}{#key post.id}<PostDiscussion postId={post.id} />{/key}{:else}<section
				class="mt-12 flex flex-col items-start gap-4 border-t border-border pt-8"
			>
				<h2>Обсуждение</h2>
				<p>Войдите, чтобы поставить лайк или оставить комментарий.</p>
				<Button href="/auth/login" variant="outline">Войти для обсуждения</Button>
			</section>{/if}
	{/if}
</article>
