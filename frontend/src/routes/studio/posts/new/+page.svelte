<script lang="ts">
	import { resolve } from '$app/paths';
	import type { PageProps } from './$types';
	import PostEditor from '../PostEditor.svelte';

	let { data }: PageProps = $props();
</script>

<svelte:head><title>Новый пост — etheyl</title></svelte:head>

<a class="studio-link back" href={resolve('/studio/posts')}>Все посты</a>
<h1 class="studio-heading">Новый пост</h1>
{#if data.access.state === 'ready' && data.access.profile}
	<PostEditor tiers={data.tiers} />
{:else}
	<div class="studio-panel">
		<p class="studio-copy">Посты можно создавать после сохранения профиля автора.</p>
		<a class="button button-primary" href={resolve('/studio/profile')}>Создать профиль</a>
	</div>
{/if}

<style>
	.back {
		display: inline-block;
		margin-block-end: 12px;
		font-size: 14px;
	}
</style>
