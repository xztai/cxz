<script lang="ts">
	import { resolve } from '$app/paths';
	import { onMount, tick } from 'svelte';
	import {
		api,
		ApiError,
		errorMessage,
		type Comment,
		type InteractionSummary
	} from '#lib/api/client.js';
	import Icon from '#lib/components/Icon.svelte';
	import { formatPostDate } from '#lib/studio/format.js';

	let {
		postId,
		canRead,
		signedIn,
		onAccessLost
	}: {
		postId: string;
		canRead: boolean;
		signedIn: boolean;
		onAccessLost: (status: number) => void;
	} = $props();
	let available = $derived(canRead && signedIn);
	let loginQuery = $derived(new URLSearchParams({ post: postId }));
	const limit = 2000;
	const controller = new AbortController();
	const signal = controller.signal;
	let summary = $state.raw<InteractionSummary | null>(null);
	let comments = $state.raw<Comment[]>([]);
	let before = $state<string | null>(null);
	let draft = $state('');
	let length = $derived(Array.from(draft).length);
	let tooLong = $derived(length > limit);
	type Action = 'load' | 'older' | 'like' | 'create' | 'delete';
	let pending = $state<Action | null>(null);
	let issue = $state<{ action: Action; text: string } | null>(null);

	onMount(() => {
		if (available) void loadComments();
		return () => controller.abort();
	});

	function focusComments(heading: HTMLHeadingElement) {
		if (window.location.hash === '#comments') heading.focus();
	}

	async function run(action: Action, perform: () => Promise<void>) {
		if (pending || !available) return;
		issue = null;
		pending = action;
		try {
			await perform();
		} catch (error) {
			if (signal.aborted) return;
			const status = error instanceof ApiError ? error.status : 0;
			const code = error instanceof ApiError ? error.code : '';
			if (status === 401 || code === 'post_access_required') onAccessLost(status);
			issue = {
				action,
				text:
					status === 401
						? 'Войдите, чтобы продолжить обсуждение.'
						: code === 'post_access_required'
							? 'Для обсуждения нужен доступ к публикации.'
							: errorMessage(error)
			};
		} finally {
			if (!signal.aborted) pending = null;
		}
	}

	async function loadComments(trigger?: HTMLButtonElement) {
		const cursor = before ?? undefined;
		await run(cursor ? 'older' : 'load', async () => {
			const result = await api.discussion(postId, signal, cursor);
			signal.throwIfAborted();
			summary = result.summary;
			const displayed = new Set(comments.map(({ id }) => id));
			const added = result.comments.filter(({ id }) => !displayed.has(id));
			comments = [...comments, ...added];
			before = result.nextCursor;
			if (trigger) {
				await tick();
				if (document.activeElement === trigger || document.activeElement === document.body) {
					const target = added[0] ? `comment-${added[0].id}` : 'comments';
					document.getElementById(target)?.focus();
				}
			}
		});
	}

	async function toggleLike() {
		if (!summary) return;
		const liked = !summary.liked;
		await run('like', async () => {
			const result = await api.setPostLike(postId, liked, signal);
			signal.throwIfAborted();
			summary = result.summary;
		});
	}

	async function submitComment(event: SubmitEvent) {
		event.preventDefault();
		const body = draft.trim();
		if (!body || tooLong || !summary) return;
		await run('create', async () => {
			const result = await api.createComment(postId, body, signal);
			signal.throwIfAborted();
			summary = result.summary;
			comments = [result.comment, ...comments];
			draft = '';
		});
	}

	async function deleteComment(comment: Comment, trigger: HTMLButtonElement) {
		if (!comment.own) return;
		const index = comments.findIndex(({ id }) => id === comment.id);
		await run('delete', async () => {
			await api.deleteComment(postId, comment.id, signal);
			signal.throwIfAborted();
			const hadFocus = document.activeElement === trigger;
			comments = comments.filter(({ id }) => id !== comment.id);
			if (summary) summary = { ...summary, commentCount: Math.max(0, summary.commentCount - 1) };
			if (hadFocus) {
				await tick();
				if (document.activeElement === document.body) {
					const adjacent = comments[index] ?? comments[index - 1];
					document
						.getElementById(adjacent ? `comment-${adjacent.id}` : 'comment-body')
						?.focus({ preventScroll: true });
				}
			}
		});
	}
