<script lang="ts">
	import { invalidate } from '$app/navigation';
	import { resolve } from '$app/paths';
	import type { PageProps } from './$types';
	import PostEditor from '../../PostEditor.svelte';

	let { data }: PageProps = $props();
	let retrying = $state(false);

	async function retry() {
		retrying = true;
		try {
			await invalidate('studio:post-editor');
		} finally {
			retrying = false;
		}
	}
</script>

<svelte:head><title>Редактировать пост — etheyl</title></svelte:head>

<a class="studio-link back" href={resolve('/studio/posts')}>Все посты</a>
<h1 class="studio-heading">Редактировать пост</h1>
{#if data.access.state === 'ready' && !data.access.profile}
	<div class="studio-panel">
		<p class="studio-copy">Посты доступны после сохранения профиля автора.</p>
		<a class="button button-primary" href={resolve('/studio/profile')}>Создать профиль</a>
	</div>
{:else if data.access.state === 'ready' && data.post}
	{#key data.post.id}
		<PostEditor initialPost={data.post} tiers={data.tiers} ownerId={data.access.ownerId} />
	{/key}
{:else}
	<div class="studio-panel">
		<p class="feedback error-feedback" role="alert">
			{data.missing ? 'Пост не найден.' : 'Не удалось загрузить пост.'}
		</p>
		{#if !data.missing}
			<button type="button" class="button button-outline" disabled={retrying} onclick={retry}>
				{retrying ? 'Загружаем…' : 'Повторить'}
			</button>
		{/if}
	</div>
{/if}

<style>
	.back {
		display: inline-block;
		margin-block-end: 12px;
		font-size: 14px;
	}
</style>
