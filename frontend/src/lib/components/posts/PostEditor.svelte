<script lang="ts">
	import { goto, beforeNavigate } from '$app/navigation';
	import { untrack } from 'svelte';
	import { fileSize } from '#lib/posts/files.js';
	import { api, ApiError, checkError, errorMessage } from '#lib/api/client.js';
	import type { components } from '#lib/api/schema.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Textarea } from '#lib/components/ui/textarea/index.js';
	import { Spinner } from '#lib/components/ui/spinner/index.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import * as Field from '#lib/components/ui/field/index.js';
	import * as Alert from '#lib/components/ui/alert/index.js';

	let {
		initial,
		tiers
	}: { initial: components['schemas']['Post']; tiers: components['schemas']['Tier'][] } = $props();
	// The route keys this editor by post ID so each editor owns its draft state.
	const seed = untrack(() => initial);
	let post = $state.raw(seed);
	let title = $state(seed.title);
	let teaser = $state(seed.teaser);
	let bodyMarkdown = $state(seed.bodyMarkdown ?? '');
	let minimumTierId = $state(seed.minimumTierId ?? '');
	let kind = $state<components['schemas']['PostFile']['kind']>('image');
	let pending = $state('');
	let failure = $state('');
	let message = $state('');
	let feedbackScope = $state('post');
	let deleting = $state(false);
	let leaving = false;
	let dirty = $derived(
		title !== post.title ||
			teaser !== post.teaser ||
			bodyMarkdown !== (post.bodyMarkdown ?? '') ||
			minimumTierId !== (post.minimumTierId ?? '')
	);
	beforeNavigate(({ cancel }) => {
		if (dirty && !leaving && !window.confirm('Выйти без сохранения изменений?')) cancel();
	});
	function beforeUnload(event: BeforeUnloadEvent) {
		if (dirty && !leaving) {
			event.preventDefault();
			event.returnValue = '';
		}
	}

	async function save(publish = false) {
		if (pending) return;
		feedbackScope = 'post';
		pending = publish ? 'publish' : 'save';
		failure = '';
		message = '';
		try {
			const result = await api.PUT('/me/posts/{id}', {
				params: { path: { id: post.id } },
				body: { title, teaser, bodyMarkdown, minimumTierId: minimumTierId || null }
			});
			checkError(result.error);
			if (result.data) {
				post = result.data.post;
				title = post.title;
			}
			if (publish) {
				const published = await api.POST('/me/posts/{id}/publish', {
					params: { path: { id: post.id } }
				});
				checkError(published.error);
				if (published.data) post = published.data.post;
			}
			message = publish ? 'Пост опубликован.' : 'Изменения сохранены.';
		} catch (cause) {
			failure = errorMessage(cause);
		} finally {
			pending = '';
		}
	}

	async function unpublish() {
		if (pending) return;
		feedbackScope = 'manage';
		pending = 'unpublish';
		failure = '';
		message = '';
		try {
			const result = await api.POST('/me/posts/{id}/unpublish', {
				params: { path: { id: post.id } }
			});
			checkError(result.error);
			if (result.data) post = result.data.post;
			message = 'Пост возвращён в черновики.';
		} catch (cause) {
			failure = errorMessage(cause);
		} finally {
			pending = '';
		}
	}

	async function upload(event: Event) {
		const input = event.currentTarget as HTMLInputElement;
		const files = Array.from(input.files ?? []);
		if (pending || !files.length) return;
		feedbackScope = 'files';
		failure = '';
		message = '';
		if (files.some((file) => file.size === 0 || file.size > 100 * 1024 * 1024)) {
			failure = 'Выберите непустые файлы до 100 МБ каждый.';
			input.value = '';
			return;
		}
		pending = 'upload';
		try {
			const csrf = await api.GET('/auth/csrf');
			checkError(csrf.error);
			if (!csrf.data) throw new Error('Missing CSRF token');
			for (const file of files) {
				message = `Загружается ${file.name}…`;
				const query = new URLSearchParams({ kind, filename: file.name });
				const response = await fetch(`/api/v1/me/posts/${post.id}/media?${query}`, {
					method: 'POST',
					credentials: 'same-origin',
					headers: { 'Content-Type': 'application/octet-stream', 'X-CSRF-Token': csrf.data.token },
					body: file,
					signal: AbortSignal.timeout(600_000)
				});
				const result = (await response.json()) as {
					media: components['schemas']['PostFile'];
				} & Partial<components['schemas']['Error']>;
				if (!response.ok)
					throw new ApiError(
						result.error?.code ?? 'upload',
						result.error?.message ?? 'Не удалось загрузить файл.'
					);
				post = { ...post, media: [...(post.media ?? []), result.media] };
			}
			message = 'Файлы добавлены к посту.';
		} catch (cause) {
			failure = errorMessage(cause);
			message = '';
		} finally {
			pending = '';
			input.value = '';
		}
	}

	async function removeFile(mediaID: string) {
		if (pending) return;
		feedbackScope = mediaID;
		pending = mediaID;
		failure = '';
		message = '';
		try {
			const result = await api.DELETE('/me/posts/{id}/media/{mediaID}', {
				params: { path: { id: post.id, mediaID } }
			});
			checkError(result.error);
			post = { ...post, media: post.media?.filter((file) => file.id !== mediaID) };
			feedbackScope = 'files';
			message = 'Файл удалён из поста.';
		} catch (cause) {
			failure = errorMessage(cause);
		} finally {
			pending = '';
		}
	}

	async function removePost() {
		if (pending) return;
		feedbackScope = 'manage';
		pending = 'delete';
		failure = '';
		message = '';
		try {
			checkError((await api.DELETE('/me/posts/{id}', { params: { path: { id: post.id } } })).error);
			leaving = true;
			await goto('/account/posts', { invalidateAll: true });
		} catch (cause) {
			failure = errorMessage(cause);
		} finally {
			pending = '';
		}
	}
