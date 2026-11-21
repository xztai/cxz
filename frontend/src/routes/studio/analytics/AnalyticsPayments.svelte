<script lang="ts">
	import type { Analytics } from '#lib/api/client.js';
	import Icon from '#lib/components/Icon.svelte';
	import { formatRubles } from '#lib/studio/format.js';

	let { daily }: { daily: Analytics['daily'] } = $props();

	const shortDate = new Intl.DateTimeFormat('ru-RU', {
		day: 'numeric',
		month: 'short',
		timeZone: 'UTC'
	});
	const fullDate = new Intl.DateTimeFormat('ru-RU', { dateStyle: 'long', timeZone: 'UTC' });
	const compactRubles = new Intl.NumberFormat('ru-RU', {
		notation: 'compact',
		style: 'currency',
		currency: 'RUB',
		maximumFractionDigits: 2
	});
	const count = new Intl.NumberFormat('ru-RU');
	const levels = [4, 3, 2, 1, 0];
	const labelPositions = [0, 0.25, 0.5, 0.75, 1];
	const lastPayment = $derived(daily.findLast((day) => day.paymentCount > 0));
	let selected = $derived(lastPayment?.date);
	let hovered = $state('');
	let announcement = $state('');
	const selectedIndex = $derived(daily.findIndex((day) => day.date === selected));
	const displayDay = $derived(
		daily.find((day) => day.date === hovered) ?? daily.find((day) => day.date === selected)
	);
	const step = $derived.by(() => {
		const highest = Math.max(4, ...daily.map((day) => day.amountKopecks));
		const target = highest / 4;
		const scale = 10 ** Math.floor(Math.log10(target));
		return Math.ceil(target / scale) * scale;
	});
	const labels = $derived([
		...new Set(labelPositions.map((fraction) => Math.round(fraction * (daily.length - 1))))
	]);

	function dayLabel(day: Analytics['daily'][number]) {
		return `${fullDate.format(new Date(day.date))}. ${formatRubles(day.amountKopecks)}. Платежей ${count.format(day.paymentCount)}.`;
	}

	function stepDay(offset: number) {
		const day = daily[selectedIndex + offset];
		if (!day) return;
		hovered = '';
		selected = day.date;
		announcement = dayLabel(day);
	}

	function moveDay(event: KeyboardEvent & { currentTarget: HTMLButtonElement }, index: number) {
		let next: number;
		switch (event.key) {
			case 'ArrowLeft':
				next = Math.max(0, index - 1);
				break;
			case 'ArrowRight':
				next = Math.min(daily.length - 1, index + 1);
				break;
			case 'Home':
				next = 0;
				break;
			case 'End':
				next = daily.length - 1;
				break;
			default:
				return;
		}
		event.preventDefault();
		event.currentTarget.parentElement?.querySelectorAll('button')[next]?.focus();
	}
</script>

