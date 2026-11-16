<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import type { components } from '#lib/api/schema.js';
	import { api, checkError, errorMessage } from '#lib/api/client.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Spinner } from '#lib/components/ui/spinner/index.js';
	import * as Field from '#lib/components/ui/field/index.js';
	import * as NativeSelect from '#lib/components/ui/native-select/index.js';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { rubles } from '#lib/creators/prices.js';
	import { billingDate } from '#lib/billing/format.js';
	let { subscription: sub }: { subscription: components['schemas']['Subscription'] } = $props();
	let pending = $state(false);
	let failure = $state('');
	let message = $state('');
	let editing = $state(false);
	let tiers = $state.raw<components['schemas']['Tier'][]>([]);
	let selected = $state('');
	let selectedTier = $derived(tiers.find((t) => t.id === selected));
	async function cancel() {
		if (pending) return;
		pending = true;
		failure = '';
		message = '';
		try {
			const result = await api.POST('/me/subscriptions/{id}/cancel', {
				params: { path: { id: sub.id } }
			});
			checkError(result.error);
			await invalidateAll();
			message = result.data?.message ?? '';
			editing = false;
		} catch (cause) {
			failure = errorMessage(cause);
		} finally {
			pending = false;
		}
	}
	async function edit() {
		if (pending) return;
		pending = true;
		failure = '';
		try {
			const result = await api.GET('/authors/{nickname}', {
				params: { path: { nickname: sub.nickname } }
			});
			checkError(result.error);
			tiers = result.data?.tiers.filter((t) => t.priceKopecks >= 100) ?? [];
			selected = sub.nextTierId ?? sub.tierId;
			editing = true;
		} catch (cause) {
			failure = errorMessage(cause);
		} finally {
			pending = false;
		}
	}
	async function change(event: SubmitEvent) {
		event.preventDefault();
		if (pending || !selectedTier) return;
		pending = true;
		failure = '';
		message = '';
		try {
			const result = await api.PUT('/me/subscriptions/{id}/tier', {
				params: { path: { id: sub.id } },
				body: { tierId: selectedTier.id, priceKopecks: selectedTier.priceKopecks }
			});
			checkError(result.error);
			await invalidateAll();
			editing = false;
			message = result.data?.message ?? '';
		} catch (cause) {
			failure = errorMessage(cause);
		} finally {
			pending = false;
		}
	}
</script>

<article class="flex flex-col gap-5 border-t border-border py-7">
	<div class="flex flex-wrap items-start justify-between gap-4">
		<div>
			<h2><a href={`/authors/${sub.nickname}`}>{sub.displayName}</a></h2>
			<p>{sub.tierName} · {rubles(sub.priceKopecks)} / месяц</p>
		</div>
		<Badge variant="outline">{sub.active ? 'Доступ открыт' : 'Нет оплаченного доступа'}</Badge>
	</div>
	<div class="text-muted-foreground">
		<p>Доступ до: {billingDate(sub.paidUntil)}</p>
		<p>{sub.autoRenew ? 'Автопродление включено' : 'Автопродление отключено'}</p>
		{#if sub.nextTierId}<p>
				Со следующего периода: {sub.nextTierName} · {rubles(sub.nextPriceKopecks ?? 0)} / месяц
			</p>{/if}
		{#if sub.stoppedReason === 'payment_canceled' || sub.stoppedReason === 'provider_rejected'}<p>
				Оплата не прошла. Автоматических повторов не будет. Для повторной оплаты выберите уровень на
				странице автора.
			</p>{:else if sub.stoppedReason === 'tier_archived'}<p>
				Уровень архивирован. Новых списаний не будет.
			</p>{/if}
	</div>
	<div class="flex flex-wrap gap-3">
		<Button href={`/authors/${sub.nickname}`} variant="outline"
			>{sub.active ? 'Читать автора' : 'Выбрать уровень и оплатить'}</Button
		>
		{#if sub.autoRenew}<Button variant="outline" onclick={cancel} disabled={pending}
				>{#if pending}<Spinner data-icon="inline-start" />{/if}Отключить автопродление</Button
			>{#if sub.active}<Button variant="link" onclick={edit} disabled={pending}
					>Сменить уровень</Button
				>{/if}{/if}
	</div>
	{#if editing}<form method="POST" onsubmit={change} class="max-w-xl">
			<Field.FieldGroup
				><Field.Field
					><Field.FieldLabel for={`tier-${sub.id}`}>Уровень на следующий период</Field.FieldLabel>
					<NativeSelect.Root
						id={`tier-${sub.id}`}
						bind:value={selected}
						disabled={pending}
						class="w-full"
						>{#each tiers as tier (tier.id)}<NativeSelect.Option value={tier.id}
								>{tier.name} · {rubles(tier.priceKopecks)}</NativeSelect.Option
							>{/each}</NativeSelect.Root
					>
					<Field.FieldDescription
						>Текущий доступ не изменится. Со следующего периода будет списываться {selectedTier
							? rubles(selectedTier.priceKopecks)
							: 'выбранная сумма'} в месяц, без перерасчёта текущего периода.</Field.FieldDescription
					></Field.Field
				><Field.Field orientation="horizontal"
					><Button type="submit" disabled={pending || !selectedTier}
						>Подтвердить цену и смену уровня</Button
					><Button type="button" variant="outline" onclick={() => (editing = false)}>Закрыть</Button
					></Field.Field
				></Field.FieldGroup
			>
		</form>{/if}
	{#if failure}<Alert.Root variant="destructive" role="alert"
			><Alert.Title>Не удалось изменить подписку</Alert.Title><Alert.Description
				>{failure}</Alert.Description
			></Alert.Root
		>{/if}
	{#if message}<p role="status">{message}</p>{/if}
</article>
