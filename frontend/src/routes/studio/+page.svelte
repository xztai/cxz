<script lang="ts">
	import { invalidate } from '$app/navigation';
	import { resolve } from '$app/paths';
	import Icon from '#lib/components/Icon.svelte';
	import { formatPostDate } from '#lib/studio/format.js';
	import AnalyticsMetrics from './AnalyticsMetrics.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	let profile = $derived(data.access.state === 'ready' ? data.access.profile : null);
	let retrying = $state(false);

	async function retry() {
		retrying = true;
		try {
			await invalidate('studio:overview');
		} finally {
			retrying = false;
		}
	}
</script>

<svelte:head><title>Обзор студии — etheyl</title></svelte:head>

{#snippet setup(profileSaved: boolean, activeTier: boolean, published: boolean, draftId?: string)}
	{@const steps = [
		{
			label: 'Профиль автора',
			done: profileSaved,
			action: 'Создать профиль',
			description: 'Добавьте имя и описание, которые увидят посетители вашей страницы.',
			href: resolve('/studio/profile')
		},
		{
			label: 'Уровень подписки',
			done: activeTier,
			action: 'Добавить уровень',
			description:
				'Предложите подписку на свои работы. Открытый пост можно опубликовать без уровня.',
			href: resolve('/studio/tiers')
		},
		{
			label: 'Первая публикация',
			done: published,
			action: draftId ? 'Открыть черновик' : 'Написать пост',
			description: 'Опубликуйте первый пост на своей странице автора.',
			href: draftId
				? resolve('/studio/posts/[id]/edit', { id: draftId })
				: resolve('/studio/posts/new')
		}
	] as const}
	{@const next = steps.find((step) => !step.done)}
	{#if next}
		{@const completed = steps.filter((step) => step.done).length}
		<section class="setup" aria-labelledby="setup-heading">
			<div class="setup-copy">
				<h2 id="setup-heading">Настройте студию</h2>
				<div class="setup-progress">
					<progress value={completed} max={steps.length} aria-label="Настройка студии"></progress>
					<span>{completed} из {steps.length}</span>
				</div>
				<p>{next.description}</p>
				<a class="studio-link text-action" href={next.href}>
					{next.action}<Icon name="arrow" size={17} />
				</a>
			</div>
			<ol class="steps" aria-label="Шаги настройки">
				{#each steps as step, index (step.label)}
					<li class={{ next: step === next }} aria-current={step === next ? 'step' : undefined}>
						<span class={['step-marker', { done: step.done }]} aria-hidden="true">
							{#if step.done}<Icon name="check" size={16} />{:else}{index + 1}{/if}
						</span>
						<span>{step.label}</span>
						{#if step.done}<span class="sr-only">Выполнено</span>{/if}
					</li>
				{/each}
			</ol>
		</section>
	{/if}
{/snippet}

{#snippet loadError(message: string)}
	<div class="studio-panel load-error">
		<p role="status">{message}</p>
		<button class="button button-outline" type="button" disabled={retrying} onclick={retry}>
			{retrying ? 'Загружаем…' : 'Повторить'}
		</button>
	</div>
{/snippet}

{#snippet emptyPosts()}
	<div class="studio-panel empty-posts">
		<div class="post-icon"><Icon name="post" size={22} /></div>
		<div>
			<h3>Постов пока нет</h3>
			{#if profile}
				<p>Здесь будут ваши публикации и черновики.</p>
				<a class="studio-link text-action" href={resolve('/studio/posts/new')}>
					Создать первый пост<Icon name="arrow" size={17} />
				</a>
			{:else}
				<p>
					<a class="studio-link" href={resolve('/studio/profile')}>Сохраните профиль автора</a>,
					чтобы написать первый пост.
				</p>
			{/if}
		</div>
	</div>
{/snippet}

<div class="overview">
	<div class="overview-header">
		<h1 class="studio-heading">Обзор</h1>
		<div class="quick-actions">
			{#if profile}
				<a class="button button-primary" href={resolve('/studio/posts/new')}>
					<Icon name="plus" size={19} />Новый пост
				</a>
				<a
					class="button button-outline"
					href={resolve('/authors/[nickname]', { nickname: profile.nickname })}
				>
					Открыть профиль<Icon name="external" size={17} />
				</a>
			{:else}
				<a class="button button-primary" href={resolve('/studio/profile')}>
					Создать профиль<Icon name="arrow" size={19} />
				</a>
			{/if}
		</div>
	</div>

	{#if !profile}
		{@render setup(false, false, false)}
	{:else if data.progress}
		{#await data.progress then progress}
			{#if progress}
				{@render setup(true, progress.activeTier, progress.published, progress.draftId)}
			{:else}
				{@render loadError('Не удалось проверить шаги настройки.')}
			{/if}
		{/await}
	{/if}

	<AnalyticsMetrics analytics={data.analytics} dependency="studio:overview" />

	<section aria-labelledby="posts-heading">
		<div class="posts-heading">
			<h2 id="posts-heading">Последние посты</h2>
			<a class="studio-link text-action" href={resolve('/studio/posts')}
				>Все посты<Icon name="arrow" size={17} /></a
			>
		</div>
		{#if !data.latestPosts}
			{@render emptyPosts()}
		{:else}
			{#await data.latestPosts}
				<div class="studio-panel posts-loading" role="status">Загружаем посты…</div>
			{:then posts}
				{#if !posts}
					{@render loadError('Не удалось загрузить посты.')}
				{:else if posts.length === 0}
					{@render emptyPosts()}
				{:else}
					<ul class="post-list">
						{#each posts as post (post.id)}
							<li>
								<a href={resolve('/studio/posts/[id]/edit', { id: post.id })}>
									<span class="post-icon"><Icon name="post" size={19} /></span>
									<span class="post-title">{post.title}</span>
									<span class="post-meta">
										<span class={['post-status', { published: post.publishedAt !== null }]}>
											{post.publishedAt ? 'Опубликован' : 'Черновик'}
										</span>
										{#if post.publishedAt}
											<time datetime={post.publishedAt}>{formatPostDate(post.publishedAt)}</time>
										{/if}
									</span>
									<span class="post-arrow"><Icon name="arrow" size={18} /></span>
								</a>
							</li>
						{/each}
					</ul>
				{/if}
			{/await}
		{/if}
	</section>
</div>

<style>
	.overview {
		display: grid;
		gap: 26px;
	}
	.overview-header,
	.posts-heading {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px 20px;
		flex-wrap: wrap;
	}
	.quick-actions {
		display: flex;
		flex-wrap: wrap;
		gap: 10px;
	}
	.quick-actions .button {
		min-height: 44px;
		padding: 9px 16px;
		font-size: 14px;
	}
	.setup {
		display: grid;
		grid-template-columns: minmax(0, 1fr) minmax(250px, 0.8fr);
		gap: 24px;
		padding: 24px;
		border: 1px solid #dce4ff;
		border-radius: 16px;
		background: var(--blue-soft);
	}
	.setup h2,
	.posts-heading h2 {
		font-size: 22px;
		font-weight: 800;
		line-height: 1.3;
		letter-spacing: -0.025em;
	}
	.setup-progress {
		display: flex;
		align-items: center;
		gap: 10px;
		margin-top: 10px;
		color: var(--blue-hover);
		font-size: 13px;
		font-weight: 800;
		font-variant-numeric: tabular-nums;
	}
	progress {
		appearance: none;
		width: 104px;
		height: 6px;
		border: 0;
		border-radius: 4px;
		background: #d3defc;
	}
	progress::-webkit-progress-bar {
		border-radius: 4px;
		background: #d3defc;
	}
	progress::-webkit-progress-value {
		border-radius: 4px;
		background: var(--blue);
	}
	progress::-moz-progress-bar {
		border-radius: 4px;
		background: var(--blue);
	}
	.setup-copy p {
		max-width: 42ch;
		margin: 12px 0 9px;
		color: #444f73;
		font-size: 14px;
		line-height: 1.65;
	}
	.text-action {
		display: inline-flex;
		align-items: center;
		gap: 7px;
		min-height: 32px;
		font-size: 14px;
		text-decoration: none;
	}
	.steps {
		align-self: center;
		display: grid;
		gap: 4px;
		margin: 0;
		padding: 0 0 0 20px;
		border-inline-start: 1px solid #dce4ff;
		list-style: none;
	}
	.steps li {
		display: flex;
		align-items: center;
		gap: 11px;
		min-height: 42px;
		padding: 5px 9px;
		border-radius: 8px;
		color: #444f73;
		font-size: 14px;
		font-weight: 700;
	}
	.steps li.next {
		background: var(--white);
		color: var(--ink);
		font-weight: 800;
	}
	.step-marker {
		display: grid;
		flex: none;
		place-items: center;
		width: 25px;
		height: 25px;
		border: 1px solid #aebde9;
		border-radius: 50%;
		color: var(--blue-hover);
		font-size: 12px;
		font-weight: 800;
	}
	.next .step-marker {
		border-color: var(--blue);
		background: var(--blue);
		color: var(--white);
	}
	.step-marker.done {
		border-color: transparent;
		background: #dce4ff;
	}
	.posts-heading {
		margin-bottom: 12px;
	}
	.empty-posts {
		display: flex;
		align-items: flex-start;
		gap: 15px;
	}
	.empty-posts h3 {
		margin: 0 0 3px;
		font-size: 16px;
		font-weight: 800;
	}
	.empty-posts p,
	.load-error p,
	.posts-loading {
		color: var(--muted);
		font-size: 14px;
		line-height: 1.65;
	}
	.empty-posts p {
		margin-bottom: 4px;
	}
	.post-list {
		padding: 0;
		margin: 0;
		border: 1px solid var(--line);
		border-radius: 16px;
		background: var(--white);
		list-style: none;
	}
	.post-list li + li {
		border-top: 1px solid var(--line);
	}
	.post-list a {
		display: grid;
		grid-template-columns: auto minmax(0, 1fr) auto 18px;
		align-items: center;
		gap: 8px 13px;
		padding: 12px 18px;
		text-decoration: none;
	}
	.post-list li:first-child a {
		border-radius: 15px 15px 0 0;
	}
	.post-list li:last-child a {
		border-radius: 0 0 15px 15px;
	}
	.post-list li:only-child a {
		border-radius: 15px;
	}
	.post-list a:hover {
		background: var(--blue-soft);
	}
	.post-list a:hover .post-title {
		color: var(--blue-hover);
		text-decoration: underline;
	}
	.post-icon {
		display: grid;
		place-items: center;
		flex: none;
		width: 38px;
		height: 38px;
		border-radius: 9px;
		background: var(--surface);
		color: var(--muted);
	}
	.post-title {
		font-size: 15px;
		font-weight: 800;
		overflow-wrap: anywhere;
	}
	.post-meta {
		display: flex;
		align-items: center;
		justify-content: flex-end;
		gap: 8px 12px;
		color: var(--muted);
		font-size: 12px;
		font-variant-numeric: tabular-nums;
	}
	.post-status {
		border-radius: 6px;
		padding: 3px 7px;
		background: var(--surface);
		color: var(--muted);
		font-weight: 750;
	}
	.post-status.published {
		background: var(--blue-soft);
		color: var(--blue-hover);
	}
	.post-arrow {
		color: var(--blue-hover);
	}
	.load-error {
		display: flex;
		align-items: center;
		justify-content: space-between;
		flex-wrap: wrap;
		gap: 14px 24px;
	}
	.load-error button {
		min-height: 44px;
		padding-block: 8px;
	}
	@media (max-width: 680px) {
		.overview {
			gap: 24px;
		}
		.setup {
			grid-template-columns: minmax(0, 1fr);
			gap: 13px;
			padding: 18px;
		}
		.steps {
			border-inline-start: 0;
			border-top: 1px solid #dce4ff;
			padding: 12px 0 0;
		}
		.post-list a {
			grid-template-columns: 38px minmax(0, 1fr) 18px;
			gap: 3px 10px;
			padding: 12px;
		}
		.post-list .post-icon,
		.post-arrow {
			grid-row: span 2;
		}
		.post-arrow {
			grid-column: 3;
			grid-row: 1 / span 2;
		}
		.post-meta {
			grid-column: 2;
			justify-content: flex-start;
			flex-wrap: wrap;
		}
	}
</style>
