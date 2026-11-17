<script lang="ts">
	import { invalidate } from '$app/navigation';
	import { resolve } from '$app/paths';
	import Icon from '#lib/components/Icon.svelte';
	import { formatPostDate } from '#lib/studio/format.js';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const hasProfile = $derived(data.access.state === 'ready' && !!data.access.profile);
	let retrying = $state(false);

	async function retry() {
		retrying = true;
		try {
			await invalidate('studio:posts');
		} finally {
			retrying = false;
		}
	}
</script>

<svelte:head><title>Посты — etheyl</title></svelte:head>

<div class="heading">
	<h1 class="studio-heading">Посты</h1>
	{#if hasProfile && (!data.listing || data.offset || data.listing.posts.length)}
		<a class="button button-primary" href={resolve('/studio/posts/new')}>
			<Icon name="plus" size={18} /> Новый пост
		</a>
	{/if}
</div>

{#if !hasProfile}
	<div class="studio-panel">
		<p class="studio-copy">Посты можно создавать после сохранения профиля автора.</p>
		<a class="button button-primary" href={resolve('/studio/profile')}>Создать профиль</a>
	</div>
{:else if !data.listing}
	<div class="studio-panel">
		<p class="feedback error-feedback" role="alert">Не удалось загрузить посты.</p>
		<button class="button button-outline" type="button" disabled={retrying} onclick={retry}>
			{retrying ? 'Загружаем…' : 'Повторить'}
		</button>
	</div>
{:else}
	{#if data.listing.posts.length}
		<ul class="studio-panel posts">
			{#each data.listing.posts as post (post.id)}
				<li>
					<a class="post" href={resolve('/studio/posts/[id]/edit', { id: post.id })}>
						<strong>{post.title}</strong>
						<div class="post-meta">
							<span class={{ published: post.publishedAt }}
								>{post.publishedAt ? 'Опубликован' : 'Черновик'}</span
							>
							{#if post.publishedAt}
								<time datetime={post.publishedAt}>{formatPostDate(post.publishedAt)}</time>
							{/if}
						</div>
					</a>
				</li>
			{/each}
		</ul>
	{:else}
		<div class="studio-panel">
			<p class="studio-copy">
				{data.offset ? 'На этой странице нет постов.' : 'У вас пока нет постов.'}
			</p>
			{#if !data.offset}
				<a class="button button-primary" href={resolve('/studio/posts/new')}>
					<Icon name="plus" size={18} /> Новый пост
				</a>
			{/if}
		</div>
	{/if}
	{@const nextOffset = data.offset + data.listing.posts.length}
	{@const hasNextPage = data.listing.hasMore && nextOffset <= 100000 && nextOffset > data.offset}
	{#if data.offset || hasNextPage}
		<nav class="pagination" aria-label="Страницы постов">
			{#if data.offset}
				<a
					class="button button-outline"
					href={resolve(`/studio/posts?offset=${Math.max(0, data.offset - 24)}`)}
				>
					Предыдущие
				</a>
			{/if}
			{#if hasNextPage}
				<a class="button button-outline" href={resolve(`/studio/posts?offset=${nextOffset}`)}>
					Следующие
				</a>
			{/if}
		</nav>
	{/if}
{/if}

<style>
	.heading {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		justify-content: space-between;
		gap: 12px 24px;
		margin-block-end: 24px;
	}

	.heading h1 {
		margin: 0;
	}

	.posts.studio-panel {
		list-style: none;
		padding-block: 0;
	}

	.posts li + li {
		border-block-start: 1px solid var(--line);
	}

	.post {
		display: flex;
		align-items: baseline;
		justify-content: space-between;
		flex-wrap: wrap;
		gap: 8px 24px;
		padding-block: 16px;
		overflow-wrap: anywhere;
		text-decoration: none;
	}

	.post:hover strong {
		color: var(--blue-hover);
		text-decoration: underline;
	}

	.post-meta {
		display: flex;
		align-items: baseline;
		flex-wrap: wrap;
		gap: 8px 12px;
		color: var(--muted);
		font-size: 13px;
	}

	.post-meta span {
		border-radius: 6px;
		padding: 3px 8px;
		background: var(--surface);
		font-weight: 700;
	}

	.post-meta .published {
		background: var(--blue-soft);
		color: var(--blue-hover);
	}

	.pagination {
		display: flex;
		flex-wrap: wrap;
		gap: 12px;
		margin-block-start: 20px;
	}
</style>