<section class="studio-panel payments" aria-labelledby="payments-heading">
	<header>
		<div>
			<h2 id="payments-heading">Платежи по дням</h2>
			<p>Сумма подтверждённых платежей</p>
		</div>
		{#if displayDay}
			<div class="day-detail">
				<div class="day-summary" aria-hidden="true">
					<time datetime={displayDay.date}>{fullDate.format(new Date(displayDay.date))}</time>
					<div>
						<strong>{formatRubles(displayDay.amountKopecks)}</strong><span
							>Платежей {count.format(displayDay.paymentCount)}</span
						>
					</div>
				</div>
				<div class="day-controls">
					<button
						class="button button-outline"
						type="button"
						aria-label="Предыдущий день"
						disabled={selectedIndex <= 0}
						onclick={() => stepDay(-1)}><Icon name="chevron" size={17} /></button
					>
					<button
						class="button button-outline"
						type="button"
						aria-label="Следующий день"
						disabled={selectedIndex >= daily.length - 1}
						onclick={() => stepDay(1)}><Icon name="chevron" size={17} /></button
					>
				</div>
				<span class="sr-only" role="status">{announcement}</span>
			</div>
		{/if}
	</header>

	{#if lastPayment}
		<div class="chart">
			<div class="y-axis" aria-hidden="true">
				{#each levels as level (level)}
					<span>{compactRubles.format((level * step) / 100)}</span>
				{/each}
			</div>
			<div class="plot">
				<div class="gridlines" aria-hidden="true">
					{#each levels as level (level)}<span></span>{/each}
				</div>
				<div
					class="bars"
					style:--day-count={daily.length}
					role="group"
					aria-label="Дневные платежи, UTC"
					onpointerleave={() => (hovered = '')}
				>
					{#each daily as day, index (day.date)}
						<button
							type="button"
							class={{ current: displayDay?.date === day.date }}
							aria-label={dayLabel(day)}
							tabindex={selected === day.date ? 0 : -1}
							onfocus={() => {
								hovered = '';
								selected = day.date;
							}}
							onclick={() => (selected = day.date)}
							onpointerenter={(event) => {
								if (event.pointerType === 'mouse') hovered = day.date;
							}}
							onkeydown={(event) => moveDay(event, index)}
						>
							<span
								class={{ paid: day.amountKopecks > 0 }}
								style:height={`${(100 * day.amountKopecks) / (4 * step)}%`}
							></span>
						</button>
					{/each}
				</div>
				<div class={['x-axis', { 'five-labels': labels.length === 5 }]} aria-hidden="true">
					{#each labels as index (index)}
						<span style:--position={`${(100 * (index + 0.5)) / daily.length}%`}>
							{shortDate.format(new Date(daily[index].date))}
						</span>
					{/each}
				</div>
			</div>
		</div>
	{:else}
		<p class="empty">За выбранный период платежей нет.</p>
	{/if}
</section>

<style>
	.payments {
		--plot-height: 168px;
	}
	header {
		display: flex;
		flex-wrap: wrap;
		align-items: start;
		justify-content: space-between;
		gap: 12px 18px;
	}
	h2 {
		font-size: 19px;
		font-weight: 800;
		line-height: 1.4;
	}
	header p,
	.day-summary,
	.empty {
		color: var(--muted);
		font-size: 13px;
	}
	header p {
		margin-top: 3px;
	}
	.day-detail {
		display: flex;
		align-items: center;
		gap: 10px;
	}
	.day-summary {
		min-width: 0;
		text-align: end;
		overflow-wrap: anywhere;
	}
	.day-summary div {
		display: flex;
		flex-wrap: wrap;
		align-items: baseline;
		justify-content: end;
		gap: 3px 12px;
		margin-top: 3px;
		font-variant-numeric: tabular-nums;
	}
	.day-summary strong {
		color: var(--ink);
		font-size: 15px;
		font-weight: 800;
	}
	.day-controls {
		display: flex;
		flex: none;
		gap: 5px;
	}
	.day-controls button {
		width: 44px;
		min-height: 44px;
		padding: 0;
	}
	.day-controls button:disabled {
		border-color: var(--line);
		background: var(--surface);
		cursor: default;
	}
	.day-controls button:first-child :global(svg) {
		transform: rotate(90deg);
	}
	.day-controls button:last-child :global(svg) {
		transform: rotate(-90deg);
	}
	.chart {
		display: grid;
		grid-template-columns: auto minmax(0, 1fr);
		gap: 10px;
		margin-top: 22px;
		color: var(--muted);
		font-size: 12px;
		font-variant-numeric: tabular-nums;
	}
	.y-axis {
		display: grid;
		grid-template-rows: repeat(4, 1fr) 0;
		height: var(--plot-height);
		text-align: end;
	}
	.y-axis span {
		height: max-content;
		transform: translateY(-50%);
		white-space: nowrap;
	}
	.plot {
		position: relative;
	}
	.gridlines {
		position: absolute;
		inset: 0 0 auto;
		display: flex;
		height: var(--plot-height);
		flex-direction: column;
		justify-content: space-between;
		pointer-events: none;
	}
	.gridlines span {
		border-top: 1px solid var(--line);
	}
	.bars {
		position: relative;
		display: grid;
		grid-template-columns: repeat(var(--day-count), minmax(0, 1fr));
		height: var(--plot-height);
	}
	.bars button {
		display: flex;
		min-width: 0;
		justify-content: center;
		align-items: end;
		padding: 0;
		border: 0;
		background: transparent;
		cursor: pointer;
	}
	.bars button:focus-visible {
		z-index: 1;
		outline-offset: -1px;
	}
	.bars button.current {
		background: var(--blue-soft);
	}
	.bars button span {
		width: min(85%, 42px);
		border-radius: 3px 3px 0 0;
		background: var(--blue);
	}
	.bars button span.paid {
		min-height: 2px;
	}
	.bars button.current span {
		background: var(--blue-hover);
	}
	.x-axis {
		position: relative;
		height: 1.4em;
		margin-top: 8px;
	}
	.x-axis span {
		position: absolute;
		left: var(--position);
		transform: translateX(-50%);
		white-space: nowrap;
	}
	.x-axis span:first-child:not(:only-child) {
		left: 0;
		transform: none;
	}
	.x-axis span:last-child:not(:only-child) {
		right: 0;
		left: auto;
		transform: none;
	}
	.empty {
		display: grid;
		min-height: 90px;
		place-items: center;
		margin-top: 15px;
		padding: 12px;
		border: 1px solid var(--line);
		border-radius: 9px;
		background: var(--surface);
		text-align: center;
	}
	@media (max-width: 640px) {
		.payments {
			--plot-height: 142px;
		}
		.day-detail {
			width: 100%;
			justify-content: space-between;
		}
		.day-summary {
			text-align: start;
		}
		.day-summary div {
			justify-content: start;
		}
		.five-labels span:nth-child(even) {
			display: none;
		}
	}
</style>
