<script lang="ts">
	import { onMount } from 'svelte';
	import { resolve } from '$app/paths';
	import SiteHeader from '#lib/components/SiteHeader.svelte';
	import { api, ApiError, errorMessage, type Profile } from '#lib/api/client.js';
	import type { PageProps } from './$types';
	let { data }: PageProps = $props();
	let profile = $state<Profile | null>(null);
	let busy = $state(true);
	let message = $state('');
	let controller: AbortController | undefined;
	onMount(() => {
		void load();
		return () => controller?.abort();
	});
	async function load() {
		controller?.abort();
		controller = new AbortController();
		busy = true;
		message = '';
		try {
			profile = (await api.author(data.nickname, controller.signal)).profile;
		} catch (error) {
			if (!controller.signal.aborted)
				message =
					error instanceof ApiError && error.status === 404
						? 'Страница автора не найдена.'
						: errorMessage(error);
		} finally {
			busy = false;
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
			<button class="button button-outline" type="button" onclick={load}>Попробовать ещё раз</button
			>{:else if profile}{#if profile.cover}<img
					class="cover"
					src={profile.cover.url}
					alt="Обложка страницы автора"
				/>{/if}
			<div class="profile-heading">
				{#if profile.avatar}<img
						class="avatar"
						src={profile.avatar.url}
						alt=""
						width="88"
						height="88"
					/>{/if}
				<div>
					<h1>{profile.displayName}</h1>
					<p class="nickname">@{profile.nickname}</p>
				</div>
			</div>
			<p class="description">{profile.description}</p>{/if}
	</div>
</main>

<style>
	.back-link {
		display: inline-block;
		color: var(--blue);
		font-size: 13px;
		margin-bottom: 30px;
	}
	.cover {
		width: 100%;
		max-height: 320px;
		object-fit: cover;
		border-radius: 16px;
		margin-bottom: 30px;
	}
	.profile-heading {
		display: flex;
		align-items: center;
		gap: 24px;
		overflow-wrap: anywhere;
	}
	.profile-heading h1 {
		margin-bottom: 10px;
	}
	.avatar {
		width: 88px;
		height: 88px;
		object-fit: cover;
		border-radius: 16px;
	}
	.nickname {
		color: var(--muted);
		font-size: 14px;
	}
	.description {
		white-space: pre-wrap;
		overflow-wrap: anywhere;
		max-width: 720px;
		font-size: 16px;
		line-height: 1.8;
		margin-top: 32px;
	}
</style>
