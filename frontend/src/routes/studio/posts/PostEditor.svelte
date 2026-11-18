<script lang="ts">
	import { goto, invalidate } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { untrack } from 'svelte';
	import { api, errorMessage, type Post, type PostInput, type Tier } from '#lib/api/client.js';
	import { studioEditorIsActive } from '#lib/session.svelte.js';

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
	let message = $state('');
	let confirmation = $state('');
	let retrying = $state(false);
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

	async function save(event: SubmitEvent) {
		event.preventDefault();
		if (operation || !isActive()) return;
		const publish = (event.submitter as HTMLButtonElement | null)?.value === 'publish';
		const input = { ...fields };
		operation = publish ? 'publish' : 'save';
		message = '';
		confirmation = '';
		try {
			const saved = post
				? await api.updatePost(ownerId, post.id, input)
				: await api.createPost(ownerId, input);
			if (!isActive()) return;
			post = saved.post;
			fields.title = post.title;

			if (publish) {
				const result = await api.publishPost(ownerId, post.id).catch(() => null);
				if (!isActive()) return;
				if (result) post = result.post;
				else message = 'Черновик сохранён. Не удалось опубликовать пост. Повторите попытку.';
			}
			await invalidate('studio:overview');
			if (message || !isActive()) return;
			confirmation = publish ? 'Пост опубликован.' : 'Изменения сохранены.';
			if (!initial) {
				await goto(resolve('/studio/posts/[id]/edit', { id: post.id }), { replace: true });
			}
		} catch (error) {
			if (isActive()) message = errorMessage(error);
		} finally {
			operation = undefined;
		}
	}

	async function unpublish() {
		if (operation || !post?.publishedAt || !isActive()) return;
		operation = 'unpublish';
		message = '';
		confirmation = '';
		try {
			const result = await api.unpublishPost(ownerId, post.id);
			if (!isActive()) return;
			post = result.post;
			confirmation = 'Пост снят с публикации.';
			await invalidate('studio:overview');
		} catch (error) {
			if (isActive()) message = errorMessage(error);
		} finally {
			operation = undefined;
		}
	}
</script>

{#if post}
	<p class="studio-copy">Статус — {post.publishedAt ? 'опубликован' : 'черновик'}</p>
{/if}
<form class="studio-form studio-panel" aria-busy={!!operation} onsubmit={save}>
	<fieldset disabled={!!operation}>
		<div class="field">
			<label for="post-title">Заголовок</label>
			<input id="post-title" bind:value={fields.title} required />
		</div>
		<div class="field">
			<label for="post-teaser">Краткое описание</label>
			<textarea id="post-teaser" bind:value={fields.teaser} rows="3" aria-describedby="teaser-hint"
			></textarea>
			<p id="teaser-hint" class="field-hint">Это описание увидят все читатели.</p>
		</div>
		<div class="field">
			<label for="post-body">Текст поста</label>
			<textarea
				id="post-body"
				bind:value={fields.bodyMarkdown}
				rows="9"
				aria-describedby="body-hint"></textarea>
			<p id="body-hint" class="field-hint">Поддерживается Markdown.</p>
		</div>
		<div class="field">
			<label for="post-tier">Кто может читать</label>
			<select id="post-tier" bind:value={fields.minimumTierId}>
				<option value={null}>Все читатели</option>
				{#if currentTierId && !availableTiers.some((tier) => tier.id === currentTierId)}
					<option value={currentTierId}>Текущий уровень подписки</option>
				{/if}
				{#each availableTiers as tier (tier.id)}
					<option value={tier.id}>{tier.name}{tier.archived ? ' (в архиве)' : ''}</option>
				{/each}
			</select>
			{#if !tiers}
				<p class="feedback error-feedback" role="status">Не удалось загрузить уровни подписки.</p>
				<button class="studio-link" type="button" disabled={retrying} onclick={reloadTiers}
					>{retrying ? 'Загружаем…' : 'Повторить'}</button
				>
			{/if}
		</div>
		<div class="actions">
			<button type="submit" class="button button-primary">
				{operation === 'save'
					? 'Сохраняем…'
					: post?.publishedAt
						? 'Сохранить изменения'
						: 'Сохранить черновик'}
			</button>
			{#if post?.publishedAt}
				<button type="button" class="button button-outline" onclick={unpublish}>
					{operation === 'unpublish' ? 'Снимаем…' : 'Снять с публикации'}
				</button>
			{:else}
				<button type="submit" class="button button-outline" value="publish">
					{operation === 'publish' ? 'Публикуем…' : 'Опубликовать'}
				</button>
			{/if}
		</div>
	</fieldset>
	{#if message}<p class="feedback error-feedback" role="alert">{message}</p>{/if}
	<p class="feedback" role="status">{confirmation}</p>
</form>

<style>
	.actions {
		display: flex;
		flex-wrap: wrap;
		gap: 12px;
	}

	.field button {
		min-height: 44px;
		cursor: pointer;
	}
</style>
