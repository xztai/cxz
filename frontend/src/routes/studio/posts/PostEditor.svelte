<script lang="ts">
	import { goto, invalidate } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { HtmlRenderer, Parser } from 'commonmark';
	import { untrack } from 'svelte';
	import {
		api,
		errorMessage,
		type Post,
		type PostFile,
		type PostInput,
		type Tier
	} from '#lib/api/client.js';
	import Icon from '#lib/components/Icon.svelte';
	import { studioEditorIsActive } from '#lib/session.svelte.js';
	import PostAttachments from './PostAttachments.svelte';

	let {
		initialPost = null,
		tiers,
		ownerId: loadedOwnerId
	}: { initialPost?: Post | null; tiers: Tier[] | null; ownerId: string } = $props();
	const ownerId = untrack(() => loadedOwnerId);
	const isActive = studioEditorIsActive(ownerId);
	const initial = untrack(() => initialPost);
	let post = $state.raw(initial);
	let fields = $state<PostInput>({
		title: initial?.title ?? '',
		teaser: initial?.teaser ?? '',
		bodyMarkdown: initial?.bodyMarkdown ?? '',
		minimumTierId: initial?.minimumTierId ?? null
	});
	let operation = $state<'save' | 'publish' | 'unpublish'>();
	let feedback = $state<App.PageState['postEditorFeedback']>(
		untrack(() => page.state.postEditorFeedback)
	);
	let retrying = $state(false);
	let attachmentsBusy = $state(false);
	let preview = $state(false);
	const parser = new Parser();
	const renderer = new HtmlRenderer({ safe: true });
	const previewHtml = $derived(preview ? renderer.render(parser.parse(fields.bodyMarkdown)) : '');
	const availableTiers = $derived(
		tiers?.filter((tier) => !tier.archived || tier.id === post?.minimumTierId) ?? []
	);
	const currentTierId = $derived(post?.minimumTierId);

	async function reloadTiers() {
		retrying = true;
		try {
			await invalidate('studio:post-editor');
		} finally {
			retrying = false;
		}
	}

	function updateFiles(update: (previous: PostFile[]) => PostFile[]) {
		if (post) post = { ...post, media: update(post.media ?? []) };
	}

	async function save(event: SubmitEvent) {
		event.preventDefault();
		if (operation || attachmentsBusy || !isActive()) return;
		const publish = (event.submitter as HTMLButtonElement | null)?.value === 'publish';
		const input = { ...fields };
		operation = publish ? 'publish' : 'save';
		feedback = undefined;

		try {
			const saved = post
				? await api.updatePost(ownerId, post.id, input)
				: await api.createPost(ownerId, input);
			if (!isActive()) return;
			post = saved.post;
			fields.title = post.title;
			feedback = { text: 'Изменения сохранены.' };

			if (publish) {
				try {
					const result = await api.publishPost(ownerId, post.id);
					if (!isActive()) return;
					post = result.post;
					feedback = { text: 'Пост опубликован.' };
				} catch {
					if (!isActive()) return;
					feedback = {
						text: 'Черновик сохранён. Не удалось опубликовать пост. Повторите попытку.',
						error: true
					};
				}
			}

			await invalidate('studio:overview');
			if (!initial && isActive()) {
				await goto(resolve('/studio/posts/[id]/edit', { id: post.id }), {
					replace: true,
					state: { postEditorFeedback: feedback }
				});
			}
		} catch (error) {
			if (isActive()) feedback = { text: errorMessage(error), error: true };
		} finally {
			operation = undefined;
		}
	}

	async function unpublish() {
		if (operation || attachmentsBusy || !post?.publishedAt || !isActive()) return;
		operation = 'unpublish';
		feedback = undefined;
		try {
			const result = await api.unpublishPost(ownerId, post.id);
			if (!isActive()) return;
			post = result.post;
			feedback = { text: 'Пост снят с публикации.' };
			await invalidate('studio:overview');
		} catch (error) {
			if (isActive()) feedback = { text: errorMessage(error), error: true };
		} finally {
			operation = undefined;
		}
	}
</script>

