<script lang="ts">
	import { resolve } from '$app/paths';
	import AuthorProfileHeader from '#lib/components/AuthorProfileHeader.svelte';
	import SiteHeader from '#lib/components/SiteHeader.svelte';
	import { api, ApiError, errorMessage, type Profile } from '#lib/api/client.js';
	import type { PageProps } from './$types';
	let { data }: PageProps = $props();
	let profile = $state<Profile | null>(null);
	let busy = $state(true);
	let message = $state('');
	let retryCount = $state(0);

	$effect(() => {
		const nickname = data.nickname;
		// A retry reloads the profile even when the nickname is unchanged.
		void retryCount;
		const controller = new AbortController();
		profile = null;
		message = '';
		busy = true;
		void load(nickname, controller.signal);
		return () => controller.abort();
	});

	async function load(nickname: string, signal: AbortSignal) {
		try {
			const result = await api.author(nickname, signal);
			if (!signal.aborted) profile = result.profile;
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

<svelte:head><title>{profile?.displayName ?? 'Страница автора'} — etheyl</title></svelte:head>
<SiteHeader />
<main id="main" class="shell simple-page">
	<a class="back-link" href={resolve('/authors')}>Все авторы</a>
	<div aria-live="polite" aria-busy={busy}>
		{#if busy}<p>Загружаем страницу…</p>{:else if message}<h1>Не удалось открыть страницу</h1>
			<p class="feedback error-feedback">{message}</p>
			<button class="button button-outline" type="button" onclick={() => retryCount++}
				>Попробовать ещё раз</button
			>{:else if profile}
			<AuthorProfileHeader {profile} />
		{/if}
	</div>
</main>

<style>
	.back-link {
		display: inline-block;
		color: var(--blue);
		font-size: 13px;
		margin-bottom: 30px;
	}
</style>
