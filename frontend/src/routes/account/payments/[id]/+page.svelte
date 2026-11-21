<script lang="ts">
	import { resolve } from '$app/paths';
	import { api, ApiError, errorMessage, type Payment } from '#lib/api/client.js';
	import SiteHeader from '#lib/components/SiteHeader.svelte';
	import { getSession } from '#lib/session.svelte.js';
	import { formatRubles } from '#lib/studio/format.js';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const session = getSession();
	let payment = $state.raw<Payment | null>(null);
	let message = $state('');
	let retry = $state(0);
	const statusText: Record<Payment['status'], string> = {
		queued: 'Готовим ссылку на оплату',
		pending: 'Ожидаем подтверждение оплаты',
		succeeded: 'Подписка оплачена',
		canceled: 'Платёж отменён',
		review: 'Платёж на проверке'
	};
	let canRefresh = $derived(
		!!message ||
			payment?.status === 'pending' ||
			payment?.status === 'queued' ||
			payment?.status === 'review'
	);

	$effect(() => {
		const id = data.id;
		const user = session.user?.id;
		void retry;
		payment = null;
		message = '';
		if (!user) return;
		const controller = new AbortController();
		let refreshTimer: ReturnType<typeof setTimeout>;
		let checksRemaining = 10;

		async function check() {
			try {
				const result = await api.payment(id, controller.signal);
				if (controller.signal.aborted) return;
				payment = result.payment;
				if (
					(payment.status === 'queued' || payment.status === 'pending') &&
					checksRemaining-- > 0
				) {
					refreshTimer = setTimeout(check, 3000);
				}
			} catch (error) {
				if (controller.signal.aborted) return;
				if (error instanceof ApiError && error.status === 401) void session.refresh();
				message =
					error instanceof ApiError && error.status === 404
						? 'Платёж не найден.'
						: errorMessage(error);
			}
		}

		void check();
		return () => {
			controller.abort();
			clearTimeout(refreshTimer);
		};
	});
</script>

<svelte:head>
	<title>Статус оплаты — etheyl</title>
	<meta name="robots" content="noindex" />
	<meta name="referrer" content="no-referrer" />
</svelte:head>

<SiteHeader />
<main id="main" class="shell simple-page">
	<div class="payment-status">
		<h1>Статус оплаты</h1>
		<div class="summary" role="status">
			{#if payment}
				<p><strong>{statusText[payment.status]}</strong></p>
				<p>{formatRubles(payment.amountKopecks)} · тестовый платёж</p>
			{:else if !session.ready || (session.user && !message)}
				<p>Проверяем статус…</p>
			{:else if !session.user}
				<p>Войдите, чтобы проверить статус платежа.</p>
			{/if}
		</div>
		{#if message}<p class="feedback error-feedback" role="alert">{message}</p>{/if}
		<div class="actions">
			{#if payment?.status === 'pending' && payment.confirmationUrl}
				<a class="button button-primary" href={payment.confirmationUrl} rel="external noreferrer">
					Продолжить оплату
				</a>
			{:else if session.ready && !session.user}
				<a class="button button-primary" href={resolve('/login')}>Войти</a>
			{/if}
			{#if canRefresh}
				<button class="button button-outline" type="button" onclick={() => retry++}>
					Проверить статус
				</button>
			{/if}
			<a class="button button-outline" href={resolve('/authors')}>Все авторы</a>
		</div>
	</div>
</main>

<style>
	.payment-status {
		width: min(100%, 560px);
		margin-inline: auto;
	}
	h1 {
		margin-bottom: 16px;
	}
	.summary {
		color: var(--muted);
		font-size: 15px;
	}
	.summary p {
		margin-bottom: 6px;
	}
	.summary strong {
		color: var(--ink);
		font-size: 18px;
		font-weight: 800;
	}
	.actions {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
		margin-top: 20px;
	}
</style>
