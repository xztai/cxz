<script lang="ts">
	import { resolve } from '$app/paths';
	import { onMount, tick } from 'svelte';
	import { api, errorMessage, type PostListItem } from '#lib/api/client.js';
	import Icon from '#lib/components/Icon.svelte';
	import { formatPostDate } from '#lib/studio/format.js';

	let { nickname }: { nickname: string } = $props();
	let posts = $state.raw<PostListItem[]>([]);
	let hasMore = $state(true);
	let busy = $state(false);
	let message = $state('');
	let nextOffset = 0;
	let list: HTMLUListElement | undefined;
	const controller = new AbortController();

	onMount(() => {
		void loadPosts();
		return () => controller.abort();
	});

	function rememberList(element: HTMLUListElement) {
		list = element;
		return () => (list = undefined);
	}

	async function loadPosts(trigger?: HTMLButtonElement) {
		if (busy || !hasMore) return;
		busy = true;
		message = '';
		const hadFocus = trigger === document.activeElement;
		const previousCount = posts.length;
		let loaded = false;
		try {
			const result = await api.authorPosts(nickname, nextOffset, controller.signal);
			if (controller.signal.aborted) return;
			const existingIds = new Set(posts.map((post) => post.id));
			posts = [...posts, ...result.posts.filter((post) => !existingIds.has(post.id))];
			nextOffset += result.posts.length;
			hasMore = result.hasMore && result.posts.length > 0 && nextOffset <= 100000;
			loaded = true;
		} catch (error) {
			if (!controller.signal.aborted) message = errorMessage(error);
		} finally {
			if (!controller.signal.aborted) busy = false;
		}
		if (hadFocus && loaded) {
			await tick();
			if (document.activeElement === trigger || document.activeElement === document.body) {
				const post = list?.children[previousCount] ?? list?.lastElementChild;
				post?.querySelector('h3')?.focus();
			}
		}
	}
</script>

<section aria-labelledby="author-posts-heading">
	<h2 id="author-posts-heading">Публикации</h2>
	{#if posts.length}
		<ul class="posts" {@attach rememberList}>
			{#each posts as post (post.id)}
				<li class="post">
					<div class="post-meta">
						{#if post.publishedAt}
							<time datetime={post.publishedAt}>{formatPostDate(post.publishedAt)}</time>
						{/if}
						{#if post.minimumTierId}
							<span class="post-access locked">
								<Icon name="lock" size={14} />
								<span class="sr-only">Уровень доступа:</span>
								{post.minimumTierName ?? 'По подписке'}
							</span>
						{:else}
							<span class="post-access">Публично</span>
						{/if}
					</div>
					<h3 tabindex="-1"><a href={resolve('/posts/[id]', { id: post.id })}>{post.title}</a></h3>
					{#if post.teaser}<p class="teaser">{post.teaser}</p>{/if}
				</li>
			{/each}
		</ul>
	{:else if busy}
		<p class="simple-intro" role="status">Загружаем публикации…</p>
	{:else if !hasMore}
		<p class="simple-intro">У автора пока нет публикаций.</p>
	{/if}
	{#if message}<p class="feedback error-feedback" role="alert">{message}</p>{/if}
	{#if hasMore && (posts.length || message)}
		<button
			class="button button-outline more"
			type="button"
			aria-disabled={busy}
			onclick={(event) => void loadPosts(event.currentTarget)}
		>
			{busy ? 'Загружаем…' : message ? 'Попробовать ещё раз' : 'Показать ещё'}
			{#if !message}<Icon name="chevron" size={16} />{/if}
		</button>
	{/if}
</section>

<style>
	h2 {
		font-size: clamp(23px, 3vw, 29px);
		line-height: 1.2;
		font-weight: 800;
		letter-spacing: -0.025em;
		margin-bottom: 16px;
	}
	.posts {
		padding: 0;
		margin: 0;
		list-style: none;
		border-top: 1px solid var(--line);
	}
	.post {
		padding-block: 20px;
		border-bottom: 1px solid var(--line);
		overflow-wrap: anywhere;
	}
	.post-meta {
		display: flex;
		align-items: baseline;
		flex-wrap: wrap;
		gap: 6px 14px;
		font-size: 13px;
		color: var(--muted);
		margin-bottom: 6px;
	}
	.post-access {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		font-weight: 700;
	}
	.locked {
		color: var(--blue-hover);
	}
	h3 {
		width: fit-content;
		font-size: clamp(18px, 2vw, 21px);
		font-weight: 800;
		line-height: 1.4;
		margin-bottom: 0;
	}
	h3 a:hover {
		color: var(--blue-hover);
		text-decoration: underline;
	}
	.teaser {
		max-width: 72ch;
		margin: 5px 0 0;
		white-space: pre-wrap;
		font-size: 15px;
		line-height: 1.65;
		color: var(--muted);
	}
	.more {
		margin-top: 18px;
	}
	.more[aria-disabled='true'] {
		cursor: wait;
		opacity: 0.65;
	}
</style>
