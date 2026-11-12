<script lang="ts">
	import { tick, untrack } from 'svelte';
	import { api, checkError, errorMessage } from '#lib/api/client.js';
	import type { components } from '#lib/api/schema.js';
	import { rubles, rublesToKopecks } from '#lib/creators/prices.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Textarea } from '#lib/components/ui/textarea/index.js';
	import { Spinner } from '#lib/components/ui/spinner/index.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import * as Field from '#lib/components/ui/field/index.js';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import * as Empty from '#lib/components/ui/empty/index.js';
	import type { PageProps } from './$types';
	type Tier = components['schemas']['Tier'];
	let { data }: PageProps = $props();
	let tiers = $state.raw(untrack(() => data.tiers));
	let active = $derived(tiers.filter((tier) => !tier.archived));
	let archived = $derived(tiers.filter((tier) => tier.archived));
	let editing = $state('');
	let name = $state('');
	let description = $state('');
	let price = $state('');
	let pending = $state(false);
	let failure = $state('');
	let message = $state('');
	let confirm = $state('');
	let nameInput = $state<HTMLElement | null>(null);
	async function edit(tier?: Tier) {
		editing = tier?.id ?? '';
		name = tier?.name ?? '';
		description = tier?.description ?? '';
		price = tier ? (tier.priceKopecks / 100).toFixed(2) : '';
		failure = '';
		confirm = '';
		if (tier) {
			await tick();
			nameInput?.focus({ preventScroll: true });
			nameInput?.scrollIntoView({ block: 'center', behavior: 'instant' });
		}
	}
	async function refresh() {
		const result = await api.GET('/me/tiers');
		checkError(result.error);
		if (result.data) tiers = result.data.tiers;
	}
	async function save(event: SubmitEvent) {
		event.preventDefault();
		if (pending) return;
		failure = '';
		message = '';
		const priceKopecks = rublesToKopecks(price);
		if (priceKopecks === null) {
			failure = 'Укажите цену от 0 до 1 000 000 ₽, не более двух знаков после запятой.';
			return;
		}
		pending = true;
		try {
			const body = { name, description, priceKopecks };
			const result = editing
				? await api.PUT('/me/tiers/{id}', { params: { path: { id: editing } }, body })
				: await api.POST('/me/tiers', { body });
			checkError(result.error);
			await refresh();
			edit();
			message = 'Уровень сохранён.';
		} catch (cause) {
			failure = errorMessage(cause);
		} finally {
			pending = false;
		}
	}
	async function retire(tier: Tier, remove: boolean) {
		if (pending) return;
		pending = true;
		failure = '';
		message = '';
		try {
			const result = remove
				? await api.DELETE('/me/tiers/{id}', { params: { path: { id: tier.id } } })
				: await api.POST('/me/tiers/{id}/archive', { params: { path: { id: tier.id } } });
			checkError(result.error);
			await refresh();
			if (editing === tier.id) edit();
			confirm = '';
			message = 'Уровень убран с публичной страницы.';
		} catch (cause) {
			failure = errorMessage(cause);
		} finally {
			pending = false;
		}
	}
	async function move(index: number, delta: number) {
		if (pending) return;
		const ids = active.map((tier) => tier.id);
		[ids[index], ids[index + delta]] = [ids[index + delta], ids[index]];
		pending = true;
		failure = '';
		message = '';
		try {
			const result = await api.PUT('/me/tiers/order', { body: { ids } });
			checkError(result.error);
			if (result.data) tiers = result.data.tiers;
			message = 'Порядок сохранён.';
		} catch (cause) {
			failure = errorMessage(cause);
		} finally {
			pending = false;
		}
	}
</script>

<svelte:head
	><title>Уровни подписки — etheyl</title><meta name="robots" content="noindex" /></svelte:head
