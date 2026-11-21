<script lang="ts">
	import { resolve } from '$app/paths';
	import AuthorProfileHeader from '#lib/components/AuthorProfileHeader.svelte';
	import SiteHeader from '#lib/components/SiteHeader.svelte';
	import { api, ApiError, errorMessage } from '#lib/api/client.js';
	import AuthorPosts from './AuthorPosts.svelte';
	import AuthorSubscriptions from './AuthorSubscriptions.svelte';
	import type { PageProps } from './$types';
	let { data }: PageProps = $props();
	let author = $state.raw<Awaited<ReturnType<typeof api.author>> | null>(null);
	let busy = $state(true);
	let message = $state('');
	let retryCount = $state(0);

	$effect(() => {
		const nickname = data.nickname;
		// A retry reloads the profile even when the nickname is unchanged.
		void retryCount;
		const controller = new AbortController();
		author = null;
		message = '';
		busy = true;
		void load(nickname, controller.signal);
		return () => controller.abort();
	});

	async function load(nickname: string, signal: AbortSignal) {
		try {
			const result = await api.author(nickname, signal);
			if (!signal.aborted) author = result;
		} catch (error) {
			if (!signal.aborted)
				message =
					error instanceof ApiError && error.status === 404
						? 'Страница автора не найдена.'
						: errorMessage(error);
		} finally {
			if (!signal.aborted) busy = false;
		}
	}
</script>

<svelte:head><title>{author?.profile.displayName ?? 'Страница автора'} — etheyl</title></svelte:head
>
<SiteHeader />
<main id="main" class="shell simple-page author-page">
	<a class="back-link" href={resolve('/authors')}>Все авторы</a>
	<div aria-live="polite" aria-busy={busy}>
		{#if busy}
			<p class="simple-intro">Загружаем страницу…</p>
		{:else if message}
			<h1>Не удалось открыть страницу</h1>
			<p class="feedback error-feedback">{message}</p>
			<button class="button button-outline" type="button" onclick={() => retryCount++}>
				Попробовать ещё раз
			</button>
		{:else if author}
			<AuthorProfileHeader profile={author.profile} publicPage />
		{/if}
	</div>
	{#if author}
		{#key author.profile.nickname}
			<div class="author-content">
				<AuthorSubscriptions profile={author.profile} tiers={author.tiers} />
				<AuthorPosts nickname={author.profile.nickname} />
			</div>
		{/key}
	{/if}
</main>

<style>
	.author-page {
		max-width: 1120px;
		padding-top: 24px;
	}
	.back-link {
		display: inline-flex;
		align-items: center;
		min-height: 44px;
		color: var(--blue-hover);
		font-size: 14px;
		font-weight: 700;
		margin-bottom: 8px;
	}
	.back-link:hover {
		text-decoration: underline;
	}
	.author-content {
		display: grid;
		gap: 48px;
		margin-top: 32px;
		padding-top: 28px;
		border-top: 1px solid var(--line);
	}
	@media (max-width: 640px) {
		.author-page {
			padding-top: 12px;
		}
		.author-content {
			gap: 36px;
			margin-top: 24px;
			padding-top: 24px;
		}
	}
</style>
