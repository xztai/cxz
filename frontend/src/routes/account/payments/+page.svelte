<script lang="ts">
	import { Button } from '#lib/components/ui/button/index.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import * as Empty from '#lib/components/ui/empty/index.js';
	import { rubles } from '#lib/creators/prices.js';
	import { billingDate, paymentStatus } from '#lib/billing/format.js';
	import type { PageProps } from './$types';
	let { data }: PageProps = $props();
</script>

<svelte:head
	><title>История платежей — etheyl</title><meta name="robots" content="noindex" /></svelte:head
>
<section class="workspace-page page-inset">
	<a class="back-link" href="/account/subscriptions">Мои подписки</a>
	<div class="workspace-heading">
		<div>
			<h1>История платежей</h1>
			<p>Все суммы относятся к тестовому магазину ЮKassa.</p>
		</div>
	</div>
	{#if data.payments.length}{#each data.payments as payment (payment.id)}<article
				class="flex flex-wrap items-center justify-between gap-5 border-t border-border py-6"
			>
				<div>
					<h2>{payment.displayName}</h2>
					<p>
						{payment.tierName} · {payment.renewal ? 'Продление' : 'Оформление'} · {billingDate(
							payment.createdAt
						)}
					</p>
					<p class="text-muted-foreground">{rubles(payment.amountKopecks)}</p>
				</div>
				<div class="flex flex-wrap items-center gap-3">
					<Badge variant="outline">{paymentStatus[payment.status]}</Badge><Button
						href={`/account/payments/${payment.id}`}
						variant="outline">Подробнее</Button
					>
				</div>
			</article>{/each}
	{:else}<Empty.Root
			><Empty.Header
				><Empty.Title>Платежей пока нет</Empty.Title><Empty.Description
					>После оформления подписки здесь появится результат оплаты.</Empty.Description
				></Empty.Header
			><Empty.Content><Button href="/authors" variant="outline">К авторам</Button></Empty.Content
			></Empty.Root
		>{/if}
	<nav class="post-pagination" aria-label="Страницы платежей">
		{#if data.offset > 0}<Button href={`?offset=${Math.max(0, data.offset - 24)}`} variant="outline"
				>Назад</Button
			>{/if}{#if data.hasMore}<Button href={`?offset=${data.offset + 24}`} variant="outline"
				>Дальше</Button
			>{/if}
	</nav>
</section>
