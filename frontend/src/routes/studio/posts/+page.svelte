<script lang="ts">
	import { afterNavigate, goto, invalidate } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { tick, untrack } from 'svelte';
	import { api, errorMessage, type OwnPostStatus, type OwnPostSummary } from '#lib/api/client.js';
	import Icon from '#lib/components/Icon.svelte';
	import { studioEditorIsActive } from '#lib/session.svelte.js';
	import { formatPostDate } from '#lib/studio/format.js';
	import { maxPage, postsDependency, postsHref } from './listing.js';
	import type { PageProps } from './$types';

	const tabs = [
		{ status: 'draft', label: 'Черновики' },
		{ status: 'published', label: 'Опубликованные' }
	] as const;
	const actions = {
		publish: {
			request: api.publishPost,
			label: 'Опубликовать',
			pending: 'Публикуем…',
			success: 'Пост опубликован.'
		},
		unpublish: {
			request: api.unpublishPost,
			label: 'Снять',
			pending: 'Снимаем…',
			success: 'Пост снят с публикации.'
		},
		delete: {
			request: api.deletePost,
			label: 'Удалить',
			pending: 'Удаляем…',
			success: 'Пост удалён.'
		}
	};
	type PostAction = keyof typeof actions;

	let { data }: PageProps = $props();
	const ownerId = untrack(() => (data.access.state === 'ready' ? data.access.ownerId : ''));
	const isActive = studioEditorIsActive(ownerId);
	const hasProfile = $derived(data.access.state === 'ready' && !!data.access.profile);
	const isDraft = $derived(data.status === 'draft');
	const transition = $derived(isDraft ? 'publish' : 'unpublish');
	let removed = $state.raw<{ id: string; listing: PageProps['data']['listing'] }>();
	const posts = $derived(
		data.listing?.posts.filter(
			(post) => removed?.listing !== data.listing || post.id !== removed.id
		) ?? []
	);
	const hasNextPage = $derived(!!data.listing?.hasMore && data.pageNumber < maxPage);
	let heading: HTMLHeadingElement;
	let dialog: HTMLDialogElement;
	let deleteTrigger: HTMLButtonElement | undefined;
	let deleteTarget = $state.raw<OwnPostSummary>();
	let pending = $state<{ id: string; action: PostAction }>();
	let rowErrors = $state<Record<string, string | undefined>>({});
	let confirmation = $state('');
	let retrying = $state(false);

	afterNavigate(({ from, to, type }) => {
		if ((type === 'link' || type === 'popstate') && from?.url.search !== to?.url.search) {
			rowErrors = {};
			confirmation = '';
			if (deleteTarget) closeDialog(false);
		}
	});

	async function retry() {
		const { status, pageNumber } = data;
		retrying = true;
		try {
			await invalidate(postsDependency);
			await tick();
			if (confirmation) await returnToPreviousIfEmpty(status, pageNumber);
			if (data.status === status && data.listing && isActive()) heading.focus();
		} finally {
			retrying = false;
		}
	}

	async function returnToPreviousIfEmpty(status: OwnPostStatus, pageNumber: number) {
		if (
			pageNumber > 1 &&
			data.status === status &&
			data.pageNumber === pageNumber &&
			data.listing?.posts.length === 0 &&
			isActive()
		) {
			await goto(postsHref(status, pageNumber - 1), { replace: true, reset: false });
		}
	}

	function openDelete(post: OwnPostSummary, trigger: HTMLButtonElement) {
		if (pending || !isActive()) return;
		deleteTrigger = trigger;
		deleteTarget = post;
	}

	function openDialog(element: HTMLDialogElement) {
		dialog = element;
		element.showModal();
		return () => {
			if (element.open) element.close();
		};
	}

	function closeDialog(restoreFocus = true) {
		dialog?.close();
		deleteTarget = undefined;
		if (restoreFocus && deleteTrigger?.isConnected) deleteTrigger.focus({ preventScroll: true });
		deleteTrigger = undefined;
	}

	async function runAction(action: PostAction, post: OwnPostSummary) {
		if (pending || !isActive()) return;
		const { status, pageNumber, listing } = data;
		const onSamePage = () => data.status === status && data.pageNumber === pageNumber;
		const detail = actions[action];
		pending = { id: post.id, action };
		rowErrors[post.id] = undefined;
		confirmation = '';

		try {
			try {
				await detail.request(ownerId, post.id);
			} catch (error) {
				if (isActive() && onSamePage()) rowErrors[post.id] = errorMessage(error);
				return;
			} finally {
				if (action === 'delete' && deleteTarget && isActive()) closeDialog();
			}
			if (!isActive()) return;

			const focusWasInRow =
				document.activeElement?.closest('[data-post-id]')?.getAttribute('data-post-id') === post.id;
			if (data.status === status) confirmation = detail.success;
			removed = { id: post.id, listing };
			await tick();
			if (focusWasInRow && isActive() && onSamePage()) heading.focus();

			await invalidate((url) => url.href === postsDependency || url.href === 'studio:overview');
			await tick();
			await returnToPreviousIfEmpty(status, pageNumber);
		} finally {
			pending = undefined;
		}
	}

	function submitDelete(event: SubmitEvent) {
		event.preventDefault();
		if (deleteTarget) void runAction('delete', deleteTarget);
	}
