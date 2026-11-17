<script lang="ts">
	import { invalidate } from '$app/navigation';
	import type { Analytics } from '#lib/api/client.js';
	import Icon from '#lib/components/Icon.svelte';
	import { formatRubles } from '#lib/studio/format.js';

	let {
		analytics,
		dependency,
		heading = 'h2'
	}: {
		analytics: Promise<Analytics | null> | null;
		dependency: string;
		heading?: 'h1' | 'h2';
	} = $props();
	let retrying = $state(false);
	const count = new Intl.NumberFormat('ru-RU');

	async function retry() {
		retrying = true;
		try {
			await invalidate(dependency);
		} finally {
			retrying = false;
		}
	}
</script>

{#snippet title(test = false)}
	<div class="section-heading">
		<svelte:element
			this={heading}
			class={heading === 'h1' ? 'studio-heading' : 'title'}
			id="analytics-heading"
		>
			Аналитика
		</svelte:element>
		{#if test}
			<span class="test-mode">Тестовый режим</span>
		{/if}
	</div>
{/snippet}

{#snippet cards(revenue?: string, subscribers?: string, loading = false)}
	<dl class="metrics" aria-busy={loading}>
		<div class="metric">
			<dt><Icon name="wallet" size={18} /> Доход за 30 дней</dt>
			<dd class={{ pending: loading }}>{revenue ?? '—'}</dd>
		</div>
		<div class="metric">
			<dt><Icon name="users" size={18} /> Активные подписки</dt>
			<dd class={{ pending: loading }}>{subscribers ?? '—'}</dd>
		</div>
	</dl>
{/snippet}

<section aria-labelledby="analytics-heading">
	{#if !analytics}
		{@render title()}
		<p class="unavailable">Аналитика станет доступна после сохранения профиля автора.</p>
		{@render cards()}
	{:else}
		{#await analytics}
			{@render title()}
			{@render cards(undefined, undefined, true)}
		{:then result}
			{@render title(result?.test)}
			{#if result && Number.isFinite(result.amountKopecks) && Number.isFinite(result.activeSubscriptions)}
				{@render cards(
					formatRubles(result.amountKopecks),
					count.format(result.activeSubscriptions)
				)}
			{:else}
				<div class="studio-panel error-state">
					<p role="status">Не удалось загрузить аналитику.</p>
					<button class="button button-outline" type="button" disabled={retrying} onclick={retry}>
						{retrying ? 'Загружаем…' : 'Повторить'}
					</button>
				</div>
			{/if}
		{/await}
	{/if}
</section>

<style>
	.section-heading {
		display: flex;
		align-items: center;
		flex-wrap: wrap;
		gap: 8px 14px;
		margin-bottom: 14px;
	}
	.title {
		font-size: 22px;
		font-weight: 800;
		line-height: 1.3;
		letter-spacing: -0.025em;
	}
	.test-mode {
		background: var(--blue-soft);
		border-radius: 6px;
		padding: 3px 8px;
		color: var(--blue-hover);
		font-size: 12px;
		font-weight: 750;
	}
	.metrics {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 14px;
		margin: 0;
	}
	.metric {
		border: 1px solid var(--line);
		border-radius: 16px;
		padding: 18px 22px;
		background: var(--white);
	}
	dt {
		display: flex;
		align-items: center;
		gap: 9px;
		color: var(--muted);
		font-size: 14px;
		font-weight: 750;
	}
	dt :global(svg) {
		flex: none;
	}
	dd {
		margin: 9px 0 0;
		font-size: 28px;
		font-weight: 800;
		letter-spacing: -0.025em;
		line-height: 1.25;
		font-variant-numeric: tabular-nums;
		overflow-wrap: anywhere;
	}
	.pending {
		width: min(95px, 75%);
		border-radius: 6px;
		background: var(--blue-soft);
		color: transparent;
	}
	.unavailable {
		margin: -3px 0 14px;
		color: var(--muted);
		font-size: 14px;
		line-height: 1.65;
	}
	.error-state {
		display: flex;
		align-items: center;
		flex-wrap: wrap;
		gap: 14px 24px;
		justify-content: space-between;
	}
	.error-state p {
		color: var(--muted);
		font-size: 14px;
	}
	.error-state button {
		min-height: 44px;
		padding-block: 8px;
	}
	@media (max-width: 640px) {
		.metrics {
			grid-template-columns: minmax(0, 1fr);
			gap: 10px;
		}
		.metric {
			padding: 16px 18px;
		}
	}
</style>
