<script lang="ts">
	import { BarChart } from 'layerchart/svg';
	import * as Chart from '#lib/components/ui/chart/index.js';
	import type { components } from '#lib/api/schema.js';
	import { rubles } from '#lib/creators/prices.js';
	let { daily }: { daily: components['schemas']['AnalyticsDay'][] } = $props();
	const config = {
		amountKopecks: { label: 'Тестовые платежи, ₽', color: 'var(--primary)' }
	} satisfies Chart.ChartConfig;
	let ticks = $derived(
		daily
			.filter(
				(_, i) => i % Math.max(1, Math.ceil((daily.length - 1) / 3)) === 0 || i === daily.length - 1
			)
			.map((d) => d.date)
	);
</script>

<Chart.Container
	{config}
	class="h-72 w-full"
	role="img"
	aria-label="Тестовые подтверждённые платежи по дням в UTC"
>
	<BarChart
		data={daily}
		x="date"
		y="amountKopecks"
		axis={true}
		padding={{ left: 64, right: 16, top: 12, bottom: 32 }}
		series={[
			{ key: 'amountKopecks', label: config.amountKopecks.label, color: config.amountKopecks.color }
		]}
		props={{
			xAxis: {
				ticks,
				format: (value) => String(value).slice(8, 10) + '.' + String(value).slice(5, 7)
			},
			yAxis: {
				ticks: 3,
				format: (value) =>
					new Intl.NumberFormat('ru-RU', { notation: 'compact', maximumFractionDigits: 0 }).format(
						Number(value) / 100
					) + ' ₽'
			}
		}}
	>
		{#snippet tooltip()}<Chart.Tooltip labelKey="date" labelFormatter={(value) => String(value)}
				>{#snippet formatter({ value })}<span>{rubles(Number(value))}</span
					>{/snippet}</Chart.Tooltip
			>{/snippet}
	</BarChart>
</Chart.Container>
