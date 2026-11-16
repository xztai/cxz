<script lang="ts">
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Empty from '#lib/components/ui/empty/index.js';
	import SubscriptionRow from '#lib/components/billing/SubscriptionRow.svelte';
	import type { PageProps } from './$types';
	let { data }: PageProps = $props();
</script>

<svelte:head
	><title>Мои подписки — etheyl</title><meta name="robots" content="noindex" /></svelte:head
>
<section class="workspace-page page-inset">
	<a href="/account" class="back-link">Личный кабинет</a>
	<div class="workspace-heading">
		<div>
			<h1>Мои подписки</h1>
			<p>Тестовый магазин. Оплаченный доступ и будущие продления.</p>
		</div>
		<Button href="/account/payments" variant="outline">История платежей</Button>
	</div>
	{#if data.subscriptions.length}{#each data.subscriptions as sub (sub.id)}<SubscriptionRow
				subscription={sub}
			/>{/each}
	{:else}<Empty.Root
			><Empty.Header
				><Empty.Title>Выберите своего автора</Empty.Title><Empty.Description
					>Здесь появятся ваши подписки. Найдите автора и выберите подходящий уровень.</Empty.Description
				></Empty.Header
			><Empty.Content><Button href="/authors">Найти автора</Button></Empty.Content></Empty.Root
		>{/if}
	<nav class="post-pagination" aria-label="Страницы подписок">
		{#if data.offset > 0}<Button href={`?offset=${Math.max(0, data.offset - 24)}`} variant="outline"
				>Назад</Button
			>{/if}{#if data.hasMore}<Button href={`?offset=${data.offset + 24}`} variant="outline"
				>Дальше</Button
			>{/if}
	</nav>
</section>
