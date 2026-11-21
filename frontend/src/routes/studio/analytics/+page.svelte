<script lang="ts">
	import { invalidate } from '$app/navigation';
	import { resolve } from '$app/paths';
	import Icon from '#lib/components/Icon.svelte';
	import { formatRubles } from '#lib/studio/format.js';
	import AnalyticsPayments from './AnalyticsPayments.svelte';
	import type { PageProps } from './$types';
	import { maxDate, minDate, presets, validPeriod } from './period.js';

	let { data }: PageProps = $props();
	let from = $derived(data.period.from);
	let to = $derived(data.period.to);
	let error = $derived(data.invalidPeriod ? 'Некорректные даты в ссылке. Показаны 30 дней.' : '');
	let retrying = $state(false);
	const count = new Intl.NumberFormat('ru-RU');
	const timestamp = new Intl.DateTimeFormat('ru-RU', {
		dateStyle: 'medium',
		timeStyle: 'short',
		timeZone: 'UTC'
	});
	const dependency = 'studio:analytics';

	function applyPeriod(event: SubmitEvent) {
		if (!validPeriod(from, to)) {
			event.preventDefault();
			error =
				from > to
					? 'Дата начала должна быть не позже даты окончания.'
					: 'Выберите период от 1 до 366 дней.';
			return;
		}
		error = '';
	}

	async function retry() {
		retrying = true;
		try {
			await invalidate(dependency);
		} finally {
			retrying = false;
		}
	}
</script>

<svelte:head><title>Аналитика — etheyl</title></svelte:head>