</script>

<div class="actions">
	{#if !signedIn}
		<a class="action" href="{resolve('/login')}?{loginQuery}"><Icon name="heart" /> Нравится</a>
	{:else}
		<button
			class="action like"
			type="button"
			aria-pressed={available && summary?.liked === true}
			aria-labelledby="like-label"
			aria-describedby={available && summary ? 'like-count' : undefined}
			aria-disabled={!available || !summary || pending !== null}
			onclick={() => void toggleLike()}
		>
			<Icon name="heart" />
			<span id="like-label">Нравится</span>
			{#if available && summary}<span class="count" id="like-count">{summary.likeCount}</span>{/if}
		</button>
	{/if}
	<a class="action" href="#comments">
		<Icon name="comment" /> Комментарии
		{#if available && summary}<span class="count">{summary.commentCount}</span>{/if}
	</a>
</div>
{#if available && issue?.action === 'like'}
	<p class="feedback error-feedback" role="alert">{issue.text}</p>
{/if}

<section class="discussion" aria-labelledby="comments">
	<h2 id="comments" tabindex="-1" {@attach focusComments}>Комментарии</h2>
	{#if !signedIn}
		<p class="simple-intro guest">
			<a href="{resolve('/login')}?{loginQuery}">Войдите</a>{canRead
				? ', чтобы читать и оставлять комментарии.'
				: ' в аккаунт с доступом к публикации, чтобы участвовать в обсуждении.'}
		</p>
	{:else if !canRead}
		<p class="simple-intro">Для обсуждения нужен доступ к публикации.</p>
	{:else if !summary}
		{#if issue}
			<p class="feedback error-feedback" role="alert">{issue.text}</p>
			<button class="button button-outline" type="button" onclick={() => void loadComments()}
				>Попробовать ещё раз</button
			>
		{:else}
			<p class="simple-intro" role="status">Загружаем комментарии…</p>
		{/if}
	{:else}
		<form onsubmit={submitComment}>
			<label for="comment-body">Ваш комментарий</label>
			<textarea
				id="comment-body"
				rows="3"
				maxlength={limit * 2}
				bind:value={draft}
				readonly={pending === 'create'}
				aria-describedby={tooLong ? 'comment-count comment-error' : 'comment-count'}
				aria-invalid={tooLong || undefined}></textarea>
			<div class="form-actions">
				<span id="comment-count" class={{ 'over-limit': tooLong }}>{length} / {limit}</span>
				<button
					class="button button-primary"
					type="submit"
					aria-disabled={pending !== null || !draft.trim() || tooLong}
				>
					{pending === 'create' ? 'Отправляем…' : 'Отправить'}
				</button>
			</div>
			{#if tooLong}
				<p id="comment-error" class="feedback error-feedback" role="alert">
					Сократите комментарий до 2000 символов.
				</p>
			{:else if issue?.action === 'create'}
				<p class="feedback error-feedback" role="alert">{issue.text}</p>
			{/if}
		</form>

		{#if comments.length}
			<ul class="comments">
				{#each comments as comment (comment.id)}
					<li id="comment-{comment.id}" tabindex="-1">
						<div class="comment-meta">
							<strong>@{comment.nickname}</strong>
							<time datetime={comment.createdAt}>{formatPostDate(comment.createdAt)}</time>
							{#if comment.own}
								<button
									class="remove"
									type="button"
									aria-disabled={pending !== null}
									onclick={(event) => void deleteComment(comment, event.currentTarget)}
								>
									Удалить<span class="sr-only"> свой комментарий</span>
								</button>
							{/if}
						</div>
						<p>{comment.body}</p>
					</li>
				{/each}
			</ul>
		{:else}
			<p class="simple-intro empty">Комментариев пока нет.</p>
		{/if}
		{#if issue?.action === 'older' || issue?.action === 'delete'}
			<p class="feedback error-feedback" role="alert">{issue.text}</p>
		{/if}
		{#if before}
			<button
				class="button button-outline older"
				type="button"
				aria-disabled={pending !== null}
				onclick={(event) => void loadComments(event.currentTarget)}
			>
				{pending === 'older' ? 'Загружаем…' : 'Загрузить ранее'}
				<Icon name="chevron" size={16} />
			</button>
		{/if}
	{/if}
</section>

<style>
	.actions {
		display: flex;
		flex-wrap: wrap;
		gap: 4px 18px;
		margin-top: 34px;
		padding-block: 6px;
		border-block: 1px solid var(--line);
	}
	.action {
		display: inline-flex;
		align-items: center;
		gap: 7px;
		min-height: 44px;
		padding: 0 8px;
		border-radius: 8px;
		color: var(--muted);
		font-size: 14px;
		font-weight: 700;
	}
	.action:not([aria-disabled='true']):hover {
		background: var(--blue-soft);
		color: var(--blue-hover);
	}
	.like[aria-pressed='true'] {
		color: var(--blue-hover);
	}
	.like[aria-pressed='true'] :global(svg) {
		fill: currentColor;
	}
	.count {
		color: var(--ink);
		font-variant-numeric: tabular-nums;
	}
	.discussion {
		margin-top: 28px;
	}
	h2 {
		margin-bottom: 14px;
		font-size: 24px;
		font-weight: 800;
		letter-spacing: -0.025em;
	}
	.guest a,
	.remove {
		color: var(--blue-hover);
		font-weight: 700;
	}
	.guest a {
		text-decoration: underline;
	}
	label {
		display: block;
		margin-bottom: 7px;
		font-size: 13px;
		font-weight: 700;
	}
	textarea {
		display: block;
		width: 100%;
		min-height: 100px;
		resize: vertical;
		border: 1px solid #aeb6ca;
		border-radius: 9px;
		padding: 12px 14px;
		background: var(--white);
		caret-color: var(--blue);
		font-size: 15px;
		line-height: 1.6;
	}
	textarea[aria-invalid='true'] {
		border-color: var(--error);
	}
	.form-actions {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		justify-content: space-between;
		gap: 10px;
		margin-top: 10px;
		color: var(--muted);
		font-size: 12px;
		font-variant-numeric: tabular-nums;
	}
	.form-actions .button {
		min-height: 44px;
		padding-block: 9px;
	}
	.over-limit {
		color: var(--error);
	}
	[aria-disabled='true'] {
		opacity: 0.65;
		cursor: default;
	}
	.button[aria-disabled='true']:hover {
		background: var(--blue);
	}
	.comments,
	.empty {
		margin-top: 22px;
		border-top: 1px solid var(--line);
	}
	.comments li,
	.empty {
		padding-block: 14px;
	}
	.comments li + li {
		border-top: 1px solid var(--line);
	}
	.comment-meta {
		display: flex;
		align-items: center;
		flex-wrap: wrap;
		gap: 0 12px;
		min-height: 28px;
	}
	.comment-meta strong {
		font-size: 14px;
		overflow-wrap: anywhere;
	}
	.comment-meta time,
	.remove {
		font-size: 13px;
	}
	.comment-meta time {
		color: var(--muted);
	}
	.remove {
		min-height: 44px;
		margin: -8px 0 -8px auto;
		padding-inline: 8px;
	}
	.remove:not([aria-disabled='true']):hover {
		text-decoration: underline;
	}
	.comments p {
		margin-top: 3px;
		white-space: pre-wrap;
		overflow-wrap: anywhere;
		font-size: 15px;
		line-height: 1.65;
	}
	.older {
		margin-top: 12px;
	}
	.older[aria-disabled='true']:hover {
		background: var(--white);
	}
	@media (max-width: 640px) {
		.actions {
			margin-top: 26px;
			gap: 4px 8px;
		}
	}
</style>
