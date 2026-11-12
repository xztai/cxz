<script lang="ts">
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Empty from '#lib/components/ui/empty/index.js';
	import { rubles } from '#lib/creators/prices.js';
	import type { PageProps } from './$types';
	let { data }: PageProps = $props();
	let owner = $derived(data.user?.id === data.profile.id);
</script>

<svelte:head
	><title>{data.profile.displayName} — etheyl</title><meta
		name="description"
		content={data.profile.description.slice(0, 160) ||
			`Страница автора ${data.profile.displayName} на etheyl.`}
	/></svelte:head
>
<section class="public-author page-inset">
	<a href="/authors" class="back-link">← Каталог авторов</a>
	{#if data.profile.cover}<div class="author-cover">
			<img src={data.profile.cover.url} alt={`Обложка автора ${data.profile.displayName}`} />
		</div>{/if}
	<div class="public-author-heading">
		<div class="public-author-identity">
			{#if data.profile.avatar}<img
					class="author-avatar"
					src={data.profile.avatar.url}
					alt={`Аватар автора ${data.profile.displayName}`}
				/>{:else}<span class="author-avatar avatar-initial" aria-hidden="true"
					>{data.profile.displayName.slice(0, 1).toUpperCase()}</span
				>{/if}
			<div>
				<h1>{data.profile.displayName}</h1>
				<p>@{data.profile.nickname}</p>
			</div>
		</div>
		{#if owner}<Button href="/account/profile" variant="outline">Редактировать профиль</Button>{/if}
	</div>
	<div class="public-author-grid">
		<section class="author-about" aria-labelledby="about-author">
			<h2 id="about-author">Об авторе</h2>
			<p>{data.profile.description || 'Автор ещё не добавил описание.'}</p>
		</section>
		<section class="public-tiers" aria-labelledby="tiers-heading">
			<div class="section-title">
				<h2 id="tiers-heading">Уровни подписки</h2>
				{#if owner}<Button href="/account/tiers" variant="link">Настроить уровни</Button>{/if}
			</div>
			{#if data.tiers.length}<p class="form-note">
					Каждый старший уровень включает доступ ко всем младшим. Оформление подписок и оплата пока
					недоступны.
				</p>
				{#each data.tiers as tier, index (tier.id)}<article class="tier-row">
						<div class="tier-title">
							<h3>{tier.name}</h3>
							<strong>{rubles(tier.priceKopecks)}<small> / месяц</small></strong>
						</div>
						<p class="tier-description">{tier.description || 'Описание пока не добавлено.'}</p>
						{#if index < data.tiers.length - 1}<p class="tier-includes">
								Также включает: {data.tiers
									.slice(index + 1)
									.map((item) => item.name)
									.join(', ')}.
							</p>{/if}
					</article>{/each}
			{:else}<Empty.Root
					><Empty.Header
						><Empty.Title>Уровни ещё не добавлены</Empty.Title><Empty.Description
							>Автор готовит варианты подписки. Загляните сюда позже.</Empty.Description
						></Empty.Header
					>{#if owner}<Empty.Content
							><Button href="/account/tiers">Создать первый уровень</Button></Empty.Content
						>{/if}</Empty.Root
				>{/if}
		</section>
	</div>
</section>