</script>

<svelte:head><title>Посты — etheyl</title></svelte:head>

<div class="heading">
	<h1
		class="studio-heading"
		tabindex="-1"
		{@attach (element) => {
			heading = element;
		}}
	>
		Посты
	</h1>
	{#if hasProfile}
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
{:else}
	<nav class="tabs" aria-label="Статус постов">
		{#each tabs as tab (tab.status)}
			<a
				href={postsHref(tab.status, 1)}
				aria-current={data.status === tab.status ? 'page' : undefined}
			>
				{tab.label}
			</a>
		{/each}
	</nav>

	<div role="status">
		{#if confirmation}
			<p class="notice"><Icon name="check" size={18} /> {confirmation}</p>
		{/if}
	</div>

	{#if !data.listing}
		<div class="studio-panel list-message">
			<p class="error-feedback" role="alert">Не удалось загрузить посты.</p>
			<button class="button button-outline" type="button" disabled={retrying} onclick={retry}>
				{retrying ? 'Загружаем…' : 'Повторить'}
			</button>
		</div>
	{:else}
		{#if posts.length}
			<div class="post-list">
				<div role="table" aria-label={isDraft ? 'Черновики' : 'Опубликованные'}>
					<div class="post-columns" role="row">
						<span role="columnheader">Пост</span>
						<span role="columnheader">Доступ</span>
						<span role="columnheader">{isDraft ? 'Изменён' : 'Опубликован'}</span>
						<span role="columnheader">Действия</span>
					</div>
					<div role="rowgroup">
						{#each posts as post (post.id)}
							{@const date = isDraft ? post.updatedAt : post.publishedAt}
							{@const editHref = resolve('/studio/posts/[id]/edit', { id: post.id })}
							<div class="post-row" role="row" data-post-id={post.id}>
								<div role="rowheader" class="post-title">
									<a href={editHref}>{post.title}</a>
								</div>
								<div role="cell" class="access">{post.minimumTierName ?? 'Для всех'}</div>
								<div role="cell" class="date">
									{#if date}
										<time datetime={date}>{formatPostDate(date)}</time>
									{/if}
								</div>
								<div role="cell" class="post-actions">
									<div class="actions">
										<a class="row-action" href={editHref}>Редактировать</a>
										<button
											class="row-action publish-action"
											type="button"
											aria-disabled={!!pending}
											onclick={() => runAction(transition, post)}
										>
											{pending?.id === post.id && pending.action === transition
												? actions[transition].pending
												: actions[transition].label}
										</button>
										<button
											class="row-action danger"
											type="button"
											aria-disabled={!!pending}
											aria-haspopup="dialog"
											onclick={(event) => openDelete(post, event.currentTarget)}
										>
											Удалить
										</button>
									</div>
									{#if rowErrors[post.id]}
										<p class="feedback error-feedback row-error" role="alert">
											{rowErrors[post.id]}
										</p>
									{/if}
								</div>
							</div>
						{/each}
					</div>
				</div>
			</div>
		{:else}
			<div class="studio-panel list-message">
				{#if pending && data.pageNumber > 1}
					<p>Обновляем список…</p>
				{:else if data.pageNumber > 1}
					<p>На этой странице нет постов.</p>
				{:else if isDraft}
					<p>Черновиков пока нет</p>
				{:else}
					<p>
						<strong>Опубликованных постов пока нет.</strong> Опубликуйте черновик, чтобы он появился здесь
					</p>
				{/if}
			</div>
		{/if}
		{#if data.pageNumber > 1 || hasNextPage}
			<nav class="pagination" aria-label="Страницы постов">
				{#if data.pageNumber > 1}
					<a
						class="previous button button-outline"
						href={postsHref(data.status, data.pageNumber - 1)}
					>
						Предыдущая
					</a>
				{/if}
				<span>Страница {data.pageNumber}</span>
				{#if hasNextPage}
					<a class="next button button-outline" href={postsHref(data.status, data.pageNumber + 1)}>
						Следующая
					</a>
				{/if}
			</nav>
		{/if}
	{/if}
{/if}

{#if deleteTarget}
	<dialog
		{@attach openDialog}
		aria-labelledby="delete-post-title"
		aria-describedby="delete-post-description"
		oncancel={(event) => {
			event.preventDefault();
			if (!pending) closeDialog();
		}}
	>
		<form aria-busy={!!pending} onsubmit={submitDelete}>
			<h2 id="delete-post-title">Удалить пост?</h2>
			<p id="delete-post-description">
				Пост <strong>«{deleteTarget.title}»</strong> будет удалён безвозвратно.
			</p>
			<div class="dialog-actions">
				<button
					class="button button-outline"
					type="button"
					aria-disabled={!!pending}
					onclick={() => {
						if (!pending) closeDialog();
					}}
				>
					Отмена
				</button>
				<button class="button button-danger" type="submit" aria-disabled={!!pending}>
					{pending ? actions.delete.pending : actions.delete.label}
				</button>
			</div>
		</form>
	</dialog>
{/if}

<style>
	.heading {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		justify-content: space-between;
		gap: 12px 24px;
		margin-block-end: 16px;
	}

	.tabs {
		display: flex;
		flex-wrap: wrap;
		gap: 0 18px;
		border-block-end: 1px solid var(--line);
		margin-block-end: 16px;
	}

	.tabs a {
		display: flex;
		align-items: center;
		min-height: 44px;
		margin-block-end: -1px;
		padding: 10px 4px 8px;
		border-block-end: 2px solid transparent;
		color: var(--muted);
		font-size: 14px;
		font-weight: 750;
		text-decoration: none;
	}

	.tabs a:hover,
	.tabs a[aria-current='page'] {
		color: var(--blue-hover);
	}

	.tabs a[aria-current='page'] {
		border-block-end-color: var(--blue);
	}

	.notice {
		display: flex;
		align-items: center;
		gap: 8px;
		margin-block: 0 12px;
		padding: 10px 14px;
		border-radius: 8px;
		background: var(--blue-soft);
		color: var(--blue-hover);
		font-size: 14px;
		font-weight: 700;
	}

	.list-message p {
		margin-block: 0;
		font-size: 15px;
		line-height: 1.65;
	}

	.list-message .button {
		margin-block-start: 12px;
	}

	.post-list {
		container-type: inline-size;
		border: 1px solid var(--line);
		border-radius: 16px;
		background: var(--white);
	}

	.post-columns,
	.post-row {
		display: grid;
		grid-template-columns: minmax(0, 1fr) 20% 128px 316px;
		align-items: center;
	}

	.post-columns > *,
	.post-row > * {
		padding: 10px 12px 10px 0;
		overflow-wrap: anywhere;
	}

	.post-columns > :first-child,
	.post-row > :first-child {
		padding-inline-start: 20px;
	}

	.post-columns > :last-child,
	.post-row > :last-child {
		padding-inline-end: 20px;
	}

	.post-columns {
		border-block-end: 1px solid var(--line);
		color: var(--muted);
		font-size: 13px;
		font-weight: 750;
	}

	.post-row + .post-row {
		border-block-start: 1px solid var(--line);
	}

	.post-title {
		font-size: 15px;
		font-weight: 800;
		line-height: 1.45;
	}

	.post-title a:hover {
		color: var(--blue-hover);
		text-decoration: underline;
	}

	.access,
	.date {
		color: var(--muted);
		font-size: 13px;
		line-height: 1.5;
	}

	.date {
		font-variant-numeric: tabular-nums;
	}

	.actions {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 2px;
	}

	.row-action {
		display: inline-flex;
		align-items: center;
		min-height: 44px;
		padding-inline: 6px;
		border-radius: 8px;
		font-size: 13px;
		font-weight: 750;
		white-space: nowrap;
		text-decoration: none;
		cursor: pointer;
	}

	.row-action:hover:not([aria-disabled='true']) {
		background: var(--surface);
		text-decoration: underline;
	}

	.row-action.publish-action {
		color: var(--blue-hover);
	}

	.row-action.danger {
		color: var(--error);
	}

	.row-action[aria-disabled='true'],
	.dialog-actions [aria-disabled='true'] {
		opacity: 0.65;
		cursor: wait;
	}

	.row-error {
		margin: 4px 0 0;
	}

	.pagination {
		display: grid;
		grid-template-columns: 1fr auto 1fr;
		align-items: center;
		gap: 8px;
		margin-block-start: 16px;
	}

	.pagination span {
		grid-area: 1 / 2;
		color: var(--muted);
		font-size: 13px;
		font-variant-numeric: tabular-nums;
	}

	.pagination .button {
		min-height: 44px;
		padding: 8px 12px;
		font-size: 13px;
	}

	.previous {
		grid-area: 1 / 1;
		justify-self: start;
	}

	.next {
		grid-area: 1 / 3;
		justify-self: end;
	}

	dialog {
		inset: 0;
		width: min(480px, calc(100% - 32px));
		max-height: calc(100% - 32px);
		margin: auto;
		padding: 24px;
		border: 1px solid var(--line);
		border-radius: 16px;
		background: var(--white);
		color: var(--ink);
		overflow-wrap: anywhere;
	}

	dialog::backdrop {
		background: #171a246e;
	}

	dialog h2 {
		margin-block: 0 12px;
		font-size: 22px;
		font-weight: 800;
		line-height: 1.3;
		letter-spacing: -0.025em;
	}

	dialog p {
		font-size: 15px;
		line-height: 1.6;
	}

	.dialog-actions {
		display: flex;
		flex-wrap: wrap;
		justify-content: flex-end;
		gap: 8px;
		margin-block-start: 24px;
	}

	.button-danger {
		background: var(--error);
		color: var(--white);
	}

	.button-danger:hover:not([aria-disabled='true']) {
		background: #8e1d2c;
	}

	@container (max-width: 740px) {
		.post-columns {
			position: absolute;
			width: 1px;
			height: 1px;
			overflow: hidden;
			clip-path: inset(50%);
			white-space: nowrap;
		}

		.post-row {
			grid-template-columns: minmax(0, 1fr) auto;
			align-items: baseline;
			gap: 6px 12px;
			padding: 14px 16px 8px;
		}

		.post-row > *,
		.post-row > :first-child,
		.post-row > :last-child {
			padding: 0;
		}

		.post-title,
		.post-actions {
			grid-column: 1 / -1;
		}

		.actions {
			margin-inline: -6px;
		}
	}

	@media (max-width: 440px) {
		.heading .button {
			min-height: 44px;
			padding: 9px 12px;
			gap: 6px;
		}

		.pagination {
			column-gap: 4px;
		}

		.pagination .button {
			padding-inline: 6px;
		}

		dialog {
			padding: 20px;
		}

		.dialog-actions .button {
			flex: 1;
			padding-inline: 12px;
		}
	}
</style>
