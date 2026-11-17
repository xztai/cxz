<script lang="ts">
	import { invalidate } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { api, errorMessage } from '#lib/api/client.js';
	import { formatRubles } from '#lib/studio/format.js';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	let saving = $state(false);
	let retrying = $state(false);
	let message = $state('');
	let saved = $state(false);

	async function retry() {
		retrying = true;
		try {
			await invalidate('studio:tiers');
		} finally {
			retrying = false;
		}
	}

	async function createTier(event: SubmitEvent) {
		event.preventDefault();
		if (saving) return;
		const form = event.currentTarget as HTMLFormElement;
		const fields = new FormData(form);
		const priceKopecks = Math.round(Number(String(fields.get('price')).replace(',', '.')) * 100);
		saved = false;
		if (!Number.isSafeInteger(priceKopecks) || priceKopecks < 0 || priceKopecks > 100000000) {
			message = 'Укажите цену от 0 до 1 000 000 ₽.';
			return;
		}
		saving = true;
		message = '';
		try {
			await api.createTier({
				name: String(fields.get('name')),
				description: String(fields.get('description')),
				priceKopecks
			});
			form.reset();
			saved = true;
			await Promise.all([invalidate('studio:tiers'), invalidate('studio:overview')]);
		} catch (error) {
			message = errorMessage(error);
		} finally {
			saving = false;
		}
	}
</script>

<svelte:head><title>Уровни подписки — etheyl</title></svelte:head>

<h1 class="studio-heading">Уровни подписки</h1>
{#if data.access.state === 'ready' && data.access.profile}
	{#if !data.tiers}
		<p class="feedback error-feedback" role="alert">Не удалось загрузить уровни подписки.</p>
		<button type="button" class="button button-outline" disabled={retrying} onclick={retry}>
			{retrying ? 'Загружаем…' : 'Повторить'}
		</button>
	{:else if data.tiers.length}
		<ul class="studio-panel tiers">
			{#each data.tiers as tier (tier.id)}
				<li>
					<div>
						<h2>{tier.name}</h2>
						{#if tier.description}<p class="studio-copy">{tier.description}</p>{/if}
					</div>
					<span>
						{tier.archived ? 'В архиве' : `${formatRubles(tier.priceKopecks)} / мес.`}
					</span>
				</li>
			{/each}
		</ul>
	{/if}
	<form class="studio-form studio-panel" aria-busy={saving} onsubmit={createTier}>
		<h2>Новый уровень</h2>
		<fieldset disabled={saving}>
			<div class="field">
				<label for="tier-name">Название</label>
				<input id="tier-name" name="name" required />
			</div>
			<div class="field">
				<label for="tier-description">Описание</label>
				<textarea id="tier-description" name="description" rows="3"></textarea>
			</div>
			<div class="field">
				<label for="tier-price">Стоимость в месяц, ₽</label>
				<input
					id="tier-price"
					name="price"
					inputmode="decimal"
					pattern={'[0-9]+([,.][0-9]{1,2})?'}
					title="Например, 150 или 150,50"
					required
				/>
			</div>
			<button type="submit" class="button button-primary">
				{saving ? 'Создаём…' : 'Создать уровень'}
			</button>
		</fieldset>
		{#if message}<p class="feedback error-feedback" role="alert">{message}</p>{/if}
		<p class="feedback" role="status">{saved ? 'Уровень создан.' : ''}</p>
	</form>
{:else}
	<div class="studio-panel">
		<p class="studio-copy">Уровни подписки доступны после сохранения профиля автора.</p>
		<a class="button button-primary" href={resolve('/studio/profile')}>Создать профиль</a>
	</div>
{/if}

<style>
	h2 {
		font-size: 19px;
		font-weight: 800;
		margin-block-end: 16px;
	}

	.tiers.studio-panel {
		list-style: none;
		padding-block: 4px;
		margin-block: 24px;
	}

	.tiers li {
		display: flex;
		flex-wrap: wrap;
		justify-content: space-between;
		align-items: baseline;
		gap: 8px 24px;
		padding-block: 16px;
	}

	.tiers li + li {
		border-block-start: 1px solid var(--line);
	}

	.tiers h2,
	.tiers p {
		margin: 0;
		overflow-wrap: anywhere;
	}

	.tiers li > span {
		font-size: 14px;
		font-weight: 700;
		font-variant-numeric: tabular-nums;
	}
</style>
