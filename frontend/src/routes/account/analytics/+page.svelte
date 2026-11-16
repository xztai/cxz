<script lang="ts">
	import AnalyticsReport from '#lib/components/analytics/AnalyticsReport.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Empty from '#lib/components/ui/empty/index.js';
	import type { PageProps } from './$types';
	let { data }: PageProps = $props();
</script>

<svelte:head
	><title>Аналитика автора — etheyl</title><meta name="robots" content="noindex" /></svelte:head
>
<section class="workspace-page page-inset">
	<a class="back-link" href="/account">Личный кабинет</a>
	<div class="workspace-heading">
		<div>
			<h1>Аналитика автора</h1>
			<p>Тестовые платежи и оплаченный доступ.</p>
		</div>
		<Button href="/account/posts" variant="outline">Мои публикации</Button>
	</div>
	{#if data.analytics}<AnalyticsReport report={data.analytics} />{:else}<Empty.Root
			><Empty.Header
				><Empty.Title>Сначала оформите профиль автора</Empty.Title><Empty.Description
					>Аналитика появится после создания публичного профиля. Первые платежи и подписки будут
					учтены автоматически.</Empty.Description
				></Empty.Header
			><Empty.Content><Button href="/account/profile">Оформить профиль</Button></Empty.Content
			></Empty.Root
		>{/if}
</section>
