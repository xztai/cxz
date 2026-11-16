<script lang="ts">
	import { navigating } from '$app/state';
	import { onMount } from 'svelte';
	import type { Component } from 'svelte';
	import type { components } from '#lib/api/schema.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Table from '#lib/components/ui/table/index.js';
	import * as Field from '#lib/components/ui/field/index.js';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import * as Empty from '#lib/components/ui/empty/index.js';
	import { rubles } from '#lib/creators/prices.js';
	let { report }: { report: components['schemas']['Analytics'] } = $props();
	let Chart = $state<Component<{ daily: components['schemas']['AnalyticsDay'][] }> | null>(null);
	let chartFailed = $state(false);
	async function loadChart() {
		chartFailed = false;
		try {
			Chart = (await import('./PaymentsChart.svelte')).default;
		} catch {
			chartFailed = true;
		}
	}
	onMount(() => {
		void loadChart();
	});
</script>

<div class="flex flex-col gap-8" aria-busy={!!navigating.to}>
	<Alert.Root
		><Alert.Title>Тестовый магазин</Alert.Title><Alert.Description
			>Все денежные показатели относятся к подтверждённым тестовым платежам. Реальных выплат авторам
			нет. Даты и границы периода — UTC.</Alert.Description
		></Alert.Root
	>
	<form method="GET">
		<Field.FieldGroup class="flex flex-wrap items-end gap-4 sm:flex-row"
			><Field.Field class="w-full sm:w-auto"
				><Field.FieldLabel for="analytics-from">С даты · UTC</Field.FieldLabel><Input
					class="h-11"
					id="analytics-from"
					name="from"
					type="date"
					value={report.from}
					min="2000-01-01"
					required
				/></Field.Field
			><Field.Field class="w-full sm:w-auto"
				><Field.FieldLabel for="analytics-to">По дату · UTC, включительно</Field.FieldLabel><Input
					class="h-11"
					id="analytics-to"
					name="to"
					type="date"
					value={report.to}
					min="2000-01-01"
					required
				/></Field.Field
			><Field.Field class="w-full sm:w-auto"
				><Button type="submit" disabled={!!navigating.to}>Показать период</Button></Field.Field
			></Field.FieldGroup
		>
		<p class="mt-3 text-muted-foreground">
			От 1 до 366 дней. Платежи учитываются по дате серверного подтверждения.
		</p>
	</form>
	{#if navigating.to}<div role="status" aria-label="Обновление отчёта">
			<Skeleton class="h-28 w-full" />
		</div>{:else}
		<Card.Root
			><Card.Header
				><Card.Title><h2>Платежи за период</h2></Card.Title><Card.Description
					>{report.from} — {report.to} · UTC</Card.Description
				></Card.Header
			><Card.Content
				><dl class="flex flex-wrap gap-x-12 gap-y-5">
					<div>
						<dt class="text-muted-foreground">Подтверждено платежей</dt>
						<dd class="mt-1 text-xl tabular-nums" data-testid="payment-count">
							{report.paymentCount}
						</dd>
					</div>
					<div>
						<dt class="text-muted-foreground">Сумма тестовых платежей</dt>
						<dd class="mt-1 text-xl tabular-nums" data-testid="payment-amount">
							{rubles(report.amountKopecks)}
						</dd>
					</div>
				</dl></Card.Content
			><Card.Footer
				><p class="text-muted-foreground">
					Ожидающие, отклонённые и неподтверждённые попытки не входят в расчёт.
				</p></Card.Footer
			></Card.Root
		>
		<section aria-labelledby="daily-heading" class="flex flex-col gap-5">
			<h2 id="daily-heading">По дням</h2>
			{#if report.paymentCount === 0}<Empty.Root
					><Empty.Header
						><Empty.Title>В этом периоде платежей нет</Empty.Title><Empty.Description
							>Выберите другие даты или дождитесь первой подтверждённой оплаты. Данные за каждый
							день доступны в таблице.</Empty.Description
						></Empty.Header
					></Empty.Root
				>{:else if Chart}<Chart daily={report.daily} />{:else if chartFailed}<Alert.Root
					><Alert.Title>График не загрузился</Alert.Title><Alert.Description
						>Суммы доступны в таблице по дням.</Alert.Description
					></Alert.Root
				><Button variant="outline" onclick={loadChart}>Загрузить график</Button>{:else}<Skeleton
					class="h-72 w-full"
					aria-label="Загрузка графика"
				/>{/if}
			<details>
				<summary class="cursor-pointer py-3">Данные по дням · доступная таблица</summary><Table.Root
					><Table.Caption>Подтверждённые тестовые платежи, UTC</Table.Caption><Table.Header
						><Table.Row
							><Table.Head>Дата</Table.Head><Table.Head>Платежи</Table.Head><Table.Head
								>Сумма, RUB</Table.Head
							></Table.Row
						></Table.Header
					><Table.Body
						>{#each report.daily as day (day.date)}<Table.Row
								><Table.Cell>{day.date}</Table.Cell><Table.Cell>{day.paymentCount}</Table.Cell
								><Table.Cell class="tabular-nums">{rubles(day.amountKopecks)}</Table.Cell
								></Table.Row
							>{/each}</Table.Body
					></Table.Root
				>
			</details>
		</section>
		<section aria-labelledby="subscriptions-heading" class="flex flex-col gap-5">
			<div>
				<h2 id="subscriptions-heading">Активные подписки сейчас</h2>
				<p class="mt-2 text-muted-foreground">
					{report.activeSubscriptions} · включая отменённое автопродление с ещё действующим доступом.
					Этот срез не зависит от выбранных дат.
				</p>
			</div>
			{#if report.tiers.length}<Table.Root
					><Table.Caption>Распределение оплаченного доступа по уровням</Table.Caption><Table.Header
						><Table.Row
							><Table.Head>Уровень</Table.Head><Table.Head>Активные подписки</Table.Head></Table.Row
						></Table.Header
					><Table.Body
						>{#each report.tiers as tier (tier.id)}<Table.Row
								><Table.Cell class="whitespace-normal"
									>{tier.name}
									{#if tier.archived}<Badge variant="outline">Архив</Badge>{/if}</Table.Cell
								><Table.Cell class="tabular-nums">{tier.activeSubscriptions}</Table.Cell></Table.Row
							>{/each}</Table.Body
					></Table.Root
				>{:else}<Empty.Root
					><Empty.Header
						><Empty.Title>Добавьте уровни подписки</Empty.Title><Empty.Description
							>После оплаты здесь появится распределение подписчиков.</Empty.Description
						></Empty.Header
					><Empty.Content
						><Button href="/account/tiers" variant="outline">Создать уровень</Button></Empty.Content
					></Empty.Root
				>{/if}
		</section>
	{/if}
</div>