</script>

{#snippet feedback(scope: string)}
	{#if feedbackScope === scope}
		{#if failure}<Alert.Root variant="destructive" role="alert"
				><Alert.Title>Не удалось выполнить действие</Alert.Title><Alert.Description
					>{failure}</Alert.Description
				></Alert.Root
			>{/if}
		{#if message}<p role="status" class="save-message">{message}</p>{/if}
	{/if}
{/snippet}

<svelte:window onbeforeunload={beforeUnload} />
<section class="creator-workspace page-inset">
	<a href="/account/posts" class="back-link">← Все публикации</a>
	<div class="workspace-heading">
		<div>
			<h1>Редактор поста</h1>
			<p>Сохраните историю и выберите её читателей.</p>
		</div>
		<Badge variant="outline">{post.publishedAt ? 'Опубликован' : 'Черновик'}</Badge>
	</div>
	<div class="post-editor-layout">
		<form
			method="POST"
			class="creator-form"
			onsubmit={(event) => {
				event.preventDefault();
				void save();
			}}
		>
			<Field.FieldGroup>
				<Field.Field
					><Field.FieldLabel for="post-title">Заголовок</Field.FieldLabel><Input
						id="post-title"
						required
						maxlength={200}
						bind:value={title}
						disabled={!!pending}
					/></Field.Field
				>
				<Field.Field
					><Field.FieldLabel for="post-teaser">Публичный анонс</Field.FieldLabel><Textarea
						id="post-teaser"
						maxlength={2000}
						bind:value={teaser}
						disabled={!!pending}
					/><Field.FieldDescription
						>Виден всем, в том числе у закрытого поста. Не добавляйте сюда закрытую часть истории.</Field.FieldDescription
					></Field.Field
				>
				<Field.Field
					><Field.FieldLabel for="post-body">Текст поста · Markdown</Field.FieldLabel><Textarea
						id="post-body"
						class="post-body-input"
						maxlength={100000}
						bind:value={bodyMarkdown}
						disabled={!!pending}
					/><Field.FieldDescription
						>Поддерживаются заголовки, списки, ссылки, цитаты и код. Произвольный HTML не
						отображается.</Field.FieldDescription
					></Field.Field
				>
				<Field.Field
					><Field.FieldLabel for="post-access">Кому доступен пост</Field.FieldLabel><select
						id="post-access"
						class="post-select"
						bind:value={minimumTierId}
						disabled={!!pending}
						><option value="">Всем</option
						>{#each tiers.filter((tier) => !tier.archived || tier.id === post.minimumTierId) as tier (tier.id)}<option
								value={tier.id}>От уровня «{tier.name}»{tier.archived ? ' (в архиве)' : ''}</option
							>{/each}</select
					><Field.FieldDescription
						>Указанный и все старшие уровни получают доступ к тексту и каждому файлу.</Field.FieldDescription
					></Field.Field
				>
			</Field.FieldGroup>
			<div class="post-actions">
				<Button type="submit" disabled={!!pending}
					>{#if pending === 'save'}<Spinner data-icon="inline-start" />{/if}Сохранить</Button
				>
				{#if !post.publishedAt}<Button
						type="button"
						variant="outline"
						onclick={() => save(true)}
						disabled={!!pending}
						>{#if pending === 'publish'}<Spinner
								data-icon="inline-start"
							/>{/if}Опубликовать</Button
					>{/if}
				<Button href={`/posts/${post.id}`} variant="link">Открыть сохранённую версию</Button>
			</div>
			{#if dirty}<p class="form-note" role="status">Есть несохранённые изменения.</p>{/if}
			{@render feedback('post')}
		</form>
		<aside class="post-files" aria-label="Файлы публикации">
			<h2>Медиа и вложения</h2>
			<p class="form-note">
				До 100 МБ на файл. Аудио и видео воспроизводятся в исходном формате, если его поддерживает
				браузер.
			</p>
			<Field.FieldGroup>
				<Field.Field
					><Field.FieldLabel for="file-kind">Тип файла</Field.FieldLabel><select
						id="file-kind"
						class="post-select"
						bind:value={kind}
						disabled={!!pending}
						><option value="image">Изображение</option><option value="audio">Аудио</option><option
							value="video">Видео</option
						><option value="attachment">Вложение для скачивания</option></select
					></Field.Field
				>
				<Field.Field
					><Field.FieldLabel for="post-files">Добавить файлы</Field.FieldLabel><Input
						id="post-files"
						type="file"
						multiple
						disabled={!!pending}
						onchange={upload}
					/><Field.FieldDescription
						>Файлы сразу добавляются к текущему посту. Для опубликованного поста они сразу доступны
						его читателям.</Field.FieldDescription
					></Field.Field
				>
			</Field.FieldGroup>
			{#if pending === 'upload'}<p class="flex items-center gap-2 mt-4">
					<Spinner />Загрузка…
				</p>{/if}
			{@render feedback('files')}
			<ul class="post-file-list">
				{#each post.media ?? [] as file (file.id)}<li>
						{#if file.thumbnailUrl}<img
								src={file.thumbnailUrl}
								alt={`Миниатюра ${file.filename}`}
							/>{/if}
						<a href={file.url}>{file.filename}</a><span class="text-sm text-muted-foreground"
							>{fileSize(file.sizeBytes)}</span
						>
						<Button
							type="button"
							variant="outline"
							disabled={!!pending}
							onclick={() => removeFile(file.id)}
							aria-label={`Удалить файл ${file.filename}`}>Удалить файл</Button
						>
						{@render feedback(file.id)}
					</li>{/each}
			</ul>
		</aside>
	</div>
	<div class="post-actions post-manage">
		{#if post.publishedAt}<Button variant="outline" disabled={!!pending} onclick={unpublish}
				>Снять с публикации</Button
			>{/if}
		<Button variant="outline" disabled={!!pending} onclick={() => (deleting = !deleting)}
			>Удалить пост</Button
		>
	</div>
	{@render feedback('manage')}
	{#if deleting}<Alert.Root
			><Alert.Title>Удалить пост и все его файлы?</Alert.Title><Alert.Description
				>Пост сразу перестанет открываться у читателей. Отменить удаление нельзя.
				<div class="post-actions">
					<Button variant="destructive" disabled={!!pending} onclick={removePost}
						>Удалить навсегда</Button
					><Button variant="outline" disabled={!!pending} onclick={() => (deleting = false)}
						>Отмена</Button
					>
				</div></Alert.Description
			></Alert.Root
		>{/if}
</section>
