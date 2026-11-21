<script lang="ts">
	import { resolve } from '$app/paths';
	import { onMount } from 'svelte';
	import { api, ApiError, errorMessage, type Post, type Profile } from '#lib/api/client.js';
	import Icon from '#lib/components/Icon.svelte';
	import { getSession } from '#lib/session.svelte.js';
	import { formatPostDate } from '#lib/studio/format.js';
	import PostDiscussion from './PostDiscussion.svelte';
	import PostMedia from './PostMedia.svelte';

	let { id, signedIn }: { id: string; signedIn: boolean } = $props();
	const session = getSession();
	const controller = new AbortController();
	const signal = controller.signal;
	let post = $state.raw<Post | null>(null);
	let profile = $state.raw<Profile | null>(null);
	let missing = $state(false);
	let message = $state('');

	onMount(() => {
		void loadPost();
		return () => controller.abort();
	});

	async function loadPost() {
		message = '';
		missing = !/^[1-9]\d{0,18}$/.test(id);
		if (missing) return;
		let opened: Post;
		try {
			({ post: opened } = await api.post(id, signal));
		} catch (error) {
			if (signal.aborted) return;
			if (error instanceof ApiError && error.status === 404) missing = true;
			else message = errorMessage(error);
			return;
		}
		if (signal.aborted) return;
		post = opened;
		try {
			const author = await api.author(opened.authorNickname, signal);
			if (!signal.aborted) profile = author.profile;
		} catch {
			// The post carries the canonical nickname even when the profile request fails.
		}
	}

	function updateAccess(status: number) {
		if (status === 401) void session.refresh();
		if (status === 403 && post && post.minimumTierId) {
			post = {
				...post,
				canRead: false,
				bodyMarkdown: undefined,
				bodyHtml: undefined,
				media: undefined
			};
		}
	}
</script>

<svelte:head
	><title>{post?.title ?? (missing ? 'Пост не найден' : 'Пост')} — etheyl</title></svelte:head
>

