<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import {
		api,
		ApiError,
		errorMessage,
		type Profile,
		type Subscription,
		type Tier
	} from '#lib/api/client.js';
	import { getSession } from '#lib/session.svelte.js';
	import { formatRubles } from '#lib/studio/format.js';

	let {
		profile,
		tiers,
		initialTier,
		current,
		checking,
		paymentsEnabled,
		onClose
	}: {
		profile: Profile;
		tiers: Tier[];
		initialTier: Tier;
		current: Subscription | null;
		checking: boolean;
		paymentsEnabled: boolean | null;
		onClose: () => void;
	} = $props();
	const session = getSession();
	let selectedId = $state<string>();
	let autoRenew = $state(false);
	let pending = $state(false);
	let message = $state('');
	let dialog: HTMLDialogElement;
	let tier = $derived(tiers.find((option) => option.id === selectedId) ?? initialTier);
	let owner = $derived(session.user?.id === profile.id);
	let blocked = $derived.by(() => {
		if (owner) return 'Вы смотрите свой профиль.';
		if (current) return 'У вас уже есть действующая подписка на этого автора.';
		if (paymentsEnabled === false) return 'Оплата пока недоступна.';
		if (tier.priceKopecks < 100) return 'Минимальная сумма оплаты — 1 ₽.';
		return '';
	});
	function openDialog(element: HTMLDialogElement) {
		dialog = element;
		element.showModal();
		return () => {
			if (element.open) element.close();
		};
	}

	async function checkout(event: SubmitEvent) {
		event.preventDefault();
		const userId = session.user?.id;
		if (!userId || checking || pending || blocked) return;
		message = '';
		pending = true;
		try {
			const { payment } = await api.checkout({
				tierId: tier.id,
				priceKopecks: tier.priceKopecks,
				autoRenew
			});
			if (!dialog.open || session.user?.id !== userId) return;
			if (payment.status === 'pending' && payment.confirmationUrl) {
				window.location.assign(payment.confirmationUrl);
			} else {
				await goto(resolve('/account/payments/[id]', { id: payment.id }));
			}
		} catch (error) {
			if (!dialog.open || session.user?.id !== userId) return;
			if (error instanceof ApiError && error.status === 401) {
				message = 'Войдите снова, чтобы продолжить оплату.';
				void session.refresh();
			} else {
				message = errorMessage(error);
			}
		} finally {
			pending = false;
		}
	}
</script>

<dialog
	{@attach openDialog}
	aria-labelledby="subscribe-title"
	aria-describedby="subscribe-author"
	onclose={onClose}
	oncancel={(event) => {
		if (pending) event.preventDefault();
	}}
>
	<form method="POST" onsubmit={checkout} aria-busy={pending}>
		<h2 id="subscribe-title">Оформить подписку</h2>
		<p id="subscribe-author" class="author">{profile.displayName}</p>
		<div class="field">
			<label for="subscribe-tier">Уровень подписки</label>
			<select
				id="subscribe-tier"
				value={tier.id}
				disabled={pending}
				onchange={(event) => {
					selectedId = event.currentTarget.value;
					message = '';
				}}
			>
				{#each tiers as option (option.id)}
					<option value={option.id}>{option.name}</option>
				{/each}
			</select>
		</div>
		<div class="total">
			<span>К оплате за месяц</span>
			<output for="subscribe-tier" aria-live="polite">{formatRubles(tier.priceKopecks)}</output>
		</div>
		<label class="renewal">
			<input type="checkbox" bind:checked={autoRenew} disabled={pending || !!current || owner} />
			<span>
				<strong>Автопродление</strong>
				<span class="hint">Списывать эту сумму ежемесячно.</span>
			</span>
		</label>
		{#if message}<p class="feedback error-feedback" role="alert">{message}</p>{/if}
		{#if blocked}
			<p class="hint" role="status">{blocked}</p>
		{:else if session.user && !checking}
			<p class="hint">Оплата проходит в тестовом режиме.</p>
		{/if}
		<div class="actions">
			<button
				class="button button-outline"
				type="button"
				aria-disabled={pending}
				onclick={() => {
					if (!pending) dialog.close();
				}}
			>
				Отмена
			</button>
			{#if session.ready && !session.user && !blocked}
				<a
					class="button button-primary"
					href="{resolve('/login')}?{new URLSearchParams({ author: profile.nickname })}"
				>
					Войти для оплаты
				</a>
			{:else}
				<button
					class="button button-primary"
					type="submit"
					aria-disabled={pending || checking || !session.ready || !!blocked}
				>
					{pending ? 'Готовим оплату…' : checking ? 'Проверяем подписку…' : 'Перейти к оплате'}
				</button>
			{/if}
		</div>
	</form>
</dialog>

<style>
	dialog {
		inset: 0;
		width: min(464px, calc(100% - 32px));
		max-height: calc(100% - 32px);
		margin: auto;
		padding: 26px;
		border: 1px solid var(--line);
		border-radius: 16px;
		background: var(--white);
		color: var(--ink);
		overflow-y: auto;
		overflow-wrap: anywhere;
	}
	dialog::backdrop {
		background: #171a247a;
	}
	h2 {
		margin: 0;
		font-size: 24px;
		font-weight: 800;
		letter-spacing: -0.025em;
		line-height: 1.25;
	}
	.author {
		margin: 6px 0 24px;
		color: var(--muted);
		font-size: 14px;
	}
	select {
		width: 100%;
		min-height: 48px;
		padding: 10px 32px 10px 12px;
		border: 1px solid #aeb6ca;
		border-radius: 9px;
		background: var(--white);
		font-size: 16px;
		text-overflow: ellipsis;
	}
	.total {
		display: flex;
		align-items: baseline;
		justify-content: space-between;
		flex-wrap: wrap;
		gap: 4px 12px;
		padding: 12px 0;
		border-block: 1px solid var(--line);
		font-size: 14px;
		font-weight: 700;
	}
	output {
		font-size: 22px;
		font-weight: 800;
		font-variant-numeric: tabular-nums;
	}
	.renewal {
		display: flex;
		align-items: start;
		gap: 10px;
		margin: 16px 0;
		font-size: 14px;
		cursor: pointer;
	}
	.renewal input {
		flex: 0 0 auto;
		width: 18px;
		height: 18px;
		margin-top: 2px;
		accent-color: var(--blue);
	}
	.hint {
		display: block;
		color: var(--muted);
		font-size: 13px;
		line-height: 1.5;
	}
	.actions {
		display: flex;
		flex-wrap: wrap;
		justify-content: flex-end;
		gap: 8px;
		margin-top: 20px;
	}
	.button[aria-disabled='true'] {
		opacity: 0.65;
		cursor: default;
	}
	@media (max-width: 440px) {
		dialog {
			padding: 20px;
		}
		.actions > .button {
			flex: 1 0 auto;
			padding-inline: 12px;
		}
	}
</style>
