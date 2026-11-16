<script lang="ts">
	import { onMount, untrack } from 'svelte';
	import type { components } from '#lib/api/schema.js';
	import { api, checkError, errorMessage } from '#lib/api/client.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Spinner } from '#lib/components/ui/spinner/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { rubles } from '#lib/creators/prices.js';
	import { billingDate, paymentStatus } from '#lib/billing/format.js';
	let { initial }: { initial: components['schemas']['Payment'] } = $props();
	// The route keys this component by payment ID, so polling owns one attempt.
	let payment = $state.raw(untrack(() => initial));
	let failure = $state('');
	let pending = $state(false);
	let waiting = $derived(payment.status === 'queued' || payment.status === 'pending');
	async function refresh() {
		if (pending) return;
		pending = true;
		try {
			const result = await api.GET('/me/payments/{id}', { params: { path: { id: payment.id } } });
			checkError(result.error);
			if (result.data && result.data.payment.id === payment.id) payment = result.data.payment;
			failure = '';
		} catch (cause) {
			failure = errorMessage(cause);
		} finally {
			pending = false;
		}
	}
	onMount(() => {
		const id = payment.id;
		let polls = 0;
		const timer = setInterval(() => {
			if (payment.id !== id || !waiting || polls++ >= 60) {
				clearInterval(timer);
				return;
			}
			void refresh();
		}, 3000);
		return () => clearInterval(timer);
	});
</script>

<svelte:head
	><title>{paymentStatus[payment.status]} — etheyl</title><meta
		name="robots"
		content="noindex"
	/></svelte:head
>
<section class="workspace-page page-inset">
	<a href="/account/payments" class="back-link">История платежей</a>
	<div class="workspace-heading">
		<div>
			<h1>Результат оплаты</h1>
			<p>Тестовый платёж · {rubles(payment.amountKopecks)}</p>
		</div>
	</div>
	<Card.Root class="max-w-xl"
		><Card.Header
			><Card.Title><h2 aria-live="polite">{paymentStatus[payment.status]}</h2></Card.Title
			><Card.Description>{billingDate(payment.createdAt)}</Card.Description></Card.Header
		>
		<Card.Content class="flex flex-col gap-5">
			{#if payment.status === 'succeeded'}<p>
					Оплата подтверждена. Доступ открыт до {billingDate(payment.periodEnd)}.
				</p>
			{:else if payment.status === 'canceled'}<p>
					Оплата не прошла. Автопродление отключено, автоматических повторов не будет. Уже
					оплаченный доступ сохраняется до конца периода. Повторить оплату можно на странице автора.
				</p>
			{:else if payment.status === 'review'}<Alert.Root
					><Alert.Title>Результат требует проверки</Alert.Title><Alert.Description
						>Новая оплата временно недоступна, пока уточняется результат предыдущей. Это защищает от
						повторного списания.</Alert.Description
					></Alert.Root
				>
			{:else if payment.confirmationUrl}<p>
					Откройте страницу ЮKassa и завершите оплату. После возвращения мы дождёмся подтверждения
					от платёжного сервиса.
				</p>
			{:else}<p role="status">
					<Spinner class="inline" /> Готовим страницу оплаты. Обычно это занимает несколько секунд.
				</p>{/if}
			{#if failure}<Alert.Root variant="destructive" role="alert"
					><Alert.Title>Не удалось обновить результат</Alert.Title><Alert.Description
						>{failure}</Alert.Description
					></Alert.Root
				>{/if}
		</Card.Content><Card.Footer class="flex flex-wrap gap-3">
			{#if payment.status === 'pending' && payment.confirmationUrl}<Button
					href={payment.confirmationUrl}
					rel="noreferrer">Оплатить в ЮKassa</Button
				>{/if}
			{#if waiting}<Button onclick={refresh} variant="outline" disabled={pending}
					>{#if pending}<Spinner data-icon="inline-start" />{/if}Обновить статус</Button
				>{/if}
			<Button
				href="/account/subscriptions"
				variant={payment.status === 'succeeded' ? 'default' : 'outline'}>Мои подписки</Button
			>
		</Card.Footer></Card.Root
	>
</section>