<main id="main" class="shell simple-page post-page">
	{#if post}
		{@const authorName = profile?.displayName ?? post.authorNickname}
		{@const authorHref = resolve('/authors/[nickname]', { nickname: post.authorNickname })}
		<article aria-labelledby="post-title">
			<header class="post-header">
				<a class="author" href={authorHref}>
					{#if profile?.avatar}
						<img class="avatar" src={profile.avatar.url} width="44" height="44" alt="" />
					{:else}
						<span class="avatar avatar-initial" aria-hidden="true"
							>{Array.from(authorName)[0]?.toUpperCase()}</span
						>
					{/if}
					<span>{authorName}</span>
				</a>
				<h1 id="post-title">{post.title}</h1>
				<div class="post-meta">
					{#if post.publishedAt}
						<time datetime={post.publishedAt}>{formatPostDate(post.publishedAt)}</time>
					{:else}
						<span class="access-label">Черновик</span>
					{/if}
					{#if post.minimumTierId}
						<span class="access-label tier-label">
							<Icon name="lock" size={15} />
							{#if post.minimumTierName}<span class="sr-only">Уровень доступа:</span>{/if}
							{post.minimumTierName ?? 'По подписке'}
						</span>
					{:else if post.publishedAt}
						<span class="access-label">Публично</span>
					{/if}
				</div>
			</header>

			{#if post.canRead}
				{#if post.bodyHtml}
					<div class="post-body">
						<!-- eslint-disable-next-line svelte/no-at-html-tags -- The API renders CommonMark with raw HTML and unsafe URLs disabled. -->
						{@html post.bodyHtml}
					</div>
				{/if}
				{#if post.media?.length}<PostMedia files={post.media} />{/if}
			{:else}
				{#if post.teaser}<p class="post-teaser">{post.teaser}</p>{/if}
				<section class="paywall" aria-labelledby="paywall-heading">
					<span class="paywall-lock"><Icon name="lock" size={24} /></span>
					<h2 id="paywall-heading">
						{post.minimumTierName ? `Нужен уровень «${post.minimumTierName}»` : 'Нужна подписка'}
					</h2>
					<a class="button button-primary" href="{authorHref}#author-tiers-heading"
						>Подписаться<Icon name="arrow" size={18} /></a
					>
				</section>
			{/if}
		</article>

		{#if post.publishedAt}
			<PostDiscussion
				postId={post.id}
				canRead={post.canRead}
				{signedIn}
				onAccessLost={updateAccess}
			/>
		{/if}
	{:else if missing || message}
		<h1>{missing ? 'Пост не найден' : 'Не удалось открыть пост'}</h1>
		{#if missing}
			<a class="button button-outline" href={resolve('/authors')}>Найти авторов</a>
		{:else}
			<p class="feedback error-feedback" role="alert">{message}</p>
			<button class="button button-outline" type="button" onclick={() => void loadPost()}>
				Попробовать ещё раз
			</button>
		{/if}
	{:else}
		<p class="simple-intro" role="status">Загружаем пост…</p>
	{/if}
</main>

<style>
	.post-page {
		max-width: 760px;
		padding-top: 48px;
	}
	.post-header {
		margin-bottom: 30px;
	}
	.author {
		display: inline-flex;
		align-items: center;
		gap: 11px;
		max-width: 100%;
		font-size: 15px;
		font-weight: 800;
		overflow-wrap: anywhere;
	}
	.author:hover {
		color: var(--blue-hover);
		text-decoration: underline;
	}
	.avatar {
		flex: none;
		width: 44px;
		height: 44px;
		border-radius: 10px;
		object-fit: cover;
	}
	.avatar-initial {
		display: grid;
		place-items: center;
		background: var(--blue-soft);
		color: var(--blue-hover);
		font-size: 19px;
	}
	h1 {
		margin: 18px 0 12px;
		overflow-wrap: anywhere;
	}
	.post-meta {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 6px 14px;
		color: var(--muted);
		font-size: 13px;
	}
	.access-label {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		padding: 3px 8px;
		border-radius: 6px;
		background: var(--surface);
		font-weight: 700;
		overflow-wrap: anywhere;
	}
	.access-label :global(svg),
	.paywall-lock {
		flex: none;
	}
	.tier-label {
		background: var(--blue-soft);
		color: var(--blue-hover);
	}
	.post-body,
	.post-teaser {
		max-width: 72ch;
		color: var(--ink);
		font-size: 17px;
		line-height: 1.75;
		overflow-wrap: anywhere;
	}
	.post-teaser {
		margin-bottom: 22px;
		white-space: pre-wrap;
	}
	.post-body :global(:where(p, ul, ol, pre, blockquote)) {
		margin: 0 0 1em;
	}
	.post-body :global(:where(h1, h2, h3, h4, h5, h6)) {
		margin: 1.35em 0 0.35em;
		font-weight: 800;
		line-height: 1.3;
	}
	.post-body :global(:where(h1, h2)) {
		font-size: 1.45em;
	}
	.post-body :global(h3) {
		font-size: 1.2em;
	}
	.post-body :global(a) {
		color: var(--blue-hover);
		text-decoration: underline;
	}
	.post-body :global(:where(ul, ol)) {
		padding-inline-start: 1.55em;
		list-style: revert;
	}
	.post-body :global(blockquote) {
		padding-inline-start: 1em;
		border-inline-start: 1px solid var(--line);
		color: var(--muted);
	}
	.post-body :global(pre) {
		overflow: auto;
		padding: 12px 16px;
		border-radius: 9px;
		background: var(--surface);
	}
	.post-body :global(hr) {
		margin-block: 1.5em;
		border-color: var(--line);
	}
	.post-body :global(img) {
		max-width: 100%;
		height: auto;
		border-radius: var(--radius);
	}
	.paywall {
		display: grid;
		grid-template-columns: auto minmax(0, 1fr) auto;
		align-items: center;
		gap: 16px;
		padding: 24px;
		border: 1px solid var(--line);
		border-radius: 16px;
		background: var(--surface);
	}
	.paywall-lock {
		display: grid;
		place-items: center;
		width: 46px;
		height: 46px;
		border-radius: 10px;
		background: var(--blue-soft);
		color: var(--blue-hover);
	}
	.paywall h2 {
		font-size: 19px;
		font-weight: 800;
		text-wrap: pretty;
		overflow-wrap: anywhere;
	}
	@media (max-width: 640px) {
		.post-page {
			padding-top: 28px;
		}
		.post-header {
			margin-bottom: 24px;
		}
		.post-body,
		.post-teaser {
			font-size: 16px;
		}
		.paywall {
			grid-template-columns: auto minmax(0, 1fr);
			gap: 14px;
			padding: 16px;
		}
		.paywall .button {
			grid-column: 1 / -1;
		}
	}
</style>
