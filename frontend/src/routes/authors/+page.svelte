<script lang="ts">
	import { onMount } from 'svelte';
	import { resolve } from '$app/paths';
	import SiteHeader from '#lib/components/SiteHeader.svelte';
	import { api, errorMessage, type Profile } from '#lib/api/client.js';
	let query = $state('');
	let activeQuery = $state('');
	let authors = $state<Profile[]>([]);
	let busy = $state(true);
	let message = $state('');
	let hasMore = $state(false);
	let controller: AbortController | undefined;
	onMount(() => {
		void load();
		return () => controller?.abort();
	});
	async function load(more = false) {
		controller?.abort();
		const current = new AbortController();
		controller = current;
		busy = true;
		message = '';
		if (!more) {
			activeQuery = query.trim();
			authors = [];
		}
		try {
			const result = await api.authors(activeQuery, more ? authors.length : 0, current.signal);
			authors = more ? [...authors, ...result.authors] : result.authors;
			hasMore = result.hasMore;
		} catch (error) {
			if (!current.signal.aborted) message = errorMessage(error);
		} finally {
			if (!current.signal.aborted) busy = false;
		}
	}
</script>

<svelte:head
	><title>Найти авторов — etheyl</title><meta
		name="description"
		content="Найдите авторов на etheyl по имени, никнейму или описанию."
	/></svelte:head
>
<SiteHeader />
<main id="main" class="shell simple-page">
	<h1>Найдите тех,<br />чьё творчество вам близко</h1>
	<p class="simple-intro">Ищите по имени, никнейму или описанию страницы.</p>
	<form
		class="search-form"
		onsubmit={(event) => {
			event.preventDefault();
			void load();
		}}
	>
		<label for="author-search">Поиск авторов</label>
		<div>
			<input
				class="search-input"
				id="author-search"
				name="q"
				bind:value={query}
				maxlength="100"
				placeholder="Имя, никнейм или тема"
			/><button class="button button-primary" type="submit">Найти</button>
		</div>
	</form>
	<div class="results" aria-live="polite" aria-busy={busy}>
		{#if message}<p class="feedback error-feedback" role="alert">{message}</p>
			<button type="button" class="button button-outline" onclick={() => load(authors.length > 0)}
				>Попробовать ещё раз</button
			>{/if}
		{#if authors.length}<ul class="author-list">
				{#each authors as author (author.id)}<li>
						<a
							class="author-link"
							href={resolve('/authors/[nickname]', { nickname: author.nickname })}
							>{#if author.avatar}<img
									src={author.avatar.url}
									alt=""
									width="64"
									height="64"
								/>{:else}<span class="avatar-placeholder" aria-hidden="true"
									>{author.displayName.slice(0, 1)}</span
								>{/if}
							<div>
								<h2>{author.displayName}</h2>
								<span>@{author.nickname}</span>
								<p>{author.description}</p>
							</div></a
						>
					</li>{/each}
			</ul>{/if}
		{#if busy}<p class="status-note">
				Загружаем авторов…
			</p>{:else if !message && !authors.length}<div class="empty-state">
				<h2>{activeQuery ? 'Ничего не нашлось' : 'Здесь появятся авторы'}</h2>
				<p>
					{activeQuery
						? 'Попробуйте другое имя или более короткий запрос.'
						: 'Публичных страниц пока нет. Загляните позже.'}
				</p>
				<a href={resolve('/#registration')}>Создать аккаунт</a>
			</div>{/if}
	</div>
	{#if hasMore && !message}<button
			class="button button-outline more-button"
			type="button"
			disabled={busy}
			onclick={() => load(true)}>Показать ещё</button
		>{/if}
</main>

<style>
	.search-form {
		max-width: 660px;
		margin-block: 32px 42px;
	}
	.search-form label {
		display: block;
		font-size: 12px;
		font-weight: 700;
		margin-bottom: 8px;
	}
	.search-form > div {
		display: flex;
		gap: 12px;
	}
	.author-list {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 28px;
		list-style: none;
		padding: 0;
		margin: 0;
	}
	.author-link {
		display: flex;
		align-items: flex-start;
		gap: 20px;
		padding: 24px 0;
		border-top: 1px solid var(--line);
		text-decoration: none;
	}
	.author-link:hover h2 {
		color: var(--blue);
	}
	.author-link img,
	.avatar-placeholder {
		width: 64px;
		height: 64px;
		object-fit: cover;
		border-radius: 14px;
		flex-shrink: 0;
	}
	.avatar-placeholder {
		display: grid;
		place-items: center;
		background: var(--blue-soft);
		color: var(--blue);
		font-size: 24px;
		font-weight: 700;
	}
	.author-link > div {
		min-width: 0;
		overflow-wrap: anywhere;
	}
	.author-link h2 {
		font-size: 20px;
		font-weight: 800;
		margin: 0 0 5px;
	}
	.author-link span,
	.author-link p {
		color: var(--muted);
		font-size: 13px;
		line-height: 1.7;
	}
	.author-link p {
		margin: 12px 0 0;
		display: -webkit-box;
		line-clamp: 3;
		-webkit-line-clamp: 3;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}
	.empty-state {
		padding-block: 32px;
	}
	.empty-state h2 {
		font-size: 24px;
		font-weight: 700;
	}
	.empty-state p,
	.status-note {
		color: var(--muted);
		font-size: 14px;
	}
	.empty-state a {
		color: var(--blue);
		font-size: 14px;
	}
	.more-button {
		margin-top: 32px;
	}
	@media (max-width: 640px) {
		.author-list {
			grid-template-columns: 1fr;
			gap: 0;
		}
		.search-form > div {
			gap: 8px;
		}
		.search-form .button {
			padding-inline: 18px;
		}
	}
</style>
