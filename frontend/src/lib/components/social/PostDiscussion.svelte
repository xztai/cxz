<script lang="ts">
	import { onMount } from 'svelte';
	import Heart from '@lucide/svelte/icons/heart';
	import { api, checkError, errorMessage, ApiError } from '#lib/api/client.js';
	import type { components } from '#lib/api/schema.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Textarea } from '#lib/components/ui/textarea/index.js';
	import { Spinner } from '#lib/components/ui/spinner/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import * as Field from '#lib/components/ui/field/index.js';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import * as Empty from '#lib/components/ui/empty/index.js';
	let { postId }: { postId: string } = $props();
	let summary = $state<components['schemas']['InteractionSummary'] | null>(null);
	let comments = $state.raw<components['schemas']['Comment'][]>([]);
	let next = $state<string | null>(null);
	let loading = $state(true);
	let pending = $state('');
	let failure = $state('');
	let message = $state('');
	let body = $state('');
	function fail(cause: unknown) {
		failure = errorMessage(cause);
		if (
			cause instanceof ApiError &&
			['post_access_required', 'not_found', 'unauthorized'].includes(cause.code)
		) {
			summary = null;
			comments = [];
			next = null;
		}
	}
	async function load(more = false) {
		if (pending) return;
		pending = 'load';
		failure = '';
		try {
			const result = await api.GET('/posts/{id}/discussion', {
				params: { path: { id: postId }, query: more && next ? { before: next } : {} }
			});
			checkError(result.error);
			if (!result.data) throw new Error('Missing discussion');
			summary = result.data.summary;
			comments = more ? [...comments, ...result.data.comments] : result.data.comments;
			next = result.data.nextCursor;
		} catch (cause) {
			fail(cause);
		} finally {
			pending = '';
			loading = false;
		}
	}
	onMount(() => {
		void load();
	});
	async function like() {
		if (!summary || pending) return;
		pending = 'like';
		failure = '';
		try {
			const options = { params: { path: { id: postId } } };
			const result = summary.liked
				? await api.DELETE('/posts/{id}/like', options)
				: await api.PUT('/posts/{id}/like', options);
			checkError(result.error);
			if (result.data) summary = result.data.summary;
		} catch (cause) {
			fail(cause);
		} finally {
			pending = '';
		}
	}
	async function comment(event: SubmitEvent) {
		event.preventDefault();
		if (pending) return;
		pending = 'comment';
		failure = '';
		message = '';
		try {
			const result = await api.POST('/posts/{id}/comments', {
				params: { path: { id: postId } },
				body: { body }
			});
			checkError(result.error);
			if (!result.data) throw new Error('Missing comment');
			summary = result.data.summary;
			comments = [result.data.comment, ...comments];
			body = '';
			message = 'Комментарий добавлен.';
		} catch (cause) {
			fail(cause);
		} finally {
			pending = '';
		}
	}
	async function remove(id: string) {
		if (pending) return;
		pending = id;
		failure = '';
		message = '';
		try {
			const result = await api.DELETE('/posts/{id}/comments/{commentID}', {
				params: { path: { id: postId, commentID: id } }
			});
			checkError(result.error);
			comments = comments.filter((c) => c.id !== id);
			if (summary) summary = { ...summary, commentCount: Math.max(0, summary.commentCount - 1) };
			message = 'Комментарий удалён.';
		} catch (cause) {
			fail(cause);
		} finally {
			pending = '';
		}
	}
	function date(value: string) {
		return new Intl.DateTimeFormat('ru-RU', { dateStyle: 'medium', timeStyle: 'short' }).format(
			new Date(value)
		);
	}
</script>

<section
	class="mt-12 flex flex-col gap-6 border-t border-border pt-8"
	aria-labelledby="discussion-heading"
>
	<h2 id="discussion-heading">Обсуждение</h2>
	{#if loading}<div role="status" aria-label="Загрузка обсуждения" class="flex flex-col gap-3">
			<Skeleton class="h-10 w-32" /><Skeleton class="h-28 w-full" />
		</div>
	{:else if summary}
		<div class="flex flex-wrap items-center gap-4">
			<Button
				variant={summary.liked ? 'default' : 'outline'}
				aria-pressed={summary.liked}
				onclick={like}
				disabled={!!pending}
				aria-label={summary.liked ? 'Убрать лайк' : 'Поставить лайк'}
				>{#if pending === 'like'}<Spinner data-icon="inline-start" />{:else}<Heart
						data-icon="inline-start"
						fill={summary.liked ? 'currentColor' : 'none'}
					/>{/if}Нравится · {summary.likeCount}</Button
			>
			<p>{summary.commentCount} комментариев</p>
		</div>
		<form method="POST" onsubmit={comment}>
			<Field.FieldGroup
				><Field.Field
					><Field.FieldLabel for="comment-body">Ваш комментарий</Field.FieldLabel><Textarea
						id="comment-body"
						name="body"
						bind:value={body}
						maxlength={2000}
						required
						disabled={!!pending}
						aria-describedby="comment-hint"
					/><Field.FieldDescription id="comment-hint"
						>До 2000 символов. Обычный текст, без HTML и Markdown.</Field.FieldDescription
					></Field.Field
				><Field.Field
					><Button type="submit" disabled={!!pending || !body.trim()}
						>{#if pending === 'comment'}<Spinner data-icon="inline-start" />{/if}Отправить
						комментарий</Button
					></Field.Field
				></Field.FieldGroup
			>
		</form>
		{#if comments.length}<ol class="flex flex-col gap-6" aria-label="Комментарии">
				{#each comments as c (c.id)}<li class="flex flex-col gap-3 border-t border-border pt-5">
						<div class="flex flex-wrap items-center justify-between gap-3">
							<p>
								<strong>@{c.nickname}</strong> ·
								<time datetime={c.createdAt} class="text-muted-foreground">{date(c.createdAt)}</time
								>
							</p>
							{#if c.own}<Button
									variant="ghost"
									aria-label="Удалить свой комментарий"
									disabled={!!pending}
									onclick={() => remove(c.id)}
									>{#if pending === c.id}<Spinner data-icon="inline-start" />{/if}Удалить</Button
								>{/if}
						</div>
						<p class="whitespace-pre-wrap break-words">{c.body}</p>
					</li>{/each}
			</ol>
		{:else}<Empty.Root
				><Empty.Header
					><Empty.Title>Начните разговор</Empty.Title><Empty.Description
						>Поделитесь впечатлением от публикации. Все комментарии появляются в одном списке, без
						веток ответов.</Empty.Description
					></Empty.Header
				></Empty.Root
			>{/if}
		{#if next}<Button variant="outline" onclick={() => load(true)} disabled={!!pending}
				>{#if pending === 'load'}<Spinner data-icon="inline-start" />{/if}Ещё комментарии</Button
			>{/if}
	{/if}
	{#if failure}<Alert.Root variant="destructive" role="alert"
			><Alert.Title>Не удалось обновить обсуждение</Alert.Title><Alert.Description
				>{failure}</Alert.Description
			></Alert.Root
		>{#if !summary}<Button variant="outline" onclick={() => load()} disabled={!!pending}
				>Попробовать снова</Button
			>{/if}{/if}
	{#if message}<p role="status">{message}</p>{/if}
</section>
