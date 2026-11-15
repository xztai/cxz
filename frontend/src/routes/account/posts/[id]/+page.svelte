<script lang="ts">
	import PostEditor from '#lib/components/posts/PostEditor.svelte';
	import type { PageProps } from './$types';
	let { data }: PageProps = $props();
</script>

<svelte:head
	><title>Редактор поста — etheyl</title><meta name="robots" content="noindex" /></svelte:head
>
{#key data.post.id}<PostEditor initial={data.post} tiers={data.tiers} />{/key}
