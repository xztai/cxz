<script lang="ts">
	import SiteHeader from '#lib/components/SiteHeader.svelte';
	import { getSession } from '#lib/session.svelte.js';
	import PostReader from './PostReader.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const session = getSession();
	let viewer = $derived(session.user?.id);
</script>

<SiteHeader />
{#if session.ready}
	{#key `${data.id}:${viewer ?? ''}`}
		<PostReader id={data.id} signedIn={viewer !== undefined} />
	{/key}
{:else}
	<main id="main" class="shell simple-page">
		<p class="simple-intro" role="status">Загружаем пост…</p>
	</main>
{/if}
