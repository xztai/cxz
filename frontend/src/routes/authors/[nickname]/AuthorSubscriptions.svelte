<script lang="ts">
	import {
		api,
		ApiError,
		errorMessage,
		type Profile,
		type Subscription,
		type Tier
	} from '#lib/api/client.js';
	import Icon from '#lib/components/Icon.svelte';
	import { getSession } from '#lib/session.svelte.js';
	import { formatRubles } from '#lib/studio/format.js';
	import SubscribeDialog from './SubscribeDialog.svelte';

	let { profile, tiers }: { profile: Profile; tiers: Tier[] } = $props();
	const session = getSession();
	let viewerId = $derived(session.user?.id);
	let isOwner = $derived(viewerId === profile.id);
	let availableTiers = $derived(tiers.toSorted((a, b) => b.position - a.position));
	let selected = $state<Tier | null>(null);
	let checking = $state(false);
	let paymentsEnabled = $state<boolean | null>(null);
	let current = $state.raw<Subscription | null>(null);
	let message = $state('');
	let retry = $state(0);

	$effect(() => {
		const authorId = profile.id;
		const userId = viewerId;
		const needsCheck = !!userId && userId !== authorId;
		void retry;
		current = null;
		paymentsEnabled = null;
		message = '';
		checking = needsCheck;
		if (!needsCheck) return;

		const controller = new AbortController();
		void loadSubscription(authorId, controller.signal);
		return () => controller.abort();
	});

	async function loadSubscription(authorId: string, signal: AbortSignal) {
		try {
			let offset = 0;
			while (!signal.aborted) {
				const result = await api.subscriptions(offset, signal);
				if (signal.aborted) return;
				paymentsEnabled = result.paymentsEnabled;
				const subscription = result.subscriptions.find((item) => item.authorId === authorId);
				if (subscription) {
					if (subscription.active) current = subscription;
					return;
				}
				offset += result.subscriptions.length;
				if (!result.hasMore || !result.subscriptions.length || offset > 100000) return;
			}
		} catch (error) {
			if (signal.aborted) return;
			if (error instanceof ApiError && error.status === 401) void session.refresh();
			message = `Не удалось проверить подписку. ${errorMessage(error)}`;
		} finally {
			if (!signal.aborted) checking = false;
		}
	}
</script>

<section aria-labelledby="author-tiers-heading">
	<h2 id="author-tiers-heading">Уровни подписки</h2>
	{#if current && availableTiers.some((tier) => tier.id !== current?.tierId)}
		<p id="active-subscription-note" class="subscription-note">
			У вас уже есть действующая подписка на этого автора.
		</p>
	{/if}
	{#if current && !availableTiers.some((tier) => tier.id === current?.tierId)}
		<p class="archived-current"><strong>Ваш уровень</strong> {current.tierName}</p>
	{/if}
	{#if !availableTiers.length}
		<p class="simple-intro">У автора пока нет открытых уровней подписки.</p>
	{:else}
		<ul class="tiers">
			{#each availableTiers as tier (tier.id)}
				{@const isCurrent = tier.id === current?.tierId}
				<li class={['tier', { current: isCurrent }]}>
					<h3>{tier.name}</h3>
					<p class="price"><strong>{formatRubles(tier.priceKopecks)}</strong> <span>/ мес</span></p>
					{#if tier.description}<p class="description">{tier.description}</p>{/if}
					{#if isCurrent}
						<div class="tier-footer">
							<span class="current-label"><Icon name="check" size={16} />Ваш уровень</span>
						</div>
					{:else if !isOwner}
						<div class="tier-footer">
							<button
								class="button button-outline subscribe"
								type="button"
								disabled={!!current}
								aria-describedby={current ? 'active-subscription-note' : undefined}
								onclick={() => (selected = tier)}
							>
								Подписаться<span class="sr-only"> на уровень «{tier.name}»</span>
							</button>
						</div>
					{/if}
				</li>
			{/each}
		</ul>
	{/if}
	{#if message}
		<p class="feedback error-feedback" role="alert">{message}</p>
		<button class="button button-outline" type="button" onclick={() => retry++}>
			Проверить подписку ещё раз
		</button>
	{/if}
</section>

{#if selected}
	<SubscribeDialog
		{profile}
		tiers={availableTiers}
		initialTier={selected}
		{current}
		{checking}
		{paymentsEnabled}
		onClose={() => (selected = null)}
	/>
{/if}

<style>
	h2 {
		font-size: clamp(23px, 3vw, 29px);
		line-height: 1.2;
		font-weight: 800;
		letter-spacing: -0.025em;
		margin-bottom: 16px;
	}
	.tiers {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(min(100%, 272px), 1fr));
		gap: 16px;
		list-style: none;
		margin: 0;
		padding: 0;
	}
	.tier {
		display: flex;
		flex-direction: column;
		min-width: 0;
		padding: 20px;
		border: 1px solid var(--line);
		border-radius: 16px;
		overflow-wrap: anywhere;
	}
	.current {
		background: var(--blue-soft);
		border-color: #c9d4fb;
	}
	h3 {
		margin: 0 0 5px;
		font-size: 17px;
		line-height: 1.4;
		font-weight: 800;
	}
	.price {
		display: flex;
		align-items: baseline;
		flex-wrap: wrap;
		gap: 0 6px;
		margin: 0;
		font-variant-numeric: tabular-nums;
	}
	.price strong {
		font-weight: 800;
		font-size: 28px;
		letter-spacing: -0.025em;
	}
	.price span,
	.description {
		color: var(--muted);
		font-size: 14px;
	}
	.description {
		line-height: 1.6;
		white-space: pre-wrap;
		margin: 10px 0 0;
	}
	.current .price span,
	.current .description {
		color: #4d5c83;
	}
	.tier-footer {
		display: flex;
		align-items: center;
		padding-top: 18px;
		margin-top: auto;
	}
	.subscribe {
		width: 100%;
		min-height: 44px;
		padding: 8px 14px;
		color: var(--blue-hover);
	}
	.subscribe:disabled {
		border-color: var(--line);
		background: var(--surface);
		color: var(--muted);
		cursor: default;
		opacity: 1;
	}
	.subscription-note {
		margin: 0 0 16px;
		color: var(--muted);
		font-size: 14px;
	}
	.current-label {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		min-height: 44px;
		color: var(--blue-hover);
		font-size: 14px;
		font-weight: 800;
	}
	.archived-current {
		display: flex;
		flex-wrap: wrap;
		gap: 4px 10px;
		margin: 0 0 16px;
		font-size: 14px;
		overflow-wrap: anywhere;
	}
	.archived-current strong {
		color: var(--blue-hover);
	}
	@media (max-width: 640px) {
		.tiers {
			grid-template-columns: minmax(0, 1fr);
		}
	}
</style>
