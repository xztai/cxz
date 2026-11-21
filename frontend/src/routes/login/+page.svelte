<script lang="ts">
	import { page } from '$app/state';
	import SiteHeader from '#lib/components/SiteHeader.svelte';
	import AuthPanel from '#lib/components/AuthPanel.svelte';
	let author = $derived(page.url.searchParams.get('author'));
	let returnAuthor = $derived(author && /^[a-z0-9_-]{3,30}$/.test(author) ? author : undefined);
	let post = $derived(page.url.searchParams.get('post'));
	let returnPost = $derived(post && /^[1-9]\d{0,18}$/.test(post) ? post : undefined);
</script>

<SiteHeader />
<main id="main" class="shell simple-page">
	<AuthPanel mode="login" {returnAuthor} {returnPost} />
</main>
