<script lang="ts">
	import { invalidate } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { onDestroy, tick } from 'svelte';
	import {
		api,
		ApiError,
		errorMessage,
		type Tier,
		type TierInput,
		type TierOrderConflictCode
	} from '#lib/api/client.js';
	import Icon from '#lib/components/Icon.svelte';
	import { formatRubles } from '#lib/studio/format.js';
	import TierDialog, { type TierDialogAction } from './TierDialog.svelte';
	import type { PageProps } from './$types';

	const limit = 50;
	const minimumPaymentKopecks = 100;
	const lockedOrderCode = 'tier_order_locked' satisfies TierOrderConflictCode;
	const orderConflictMessage = 'Порядок изменился, обновляем список';
	const orderDraftMessage = 'Порядок изменён. Сохраните его, чтобы применить.';
	const directions = [
		{ offset: -1, label: 'Выше' },
		{ offset: 1, label: 'Ниже' }
	];

	let { data }: PageProps = $props();
	let tiers = $derived(data.tiers);
	const ordered = $derived((tiers ?? []).toSorted((a, b) => b.position - a.position));
	const savedActive = $derived(ordered.filter((tier) => !tier.archived));
	const archived = $derived(ordered.filter((tier) => tier.archived));
	const orderLayout = $derived(
		JSON.stringify(
			ordered.map(({ id, position, used, archived }) => [id, position, used, archived])
		)
	);
	let draft = $state.raw<{ layout: string; ids: string[] }>();
	const active = $derived.by(() => {
		const current = draft;
		if (!current || current.layout !== orderLayout) return savedActive;
		return savedActive.toSorted((a, b) => current.ids.indexOf(a.id) - current.ids.indexOf(b.id));
	});
	const orderChanged = $derived(active.some((tier, index) => tier.id !== savedActive[index].id));
	const atLimit = $derived(savedActive.length >= limit);
	const movementSections = $derived.by(() => {
		const sections: Record<string, number | undefined> = Object.create(null);
		let section = 0;
		for (const tier of ordered) {
			if (tier.used || tier.archived) section++;
			else sections[tier.id] = section;
		}
		return sections;
	});

	let heading: HTMLHeadingElement;
	let newTierButton: HTMLButtonElement | undefined;
	let dialogTrigger: HTMLButtonElement | undefined;
	let dialog = $state.raw<TierDialogAction>();
	let activity = $state<'reload' | 'reorder'>();
	let status = $state('');
	let problem = $state('');
	let toastTimer: ReturnType<typeof setTimeout> | undefined;
	let draggedId = $state<string>();
	let targetId = $state<string>();
	const draggedIndex = $derived(active.findIndex((tier) => tier.id === draggedId));

	onDestroy(() => clearTimeout(toastTimer));

	function canMove(fromId: string, toId: string | undefined) {
		const section = movementSections[fromId];
		return (
			toId !== undefined &&
			fromId !== toId &&
			section !== undefined &&
			section === movementSections[toId]
		);
	}

	function restoreFocus(trigger?: HTMLButtonElement) {
		if (trigger?.isConnected && !trigger.disabled) trigger.focus();
		else if (newTierButton?.isConnected && !newTierButton.disabled) newTierButton.focus();
		else heading.focus();
	}

	async function retry() {
		activity = 'reload';
		try {
			await invalidate('studio:tiers');
		} finally {
			activity = undefined;
		}
		if (tiers) heading.focus();
	}

	function openDialog(action: TierDialogAction, trigger: HTMLButtonElement) {
		if (activity || dialog || (action.type === 'create' && (!tiers || atLimit))) return;
		status = '';
		problem = '';
		dialogTrigger = trigger;
		dialog = action;
	}

	async function closeDialog() {
		const trigger = dialogTrigger;
		dialog = undefined;
		dialogTrigger = undefined;
		await tick();
		restoreFocus(trigger);
	}

	async function refreshLists() {
		const previousDraft = orderChanged ? draft : undefined;
		await Promise.all([invalidate('studio:tiers'), invalidate('studio:overview')]);
		if (previousDraft && previousDraft.layout !== orderLayout) {
			draft = undefined;
			return ' Несохранённый порядок сброшен, потому что список уровней изменился.';
		}
		return '';
	}

	async function saveTier(input: TierInput, tier?: Tier) {
		try {
			if (tier) await api.updateTier(tier.id, input);
			else await api.createTier(input);
		} catch (error) {
			if (error instanceof ApiError && (error.code === 'tier_limit' || error.status === 404)) {
				status = (await refreshLists()).trimStart();
			}
			throw error;
		}
		const orderNotice = await refreshLists();
		status = (tier ? 'Изменения сохранены.' : 'Уровень создан.') + orderNotice;
	}

	async function retireTier(tier: Tier, action: 'delete' | 'archive') {
		if (action === 'archive') await api.archiveTier(tier.id);
		else await api.deleteTier(tier.id);
		const orderNotice = await refreshLists();
		const movedToArchive =
			action === 'archive' || tiers?.some((current) => current.id === tier.id && current.archived);
		status =
			(movedToArchive ? 'Уровень перемещён в архив.' : 'Уровень убран с публичной страницы.') +
			orderNotice;
	}

	async function moveTier(id: string, destination: number, trigger?: HTMLButtonElement) {
		if (activity || dialog || !canMove(id, active[destination]?.id)) return;
		const document = trigger?.ownerDocument;
		const hadFocus = trigger && document?.activeElement === trigger;
		const ids = active.map((tier) => tier.id);
		ids.splice(ids.indexOf(id), 1);
		ids.splice(destination, 0, id);
		draft = ids.some((id, index) => id !== savedActive[index].id)
			? { layout: orderLayout, ids }
			: undefined;
		status = orderChanged ? orderDraftMessage : '';
		problem = '';
		await tick();
		if (hadFocus && document?.activeElement === document?.body) restoreFocus(trigger);
	}

	async function saveOrder() {
		if (!orderChanged || !draft || activity || dialog) return;
		clearTimeout(toastTimer);
		activity = 'reorder';
		problem = '';
		status = '';
		try {
			const result = await api.reorderTiers({ ids: draft.ids });
			tiers = result.tiers;
			draft = undefined;
			status = 'Порядок уровней сохранён.';
		} catch (error) {
			if (error instanceof ApiError && error.status === 409 && error.code === lockedOrderCode) {
				status = orderConflictMessage;
				await invalidate('studio:tiers');
				draft = undefined;
				toastTimer = setTimeout(() => {
					if (status === orderConflictMessage) status = '';
				}, 5000);
			} else if (error instanceof ApiError && error.code === 'validation') {
				problem = 'Список уровней изменился. Проверьте его и повторите попытку.';
				await invalidate('studio:tiers');
				draft = undefined;
			} else {
				problem = `Не удалось изменить порядок. ${errorMessage(error)}`;
			}
		} finally {
			activity = undefined;
		}
	}

	function startDrag(event: DragEvent, id: string) {
		if (activity || dialog || !event.dataTransfer) {
			event.preventDefault();
			return;
		}
		draggedId = id;
		event.dataTransfer.effectAllowed = 'move';
		event.dataTransfer.setData('text/plain', id);
	}

	function hoverRow(event: DragEvent, id: string) {
		if (!draggedId || !canMove(draggedId, id)) {
			targetId = undefined;
			return;
		}
		event.preventDefault();
		if (event.dataTransfer) event.dataTransfer.dropEffect = 'move';
		targetId = id;
	}

	function leaveRow(event: DragEvent, id: string) {
		const row = event.currentTarget as HTMLElement;
		if (targetId === id && !row.contains(event.relatedTarget as Node | null)) targetId = undefined;
	}

	function clearDrag() {
		draggedId = undefined;
		targetId = undefined;
	}

	function dropTier(event: DragEvent, index: number) {
		if (!draggedId || targetId !== active[index]?.id) return;
		event.preventDefault();
		const id = draggedId;
		clearDrag();
		void moveTier(id, index);
	}
