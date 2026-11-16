<script lang="ts">
	import { goto } from '$app/navigation';
	import { api, checkError, errorMessage } from '#lib/api/client.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Checkbox } from '#lib/components/ui/checkbox/index.js';
	import { Spinner } from '#lib/components/ui/spinner/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Field from '#lib/components/ui/field/index.js';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { rubles } from '#lib/creators/prices.js';
	import type { PageProps } from './$types';
	let { data }: PageProps = $props();
	let autoRenew = $state(false);
	let pending = $state(false);
	let failure = $state('');
	async function checkout(event: SubmitEvent) {
		event.preventDefault();
		if (pending) return;
		pending = true;
		failure = '';
		try {
			const result = await api.POST('/billing/checkout', {
				body: { tierId: data.tier.id, priceKopecks: data.tier.priceKopecks, autoRenew }
			});
			checkError(result.error);
			if (!result.data) throw new Error('Missing payment');
			await goto(`/account/payments/${result.data.payment.id}`);
		} catch (cause) {
			failure = errorMessage(cause);
		} finally {
			pending = false;
		}
	}
</script>

<svelte:head
	><title>Подписка на {data.profile.displayName} — etheyl</title><meta
		name="robots"
		content="noindex"
	/></svelte:head
>
<section class="workspace-page page-inset">
	<a class="back-link" href={`/authors/${data.profile.nickname}`}>К автору</a>
	<div class="workspace-heading">
		<div>
			<h1>Поддержать автора</h1>
			<p>{data.profile.displayName}</p>
		</div>
	</div>
	<Card.Root class="max-w-xl">
		<Card.Header
			><Card.Title><h2>{data.tier.name}</h2></Card.Title><Card.Description
				>{data.tier.description ||
					'Доступ к публикациям этого и младших уровней.'}</Card.Description
			></Card.Header
		>
		<Card.Content class="flex flex-col gap-6">
			<p><strong>{rubles(data.tier.priceKopecks)}</strong> за календарный месяц</p>
			<Alert.Root
				><Alert.Title>Тестовая оплата</Alert.Title><Alert.Description
					>Используется тестовый магазин ЮKassa. Вводите тестовые платёжные данные. Реальные выплаты
					авторам не выполняются.</Alert.Description
				></Alert.Root
			>
			<form id="checkout" method="POST" onsubmit={checkout}>
				<Field.FieldGroup>
					<Field.Field orientation="horizontal">
						<Checkbox id="auto-renew" bind:checked={autoRenew} disabled={pending} />
						<Field.FieldContent
							><Field.FieldLabel for="auto-renew">Согласен на автопродление</Field.FieldLabel
							><Field.FieldDescription
								>Разрешаю сохранить способ оплаты и списывать {rubles(data.tier.priceKopecks)} каждый
								календарный месяц. Отмена доступна в кабинете; оплаченный доступ сохраняется.</Field.FieldDescription
							></Field.FieldContent
						>
					</Field.Field>
					<Field.FieldDescription
						>Без согласия — одна оплата за месяц, без автоматических списаний. При смене уровня вы
						отдельно подтвердите новую цену. Возврат с ЮKassa сам по себе не подтверждает оплату.</Field.FieldDescription
					>
				</Field.FieldGroup>
			</form>
			{#if failure}<Alert.Root variant="destructive" role="alert"
					><Alert.Title>Не удалось оформить подписку</Alert.Title><Alert.Description
						>{failure}</Alert.Description
					></Alert.Root
				>{/if}
		</Card.Content>
		<Card.Footer
			><Button type="submit" form="checkout" disabled={pending || data.tier.priceKopecks < 100}
				>{#if pending}<Spinner data-icon="inline-start" />{/if}Перейти к оплате · {rubles(
					data.tier.priceKopecks
				)}</Button
			></Card.Footer
		>
	</Card.Root>
</section>