<header class="page-header">
	<div class="page-title">
		<h1 class="studio-heading">Аналитика</h1>
		{#await data.analytics then report}
			{#if report?.test}<span class="test-mode">Тестовый режим</span>{/if}
		{/await}
	</div>
	{#await data.analytics then report}
		{#if report}
			<p class="updated">
				Данные обновлены <time datetime={report.asOf}
					>{timestamp.format(new Date(report.asOf))} UTC</time
				>
			</p>
		{/if}
	{/await}
</header>

{#snippet cards(amount?: number, payments?: number, active?: number)}
	<dl class="metrics" aria-busy={amount === undefined}>
		<div class="metric">
			<dt><Icon name="wallet" size={18} /> Сумма платежей</dt>
			<dd>{amount === undefined ? '—' : formatRubles(amount)}</dd>
		</div>
		<div class="metric">
			<dt><Icon name="analytics" size={18} /> Число платежей</dt>
			<dd>{payments === undefined ? '—' : count.format(payments)}</dd>
		</div>
		<div class="metric">
			<dt><Icon name="users" size={18} /> Активные подписки</dt>
			<dd>
				{active === undefined ? '—' : count.format(active)}
				<span class="metric-note">На момент обновления</span>
			</dd>
		</div>
	</dl>
{/snippet}

{#if !data.analytics}
	<div class="studio-panel unavailable">
		<p>Аналитика станет доступна после сохранения профиля автора.</p>
		<a class="studio-link" href={resolve('/studio/profile')}>Открыть профиль</a>
	</div>
{:else}
	<section class="studio-panel period" aria-labelledby="period-heading">
		<div class="period-start">
			<h2 id="period-heading">Период <span>UTC</span></h2>
			<nav class="presets" aria-label="Быстрый выбор периода">
				{#each presets as days (days)}
					<a
						href={resolve(`/studio/analytics?days=${days}`)}
						data-sveltekit-reset="false"
						data-sveltekit-preload-data="false"
						aria-current={days === data.preset ? 'true' : undefined}
						onclick={() => {
							from = data.period.from;
							to = data.period.to;
							error = '';
						}}>{days} дней</a
					>
				{/each}
			</nav>
		</div>
		<form
			action={resolve('/studio/analytics')}
			method="get"
			data-sveltekit-reset="false"
			onsubmit={applyPeriod}
			oninput={() => (error = '')}
		>
			<label>
				<span>С</span>
				<input
					type="date"
					name="from"
					aria-label="Дата начала"
					aria-describedby={error ? 'period-error' : undefined}
					aria-invalid={!!error}
					min={minDate}
					max={maxDate}
					required
					bind:value={from}
				/>
			</label>
			<label>
				<span>По</span>
				<input
					type="date"
					name="to"
					aria-label="Дата окончания"
					aria-describedby={error ? 'period-error' : undefined}
					aria-invalid={!!error}
					min={minDate}
					max={maxDate}
					required
					bind:value={to}
				/>
			</label>
			<button class="button button-outline apply" type="submit">Применить</button>
		</form>
		{#if error}<p class="period-error" id="period-error" role="alert">{error}</p>{/if}
	</section>

	{#await data.analytics}
		{@render cards()}
		<div class="studio-panel loading" role="status">Загружаем аналитику…</div>
	{:then report}
		{#if report}
			{@render cards(report.amountKopecks, report.paymentCount, report.activeSubscriptions)}
			<AnalyticsPayments daily={report.daily} />

			<section class="studio-panel tiers" aria-labelledby="tiers-heading">
				<h2 id="tiers-heading">Подписчики по уровням</h2>
				{#if report.tiers.length}
					<table aria-labelledby="tiers-heading">
						<thead>
							<tr><th scope="col">Уровень</th><th scope="col">Активные подписки</th></tr>
						</thead>
						<tbody>
							{#each report.tiers as tier (tier.id)}
								<tr>
									<th scope="row">
										<span class="tier-name">{tier.name}</span>
										{#if tier.archived}<span class="archived">В архиве</span>{/if}
									</th>
									<td>{count.format(tier.activeSubscriptions)}</td>
								</tr>
							{/each}
						</tbody>
					</table>
				{:else}
					<p class="empty">Уровней подписки пока нет.</p>
				{/if}
			</section>
		{:else}
			<div class="studio-panel unavailable">
				<p role="status">Не удалось загрузить аналитику.</p>
				<button class="button button-outline" type="button" disabled={retrying} onclick={retry}>
					{retrying ? 'Загружаем…' : 'Повторить'}
				</button>
			</div>
		{/if}
	{/await}
{/if}

<style>
	.page-header,
	.page-title {
		display: flex;
		align-items: center;
		flex-wrap: wrap;
		gap: 8px 14px;
	}
	.page-header {
		justify-content: space-between;
		margin-bottom: 22px;
	}
	.test-mode,
	.archived {
		border-radius: 6px;
		padding: 3px 8px;
		font-size: 12px;
		font-weight: 750;
		white-space: nowrap;
	}
	.test-mode {
		background: var(--blue-soft);
		color: var(--blue-hover);
	}
	.updated,
	.metric-note,
	.empty,
	.loading {
		color: var(--muted);
		font-size: 13px;
	}
	time {
		white-space: nowrap;
		font-variant-numeric: tabular-nums;
	}
	.studio-panel.period {
		display: flex;
		flex-wrap: wrap;
		align-items: end;
		justify-content: space-between;
		gap: 12px 20px;
		padding: 15px 18px;
	}
	.period h2 {
		display: flex;
		align-items: center;
		gap: 8px;
		margin-bottom: 7px;
		font-size: 14px;
		font-weight: 800;
	}
	.period h2 span {
		color: var(--muted);
		font-size: 11px;
		font-weight: 800;
	}
	.presets {
		display: flex;
		gap: 2px;
		padding: 3px;
		border: 1px solid var(--line);
		border-radius: 10px;
		background: var(--surface);
	}
	.presets a {
		display: grid;
		min-height: 34px;
		place-items: center;
		padding: 0 12px;
		border-radius: 7px;
		color: var(--muted);
		font-size: 13px;
		font-weight: 750;
		text-decoration: none;
		white-space: nowrap;
	}
	.presets a:hover,
	.presets [aria-current] {
		background: var(--blue-soft);
		color: var(--blue-hover);
	}
	.presets [aria-current] {
		box-shadow: inset 0 0 0 1px var(--blue);
	}
	form {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 8px 10px;
	}
	label {
		display: flex;
		align-items: center;
		gap: 7px;
		color: var(--muted);
		font-size: 13px;
		font-weight: 750;
	}
	input {
		min-width: 0;
		min-height: 42px;
		padding: 7px 9px;
		border: 1px solid #aeb6ca;
		border-radius: 9px;
		background: var(--white);
		color: var(--ink);
		font: inherit;
		font-variant-numeric: tabular-nums;
	}
	input[aria-invalid='true'] {
		border-color: var(--error);
	}
	.apply {
		min-height: 42px;
		padding: 7px 14px;
		font-size: 13px;
	}
	.period-error {
		flex-basis: 100%;
		color: var(--error);
		font-size: 13px;
	}
	.metrics {
		display: grid;
		grid-template-columns: repeat(3, minmax(0, 1fr));
		gap: 14px;
		margin: 14px 0 20px;
	}
	.metric {
		padding: 16px 20px;
		border: 1px solid var(--line);
		border-radius: 16px;
		background: var(--white);
	}
	dt {
		display: flex;
		align-items: center;
		gap: 8px;
		color: var(--muted);
		font-size: 13px;
		font-weight: 750;
	}
	dt :global(svg) {
		flex: none;
	}
	dd {
		margin: 5px 0 0;
		font-size: 28px;
		font-weight: 800;
		font-variant-numeric: tabular-nums;
		letter-spacing: -0.025em;
		line-height: 1.25;
		overflow-wrap: anywhere;
	}
	.metric-note {
		display: block;
		margin-top: 3px;
		font-size: 12px;
		font-weight: 500;
		letter-spacing: normal;
		line-height: 1.5;
	}
	.tiers,
	.unavailable {
		margin-top: 16px;
	}
	.tiers h2 {
		font-size: 19px;
		font-weight: 800;
		line-height: 1.4;
	}
	.tiers .empty {
		margin-top: 10px;
	}
	table {
		width: 100%;
		margin-top: 10px;
		border-collapse: collapse;
		font-size: 14px;
		text-align: start;
	}
	th,
	td {
		padding: 9px 0;
		border-bottom: 1px solid var(--line);
		text-align: start;
	}
	thead th {
		color: var(--muted);
		font-size: 13px;
		font-weight: 750;
	}
	tbody th {
		font-weight: 750;
	}
	tr > :last-child {
		width: 1%;
		padding-inline-start: 16px;
		text-align: end;
		font-variant-numeric: tabular-nums;
	}
	tbody tr:last-child > * {
		padding-bottom: 0;
		border: 0;
	}
	.tier-name {
		overflow-wrap: anywhere;
	}
	.archived {
		display: inline-block;
		margin-inline-start: 5px;
		background: var(--surface);
		color: var(--muted);
	}
	@media (min-width: 641px) {
		thead tr > :last-child {
			white-space: nowrap;
		}
	}
	.unavailable {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 10px 18px;
		font-size: 14px;
	}
	@media (max-width: 640px) {
		.page-header {
			margin-bottom: 16px;
		}
		.studio-panel.period {
			display: grid;
			justify-content: stretch;
			padding: 14px;
		}
		.presets a {
			flex: 1;
			min-height: 42px;
			padding-inline: 8px;
		}
		form {
			display: grid;
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}
		label {
			display: grid;
			gap: 3px;
		}
		input {
			width: 100%;
			min-height: 44px;
			padding-inline: 7px;
		}
		.apply {
			grid-column: 1 / -1;
			min-height: 44px;
		}
		.metrics {
			grid-template-columns: repeat(2, minmax(0, 1fr));
			gap: 8px;
		}
		.metric {
			padding: 12px 14px;
		}
		.metric:last-child {
			grid-column: 1 / -1;
		}
		dd {
			font-size: 23px;
		}
	}
</style>