>
<section class="creator-workspace page-inset">
	<a href="/account" class="back-link">← Личный кабинет</a>
	<div class="workspace-heading">
		<div>
			<h1>Уровни подписки</h1>
			<p>Выберите, что будет доступно на каждом уровне.</p>
		</div>
		{#if data.profile}<Button href={`/authors/${data.profile.nickname}`} variant="outline"
				>Открыть страницу</Button
			>{/if}
	</div>
	<nav class="creator-nav" aria-label="Настройки автора">
		<a href="/account/profile">Профиль</a><a href="/account/tiers" aria-current="page"
			>Уровни подписки</a
		>
	</nav>
	{#if failure}<Alert.Root variant="destructive" role="alert"
			><Alert.Title>Не удалось сохранить</Alert.Title><Alert.Description
				>{failure}</Alert.Description
			></Alert.Root
		>{/if}
	{#if message}<p role="status" class="save-message">{message}</p>{/if}
	{#if !data.profile}<Empty.Root
			><Empty.Header
				><Empty.Title>Начните с профиля</Empty.Title><Empty.Description
					>Сохраните имя и адрес автора, чтобы создавать уровни подписки.</Empty.Description
				></Empty.Header
			><Empty.Content><Button href="/account/profile">Оформить профиль</Button></Empty.Content
			></Empty.Root
		>
	{:else}
		<div class="tier-settings-grid">
			<div class="tier-list">
				<p class="form-note">
					Старшие уровни находятся сверху и включают доступ ко всем младшим. Новый уровень всегда
					добавляется сверху. Позиция используемого уровня фиксирована.
				</p>
				{#if !active.length}<Empty.Root
						><Empty.Header
							><Empty.Title>Ваш первый уровень</Empty.Title><Empty.Description
								>Придумайте название, опишите доступ и укажите цену в форме.</Empty.Description
							></Empty.Header
						></Empty.Root
					>{/if}
				{#each active as tier, index (tier.id)}<article class="tier-row">
						<div class="tier-title">
							<h2>{tier.name}</h2>
							<strong>{rubles(tier.priceKopecks)}<small> / месяц</small></strong>
						</div>
						{#if tier.used}<Badge variant="outline">Позиция закреплена</Badge>{/if}
						<p class="tier-description">{tier.description || 'Описание пока не добавлено.'}</p>
						<p class="form-note">
							{index < active.length - 1
								? `Включает все младшие уровни: ${active
										.slice(index + 1)
										.map((item) => item.name)
										.join(', ')}.`
								: 'Базовый уровень доступа.'}
						</p>
						<div class="tier-actions">
							<Button variant="outline" onclick={() => edit(tier)} disabled={pending}
								>Редактировать</Button
							><Button variant="ghost" onclick={() => (confirm = tier.id)} disabled={pending}
								>Убрать уровень</Button
							>
							<Button
								variant="ghost"
								aria-label={`Поднять уровень «${tier.name}»`}
								onclick={() => move(index, -1)}
								disabled={pending || tier.used || index === 0 || active[index - 1]?.used}
								>Выше</Button
							>
							<Button
								variant="ghost"
								aria-label={`Опустить уровень «${tier.name}»`}
								onclick={() => move(index, 1)}
								disabled={pending ||
									tier.used ||
									index === active.length - 1 ||
									active[index + 1]?.used}>Ниже</Button
							>
						</div>
						{#if confirm === tier.id}<div class="retire-confirm">
								<p>
									{tier.used
										? 'Уровень останется в архиве вместе со своей позицией и историей доступа.'
										: 'Можно сохранить уровень в архиве или удалить его безвозвратно.'}
								</p>
								<div class="tier-actions">
									<Button variant="outline" onclick={() => retire(tier, false)} disabled={pending}
										>Архивировать</Button
									>{#if !tier.used}<Button
											variant="destructive"
											onclick={() => retire(tier, true)}
											disabled={pending}>Удалить</Button
										>{/if}<Button variant="ghost" onclick={() => (confirm = '')} disabled={pending}
										>Отмена</Button
									>
								</div>
							</div>{/if}
					</article>{/each}
				{#if archived.length}<section class="tier-archive">
						<h2>Архив</h2>
						<p class="form-note">
							Эти уровни скрыты с публичной страницы. Их история доступа сохранена.
						</p>
						{#each archived as tier (tier.id)}<div class="archive-row">
								<span>{tier.name}</span><span>{rubles(tier.priceKopecks)} / месяц</span>
							</div>{/each}
					</section>{/if}
			</div>
			<form method="POST" onsubmit={save} class="creator-form tier-form">
				<h2>{editing ? 'Редактировать уровень' : 'Новый уровень'}</h2>
				<Field.FieldGroup
					><Field.Field
						><Field.FieldLabel for="tier-name">Название уровня</Field.FieldLabel><Input
							id="tier-name"
							bind:ref={nameInput}
							bind:value={name}
							required
							maxlength={80}
							disabled={pending}
						/></Field.Field
					>
					<Field.Field
						><Field.FieldLabel for="tier-description">Что входит в уровень</Field.FieldLabel
						><Textarea
							id="tier-description"
							bind:value={description}
							maxlength={5000}
							rows={5}
							disabled={pending}
						/></Field.Field
					>
					<Field.Field
						><Field.FieldLabel for="tier-price">Цена, ₽ в месяц</Field.FieldLabel><Input
							id="tier-price"
							bind:value={price}
							required
							inputmode="decimal"
							placeholder="Например, 299,50"
							disabled={pending}
							aria-describedby="price-help"
						/><Field.FieldDescription id="price-help"
							>От 0 до 1 000 000 ₽, с точностью до копейки.</Field.FieldDescription
						></Field.Field
					>
					<Button type="submit" disabled={pending}
						>{#if pending}<Spinner data-icon="inline-start" />{/if}{editing
							? 'Сохранить изменения'
							: 'Создать уровень'}</Button
					>
					{#if editing}<Button variant="ghost" onclick={() => edit()} disabled={pending}
							>Отменить редактирование</Button
						>{/if}</Field.FieldGroup
				>
				<p class="form-note">
					Сейчас можно настроить уровни. Оформление подписок и оплата появятся позже.
				</p>
			</form>
		</div>{/if}
</section>