</script>

<svelte:head><title>Уровни подписки — etheyl</title></svelte:head>

{#snippet tierInfo(tier: Tier, invertedPrice = false)}
	{@const belowMinimum = !tier.archived && tier.priceKopecks < minimumPaymentKopecks}
	<div class="tier-title">
		<h2>{tier.name}</h2>
		{#if tier.archived}<span class="badge">Архив</span>{/if}
		{#if tier.used}
			<span class="badge badge-used">Используется</span>
			{#if !tier.archived}<span class="locked-note">Позиция закреплена</span>{/if}
		{/if}
	</div>
	<p class="tier-price">{formatRubles(tier.priceKopecks)} <span>/ мес.</span></p>
	{#if tier.description || invertedPrice || belowMinimum}
		<div class="tier-details">
			{#if tier.description}<p class="tier-description">{tier.description}</p>{/if}
			{#if invertedPrice}
				<p class="price-warning">
					<Icon name="warning" size={16} />
					Дороже по порядку, но дешевле по цене
				</p>
			{/if}
			{#if belowMinimum}
				<p class="price-warning">
					<Icon name="warning" size={16} />
					Этот уровень нельзя оформить как подписку, минимальный платёж 1 ₽
				</p>
			{/if}
		</div>
	{/if}
{/snippet}

<div class="tiers-page">
	<header class="tiers-header">
		<div class="heading-group">
			<h1
				class="studio-heading"
				{@attach (element) => {
					heading = element;
				}}
				tabindex="-1"
			>
				Уровни подписки
			</h1>
			{#if tiers}<p class="active-count">{active.length} из {limit} активных</p>{/if}
		</div>
		{#if data.access.state === 'ready' && data.access.profile}
			<div class="create-action">
				<button
					{@attach (element) => {
						newTierButton = element;
					}}
					type="button"
					class="button button-primary"
					disabled={!tiers || atLimit}
					aria-disabled={activity !== undefined}
					aria-describedby={atLimit ? 'tier-limit' : undefined}
					onclick={(event) => openDialog({ type: 'create' }, event.currentTarget)}
				>
					<Icon name="plus" size={18} />
					Новый уровень
				</button>
				{#if atLimit}<p id="tier-limit">Достигнут лимит активных уровней.</p>{/if}
			</div>
		{/if}
	</header>

	{#if data.access.state === 'ready' && data.access.profile}
		{#if tiers}
			<div class="order-toolbar">
				<p>
					Порядок определяет доступ к постам, а не цена. Создавайте уровни от низшего к высшему:
					новый уровень всегда добавляется выше всех
				</p>
				{#if active.length}
					<button
						class="button button-outline save-order"
						type="button"
						aria-disabled={!orderChanged || activity !== undefined}
						onclick={saveOrder}
					>
						{activity === 'reorder' ? 'Сохраняем порядок…' : 'Сохранить порядок'}
					</button>
				{/if}
			</div>
		{/if}

		<p class={['notice', { 'sr-only': status === orderDraftMessage }]} role="status">
			{status === orderConflictMessage ? '' : status}
		</p>
		{#if problem}<p class="notice error-feedback" role="alert">{problem}</p>{/if}

		{#if !tiers}
			<div class="studio-panel load-error">
				<p role="alert">Не удалось загрузить уровни подписки.</p>
				<button
					class="button button-outline"
					type="button"
					disabled={activity === 'reload'}
					onclick={retry}
				>
					{activity === 'reload' ? 'Загружаем…' : 'Повторить'}
				</button>
			</div>
		{:else}
			{#if active.length}
				<ol
					class="studio-panel tier-list"
					role="list"
					aria-label="Активные уровни"
					aria-busy={activity === 'reorder'}
				>
					{#each active as tier, index (tier.id)}
						{@const movable = directions.some(({ offset }) =>
							canMove(tier.id, active[index + offset]?.id)
						)}
						{@const invertedPrice = active
							.slice(index + 1)
							.some((lower) => lower.priceKopecks > tier.priceKopecks)}
						<li
							class={[
								'tier-row',
								{
									dragging: draggedId === tier.id,
									'drop-above': targetId === tier.id && index < draggedIndex,
									'drop-below': targetId === tier.id && index > draggedIndex
								}
							]}
							ondragover={(event) => hoverRow(event, tier.id)}
							ondragleave={(event) => leaveRow(event, tier.id)}
							ondrop={(event) => dropTier(event, index)}
						>
							{#if tier.used}
								<span class="row-symbol"><Icon name="lock" size={17} /></span>
							{:else}
								<span
									class="row-symbol grip"
									aria-hidden="true"
									draggable={movable && !activity}
									ondragstart={(event) => startDrag(event, tier.id)}
									ondragend={clearDrag}><Icon name="grip" size={22} /></span
								>
							{/if}
							{@render tierInfo(tier, invertedPrice)}
							<div class="row-actions" role="group" aria-label={`Действия уровня «${tier.name}»`}>
								{#if !tier.used}
									<div class="move-actions">
										{#each directions as { offset, label } (offset)}
											{@const available = canMove(tier.id, active[index + offset]?.id)}
											<button
												class="row-button"
												type="button"
												aria-disabled={!available || activity !== undefined}
												tabindex={available ? undefined : -1}
												onclick={(event) => moveTier(tier.id, index + offset, event.currentTarget)}
												>{label}</button
											>
										{/each}
									</div>
								{/if}
								<div class="manage-actions">
									<button
										class="row-button"
										type="button"
										aria-disabled={activity !== undefined}
										onclick={(event) => openDialog({ type: 'edit', tier }, event.currentTarget)}
										>Изменить</button
									>
									{#if tier.used}
										<button
											class="row-button"
											type="button"
											aria-disabled={activity !== undefined}
											onclick={(event) =>
												openDialog({ type: 'archive', tier }, event.currentTarget)}
											>Архивировать</button
										>
									{:else}
										<button
											class="row-button danger"
											type="button"
											aria-disabled={activity !== undefined}
											onclick={(event) => openDialog({ type: 'delete', tier }, event.currentTarget)}
											>Удалить</button
										>
									{/if}
								</div>
							</div>
						</li>
					{/each}
				</ol>
			{:else}
				<div class="studio-panel empty">Создайте первый уровень подписки.</div>
			{/if}

			<details class="studio-panel archive">
				<summary>
					<span>Архив</span>
					<span class="archive-count">{archived.length}</span>
					<span class="archive-chevron"><Icon name="chevron" size={18} /></span>
				</summary>
				{#if archived.length}
					<ul class="archive-list" role="list" aria-label="Архивные уровни">
						{#each archived as tier (tier.id)}
							<li class="archive-row">{@render tierInfo(tier)}</li>
						{/each}
					</ul>
				{:else}
					<p class="archive-empty">В архиве пока нет уровней.</p>
				{/if}
			</details>
		{/if}
	{:else}
		<div class="studio-panel profile-required">
			<p class="studio-copy">Уровни подписки доступны после сохранения профиля автора.</p>
			<a class="button button-primary" href={resolve('/studio/profile')}>Создать профиль</a>
		</div>
	{/if}
</div>

<p class="order-toast" role="status">{status === orderConflictMessage ? status : ''}</p>

{#if dialog}
	<TierDialog action={dialog} onsave={saveTier} onretire={retireTier} onclose={closeDialog} />
{/if}

<style>
	.tiers-page {
		container-type: inline-size;
	}

	.tiers-header {
		display: flex;
		flex-wrap: wrap;
		justify-content: space-between;
		align-items: center;
		gap: 12px 20px;
		margin-bottom: 16px;
	}

	.create-action {
		display: grid;
		justify-items: start;
		gap: 6px;
	}

	.create-action p,
	.order-toolbar p,
	.empty,
	.archive-empty,
	.load-error p {
		color: var(--muted);
		font-size: 14px;
		line-height: 1.55;
	}

	.create-action p {
		margin: 0;
		font-size: 12px;
	}

	.create-action button:disabled {
		background: var(--blue);
		cursor: default;
	}

	.create-action button[aria-disabled='true']:not(:disabled) {
		opacity: 0.65;
		cursor: wait;
	}

	.heading-group {
		display: grid;
		gap: 6px;
	}

	.order-toolbar {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		justify-content: space-between;
		gap: 12px 24px;
		margin-bottom: 16px;
	}

	.order-toolbar p,
	.active-count,
	.notice {
		margin: 0;
	}

	.order-toolbar p {
		flex: 1 1 340px;
		max-width: 69ch;
	}

	.save-order {
		min-height: 44px;
		padding: 9px 16px;
		font-size: 14px;
	}

	.save-order[aria-disabled='true'] {
		border-color: var(--line);
		background: var(--white);
		color: var(--muted);
		cursor: default;
	}

	.save-order:not([aria-disabled='true']) {
		border-color: var(--blue);
		color: var(--blue-hover);
	}

	.active-count {
		color: var(--ink);
		font-size: 14px;
		font-weight: 800;
		font-variant-numeric: tabular-nums;
	}

	.notice {
		font-size: 14px;
		line-height: 1.6;
	}

	.notice:not(:empty) {
		margin-bottom: 12px;
	}

	.order-toast {
		position: fixed;
		z-index: 5;
		inset-inline-end: 16px;
		inset-block-end: max(16px, env(safe-area-inset-bottom));
		max-width: calc(100% - 32px);
		margin: 0;
		border-radius: 10px;
		background: var(--ink);
		color: var(--white);
		font-size: 14px;
		font-weight: 700;
		line-height: 1.5;
	}

	.order-toast:not(:empty) {
		padding: 13px 16px;
	}

	.load-error p {
		margin-bottom: 12px;
	}

	.tiers-page .studio-panel.tier-list,
	.tiers-page .studio-panel.archive,
	.archive-list {
		margin: 0;
		padding: 0;
		list-style: none;
	}

	.tier-row {
		position: relative;
		display: grid;
		grid-template-columns: 32px minmax(0, 1fr) max-content 276px;
		align-items: start;
		column-gap: 12px;
		padding: 15px 18px;
	}

	.tier-row + .tier-row,
	.archive-row,
	.archive-empty {
		border-block-start: 1px solid var(--line);
	}

	.tier-row.dragging {
		background: var(--surface);
	}

	.drop-above::before,
	.drop-below::before {
		content: '';
		position: absolute;
		z-index: 1;
		inset-inline: 12px;
		height: 2px;
		background: var(--blue);
		pointer-events: none;
	}

	.drop-above::before {
		inset-block-start: 0;
	}

	.drop-below::before {
		inset-block-end: 0;
	}

	.row-symbol {
		display: grid;
		width: 32px;
		height: 44px;
		place-items: center;
		border-radius: 8px;
		color: var(--muted);
	}

	.grip[draggable='true'] {
		cursor: grab;
	}

	.grip[draggable='true']:hover {
		background: var(--blue-soft);
		color: var(--blue-hover);
	}

	.grip[draggable='false'] {
		opacity: 0.55;
	}

	.tier-title {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 4px 8px;
		min-width: 0;
		padding-block: 9px;
		overflow-wrap: anywhere;
	}

	.tier-title h2 {
		font-size: 16px;
		font-weight: 800;
		line-height: 1.5;
	}

	.tier-title h2,
	.tier-description,
	.tier-price,
	.price-warning {
		margin: 0;
	}

	.badge {
		border-radius: 6px;
		background: var(--surface);
		padding: 3px 7px;
		color: var(--muted);
		font-size: 12px;
		font-weight: 750;
		line-height: 1.4;
	}

	.badge-used {
		background: var(--blue-soft);
		color: var(--blue-hover);
	}

	.locked-note {
		color: var(--muted);
		font-size: 12px;
	}

	.tier-details {
		display: grid;
		grid-column: 2 / -1;
		grid-row: 2;
		gap: 5px;
		overflow-wrap: anywhere;
	}

	.tier-description {
		max-width: 65ch;
		color: var(--muted);
		font-size: 14px;
		line-height: 1.55;
		white-space: pre-wrap;
	}

	.price-warning {
		display: flex;
		align-items: start;
		gap: 6px;
		color: #82510e;
		font-size: 13px;
		font-weight: 700;
		line-height: 1.55;
	}

	.price-warning :global(svg) {
		flex: none;
		margin-top: 2px;
	}

	.tier-price {
		padding-block: 11px;
		text-align: right;
		font-size: 15px;
		font-weight: 800;
		line-height: 22px;
		font-variant-numeric: tabular-nums;
		white-space: nowrap;
	}

	.tier-price span {
		color: var(--muted);
		font-size: 13px;
		font-weight: 600;
	}

	.row-actions,
	.move-actions,
	.manage-actions {
		display: flex;
		flex-wrap: wrap;
		gap: 2px;
	}

	.row-actions {
		grid-column: 4;
		grid-row: 1;
		justify-content: flex-end;
		gap: 8px;
	}

	.row-button {
		min-height: 44px;
		border-radius: 8px;
		padding: 0 7px;
		color: var(--blue-hover);
		font-size: 13px;
		font-weight: 750;
		cursor: pointer;
	}

	.row-button.danger {
		color: var(--error);
	}

	.row-button:not([aria-disabled='true']):hover {
		background: var(--blue-soft);
	}

	.row-button.danger:not([aria-disabled='true']):hover {
		background: #fff1f3;
	}

	.row-button[aria-disabled='true'] {
		color: var(--muted);
		opacity: 0.55;
		cursor: default;
	}

	.tiers-page .studio-panel.archive {
		margin-top: 16px;
	}

	summary {
		display: flex;
		align-items: center;
		gap: 9px;
		min-height: 56px;
		border-radius: 15px;
		padding: 12px 18px;
		list-style: none;
		font-size: 16px;
		font-weight: 800;
		cursor: pointer;
	}

	summary::-webkit-details-marker {
		display: none;
	}

	summary:hover {
		color: var(--blue-hover);
	}

	.archive-count {
		min-width: 24px;
		border-radius: 6px;
		background: var(--surface);
		padding: 2px 5px;
		color: var(--muted);
		text-align: center;
		font-size: 12px;
		font-variant-numeric: tabular-nums;
	}

	.archive-chevron {
		margin-inline-start: auto;
		color: var(--muted);
	}

	details[open] .archive-chevron {
		transform: rotate(180deg);
	}

	.archive-row {
		display: grid;
		grid-template-columns: minmax(0, 1fr) max-content;
		column-gap: 12px;
		background: var(--surface);
		padding: 8px 18px 12px;
	}

	.archive-row:last-child {
		border-radius: 0 0 15px 15px;
	}

	.archive-row .tier-details {
		grid-column: 1 / -1;
	}

	.archive-row .badge:not(.badge-used) {
		background: var(--white);
	}

	.archive-empty {
		margin: 0;
		padding: 12px 18px 16px;
	}

	.profile-required .studio-copy {
		margin-top: 0;
	}

	@container (max-width: 710px) {
		.tier-row {
			grid-template-columns: 32px minmax(0, 1fr);
			gap: 2px 8px;
			padding: 12px;
		}

		.row-symbol {
			height: 34px;
		}

		.tier-row .tier-title {
			padding-block: 5px 0;
		}

		.tier-row :is(.tier-price, .tier-details, .row-actions) {
			grid-column: 2;
			grid-row: auto;
		}

		.tier-price {
			padding-block: 2px;
			text-align: start;
		}

		.row-actions {
			justify-content: flex-start;
			margin-inline-start: -7px;
			row-gap: 0;
		}

		.archive-row {
			grid-template-columns: minmax(0, 1fr);
			gap: 2px;
			padding-block: 12px;
		}

		.archive-row .tier-title {
			padding-block: 0;
		}

		.archive-row .tier-details {
			grid-row: auto;
		}
	}
</style>