<header class="editor-header">
	<div class="editor-toolbar">
		<div class="editor-context">
			<a class="back-link studio-link" href={resolve('/studio/posts')}>
				<span class="back-icon"><Icon name="arrow" size={17} /></span>
				Назад
			</a>
			<span class={['post-status', { published: post?.publishedAt }]}>
				{post?.publishedAt ? 'Опубликован' : 'Черновик'}
			</span>
		</div>
		<div class="editor-actions">
			<button
				type="submit"
				form="post-editor-form"
				class={['button', post?.publishedAt ? 'button-primary' : 'button-outline']}
				disabled={!!operation || attachmentsBusy}
			>
				{operation === 'save' ? 'Сохраняем…' : 'Сохранить'}
			</button>
			{#if post?.publishedAt}
				<button
					type="button"
					class="button button-outline"
					disabled={!!operation || attachmentsBusy}
					onclick={unpublish}
				>
					{operation === 'unpublish' ? 'Снимаем…' : 'Снять'}
				</button>
			{:else}
				<button
					type="submit"
					form="post-editor-form"
					class="button button-primary"
					disabled={!!operation || attachmentsBusy}
					value="publish"
				>
					{operation === 'publish' ? 'Публикуем…' : 'Опубликовать'}
				</button>
			{/if}
		</div>
	</div>
	<h1 class="studio-heading">{initial ? 'Редактировать пост' : 'Новый пост'}</h1>
	{#if feedback?.error}
		<p class="feedback error-feedback" role="alert">{feedback.text}</p>
	{:else}
		<p class="feedback" role="status">{feedback?.text ?? ''}</p>
	{/if}
</header>

<div class="editor-layout">
	<form
		id="post-editor-form"
		class="editor-form studio-panel"
		aria-busy={!!operation}
		onsubmit={save}
	>
		<fieldset disabled={!!operation}>
			<div class="field title-field">
				<label for="post-title">Заголовок</label>
				<input id="post-title" bind:value={fields.title} placeholder="Название поста" required />
			</div>
			<div class="post-details">
				<div class="field">
					<label for="post-teaser">Тизер</label>
					<textarea
						id="post-teaser"
						bind:value={fields.teaser}
						rows="2"
						aria-describedby="teaser-hint"></textarea>
					<p id="teaser-hint" class="field-hint">Виден всем</p>
				</div>
				<div class="field">
					<label for="post-tier">Доступ</label>
					<select id="post-tier" bind:value={fields.minimumTierId}>
						<option value={null}>Публичный</option>
						{#if currentTierId && !availableTiers.some((tier) => tier.id === currentTierId)}
							<option value={currentTierId}>Текущий уровень подписки</option>
						{/if}
						{#each availableTiers as tier (tier.id)}
							<option value={tier.id}>{tier.name}{tier.archived ? ' (в архиве)' : ''}</option>
						{/each}
					</select>
					{#if !tiers}
						<p class="field-error" role="status">Не удалось загрузить уровни подписки.</p>
						<button
							class="studio-link retry"
							type="button"
							disabled={retrying}
							onclick={reloadTiers}>{retrying ? 'Загружаем…' : 'Повторить'}</button
						>
					{/if}
				</div>
			</div>
			<div class="field body-field">
				<div class="body-toolbar">
					<label for="post-body">Текст поста</label>
					<div class="editor-modes" role="group" aria-label="Режим редактора">
						<button
							type="button"
							aria-pressed={!preview}
							aria-controls="post-body"
							onclick={() => (preview = false)}>Редактор</button
						>
						<button
							type="button"
							aria-pressed={preview}
							aria-controls="post-preview"
							onclick={() => (preview = true)}>Предпросмотр</button
						>
					</div>
				</div>
				<textarea
					id="post-body"
					hidden={preview}
					bind:value={fields.bodyMarkdown}
					rows="11"
					placeholder="Напишите пост…"
					aria-describedby="body-hint"></textarea>
				<div id="post-preview" class="markdown-preview" hidden={!preview}>
					{#if previewHtml}
						<!-- CommonMark's safe renderer omits raw HTML and unsafe URLs. -->
						<!-- eslint-disable-next-line svelte/no-at-html-tags -->
						{@html previewHtml}
					{:else}
						<p class="preview-empty">Здесь появится текст поста.</p>
					{/if}
				</div>
				<p id="body-hint" class="field-hint">Поддерживается Markdown.</p>
			</div>
		</fieldset>
	</form>

	<PostAttachments
		postId={post?.id}
		files={post?.media ?? []}
		{ownerId}
		disabled={!!operation}
		bind:busy={attachmentsBusy}
		onFilesChange={updateFiles}
	/>
</div>

<style>
	.editor-header {
		margin-block-end: 24px;
	}
	.editor-toolbar,
	.editor-context,
	.editor-actions,
	.body-toolbar {
		display: flex;
		align-items: center;
	}
	.editor-toolbar {
		justify-content: space-between;
		flex-wrap: wrap;
		gap: 12px 24px;
		margin-block-end: 22px;
	}
	.editor-context,
	.editor-actions {
		gap: 10px;
	}
	.editor-context {
		gap: 16px;
	}
	.back-link {
		display: inline-flex;
		align-items: center;
		gap: 7px;
		min-height: 44px;
		font-size: 14px;
	}
	.back-icon {
		transform: rotate(180deg);
	}
	.post-status {
		display: inline-flex;
		align-items: center;
		gap: 7px;
		padding: 5px 9px;
		border: 1px solid var(--line);
		border-radius: 8px;
		background: var(--white);
		color: var(--muted);
		font-size: 12px;
		font-weight: 750;
	}
	.post-status::before {
		width: 6px;
		height: 6px;
		border-radius: 50%;
		background: currentColor;
		content: '';
	}
	.post-status.published {
		border-color: var(--blue-soft);
		background: var(--blue-soft);
		color: var(--blue-hover);
	}
	.editor-actions .button {
		min-height: 44px;
		padding: 9px 16px;
		font-size: 14px;
	}
	.editor-header .feedback {
		margin-block-end: 0;
	}
	.editor-layout {
		display: grid;
		gap: 24px;
		align-items: start;
	}
	.editor-form {
		container-type: inline-size;
		min-width: 0;
	}
	.editor-form fieldset,
	.post-details > * {
		min-width: 0;
	}
	.editor-form .field {
		margin: 0;
	}
	.post-details {
		display: grid;
		align-items: start;
		gap: 20px;
		margin-block: 20px 24px;
	}
	.editor-form textarea,
	.editor-form select {
		display: block;
		width: 100%;
		min-height: 50px;
		padding: 12px 14px;
		border: 1px solid #aeb6ca;
		border-radius: 9px;
		background: var(--white);
		color: var(--ink);
		font-size: 15px;
		line-height: 24px;
	}
	.editor-form textarea {
		resize: vertical;
		caret-color: var(--blue);
	}
	.editor-form textarea::placeholder {
		color: #747b8e;
		opacity: 1;
	}
	.editor-form select {
		text-overflow: ellipsis;
	}
	.editor-form textarea[hidden] {
		display: none;
	}
	.editor-form textarea:focus,
	.editor-form select:focus {
		border-color: var(--blue);
		outline: 2px solid var(--blue);
		outline-offset: 1px;
	}
	.retry {
		min-height: 44px;
		font-size: 13px;
		cursor: pointer;
	}
	.body-toolbar {
		justify-content: space-between;
		flex-wrap: wrap;
		gap: 6px 12px;
		margin-block-end: 8px;
	}
	.body-toolbar label {
		margin: 0;
	}
	.editor-modes {
		display: inline-flex;
		max-width: 100%;
		padding: 3px;
		border-radius: 9px;
		background: var(--surface);
	}
	.editor-modes button {
		min-height: 36px;
		padding: 5px 10px;
		border: 1px solid transparent;
		border-radius: 7px;
		color: var(--muted);
		font-size: 13px;
		font-weight: 700;
		cursor: pointer;
	}
	.editor-modes button[aria-pressed='true'] {
		border-color: var(--line);
		background: var(--white);
		color: var(--blue-hover);
	}
	.editor-modes button:hover:not(:disabled) {
		color: var(--blue-hover);
	}
	.markdown-preview {
		min-height: calc(11 * 24px + 26px);
		padding: 12px 14px;
		overflow-wrap: anywhere;
		border: 1px solid #aeb6ca;
		border-radius: 9px;
		background: var(--white);
		font-size: 15px;
		line-height: 1.65;
	}
	.markdown-preview :global(> * + *) {
		margin-block-start: 0.8em;
	}
	.markdown-preview :global(h1),
	.markdown-preview :global(h2),
	.markdown-preview :global(h3) {
		font-weight: 800;
		line-height: 1.25;
	}
	.markdown-preview :global(h1) {
		font-size: 1.6em;
	}
	.markdown-preview :global(h2) {
		font-size: 1.35em;
	}
	.markdown-preview :global(h3) {
		font-size: 1.15em;
	}
	.markdown-preview :global(ul),
	.markdown-preview :global(ol) {
		padding-inline-start: 1.5em;
		list-style: revert;
	}
	.markdown-preview :global(blockquote) {
		padding-inline-start: 1em;
		border-inline-start: 1px solid var(--line);
		color: var(--muted);
	}
	.markdown-preview :global(pre) {
		overflow: auto;
		padding: 10px 12px;
		border-radius: 8px;
		background: var(--surface);
	}
	.markdown-preview :global(a[href]) {
		color: var(--blue-hover);
		text-decoration: underline;
	}
	.preview-empty {
		color: var(--muted);
	}
	@container (min-width: 500px) {
		.post-details {
			grid-template-columns: minmax(0, 1fr) minmax(180px, 0.6fr);
		}
	}
	@media (min-width: 1160px) {
		.editor-layout {
			grid-template-columns: minmax(0, 1fr) 302px;
		}
	}
	@media (max-width: 520px) {
		.editor-form.studio-panel {
			padding: 16px;
		}
		.editor-toolbar {
			margin-block-end: 18px;
		}
		.editor-actions {
			display: grid;
			grid-template-columns: 1fr 1fr;
			width: 100%;
		}
	}
</style>
