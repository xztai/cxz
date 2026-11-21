<script lang="ts">
	import { page } from '$app/state';
	import SiteHeader from '#lib/components/SiteHeader.svelte';
	import AuthPanel from '#lib/components/AuthPanel.svelte';
	let author = $derived(page.url.searchParams.get('author'));
	let returnAuthor = $derived(author && /^[a-z0-9_-]{3,30}$/.test(author) ? author : undefined);
</script>

<SiteHeader />
<main id="main" class="shell simple-page"><AuthPanel mode="login" {returnAuthor} /></main>
