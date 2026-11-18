<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import Icon from '#lib/components/Icon.svelte';
	import SiteHeader from '#lib/components/SiteHeader.svelte';
	import { getSession } from '#lib/session.svelte.js';
	import type { LayoutProps } from './$types';

	let { data, children }: LayoutProps = $props();
	let menuOpen = $state(false);
	const session = getSession();
	const ownerId = $derived(
		data.access.state === 'ready' && session.isStudioOwner(data.access.ownerId)
			? data.access.ownerId
			: null
	);
	const previewingProfile = $derived(
		ownerId !== null &&
			page.route.id === '/studio/profile' &&
			(page.state.profilePreview || page.url.searchParams.has('preview'))
	);

	const navigation = [
		{ path: '/studio', label: 'Обзор', icon: 'overview' },
		{ path: '/studio/profile', label: 'Профиль', icon: 'profile' },
		{ path: '/studio/tiers', label: 'Уровни подписки', icon: 'layers' },
		{ path: '/studio/posts', label: 'Посты', icon: 'post' },
		{ path: '/studio/analytics', label: 'Аналитика', icon: 'analytics' }
	] as const;
</script>

<svelte:head><meta name="robots" content="noindex" /></svelte:head>

{#snippet links()}
	<nav aria-label="Студия">
		{#each navigation as { path, label, icon } (path)}
			{@const current = page.route.id === path}
			{@const inSection = path !== '/studio' && page.route.id?.startsWith(path + '/')}
			<a
				href={resolve(path)}
				aria-current={current ? 'page' : inSection ? 'location' : undefined}
				onclick={() => (menuOpen = false)}
			>
				<Icon name={icon} size={19} />
				{label}
			</a>
		{/each}
	</nav>
{/snippet}

<SiteHeader />
<div class={previewingProfile ? 'profile-preview' : 'studio'}>
	<div class="shell studio-shell">
		{#if ownerId !== null && !previewingProfile}
			<aside class="sidebar">
				<p>Студия</p>
				{@render links()}
			</aside>
			<details class="mobile-menu" bind:open={menuOpen}>
				<summary>
					<Icon name="menu" size={19} />
					Меню студии
					<span class="menu-chevron"><Icon name="chevron" size={16} /></span>
				</summary>
				{@render links()}
			</details>
		{/if}
		<main id="main" class={{ 'simple-page': previewingProfile }} tabindex="-1">
			{#if ownerId !== null}
				{#key ownerId}
					{@render children()}
				{/key}
			{:else}
				<h1 class="studio-heading">Студия</h1>
				{#if (session.verified && !session.user) || (data.access.state === 'unauthorized' && !session.user)}
					<p class="studio-copy">Войдите в аккаунт, чтобы открыть студию.</p>
					<a class="button button-primary" href={resolve('/login')}>Войти</a>
				{:else if !session.ready || session.studioStatus === 'pending'}
					<p class="studio-copy" role="status">Загружаем…</p>
				{:else}
					<p class="studio-copy" role="status">Не удалось загрузить студию.</p>
					<button
						class="button button-outline"
						type="button"
						onclick={() => session.recheckStudio()}
					>
						Повторить
					</button>
				{/if}
			{/if}
		</main>
	</div>
</div>

<style>
	.studio {
		background: var(--surface);
		min-height: calc(100vh - 88px);
	}
	.studio .studio-shell {
		display: grid;
		grid-template-columns: 220px minmax(0, 1fr);
		column-gap: 40px;
		min-height: inherit;
	}
	.sidebar {
		border-inline-end: 1px solid var(--line);
		padding: 40px 20px 40px 0;
	}
	.sidebar p {
		padding-inline: 12px;
		margin-bottom: 16px;
		font-size: 18px;
		font-weight: 800;
		letter-spacing: -0.02em;
	}
	nav {
		display: grid;
		gap: 5px;
	}
	nav a,
	summary {
		display: flex;
		align-items: center;
		min-height: 44px;
		gap: 11px;
		padding: 9px 12px;
		border-radius: 8px;
		color: var(--muted);
		font-size: 14px;
		font-weight: 750;
		text-decoration: none;
	}
	nav a:hover,
	nav a[aria-current] {
		color: var(--blue-hover);
		background: var(--blue-soft);
	}
	nav a[aria-current] {
		font-weight: 800;
	}
	.mobile-menu {
		display: none;
	}
	.studio main {
		min-width: 0;
		padding-block: 36px 64px;
	}
	main:only-child {
		grid-column: 1 / -1;
	}
	main:focus {
		outline: none;
	}
	.studio :global(.studio-heading) {
		margin: 0;
		font-size: 34px;
		font-weight: 800;
		line-height: 1.2;
		letter-spacing: -0.035em;
		text-wrap: balance;
	}
	.studio :global(.studio-panel) {
		border: 1px solid var(--line);
		border-radius: 16px;
		background: var(--white);
		padding: 24px;
	}
	.studio :global(.studio-heading + .studio-panel) {
		margin-block-start: 24px;
	}
	.studio :global(.studio-copy) {
		margin-block: 12px 20px;
		color: var(--muted);
		font-size: 14px;
		line-height: 1.65;
	}
	.studio :global(.studio-link) {
		color: var(--blue-hover);
		font-weight: 750;
		text-underline-offset: 4px;
	}
	.studio :global(.studio-link:hover) {
		text-decoration: underline;
	}
	.studio :global(.studio-form) {
		max-width: 640px;
		margin-top: 24px;
	}
	.studio :global(.studio-form textarea),
	.studio :global(.studio-form select) {
		width: 100%;
		min-height: 50px;
		border: 1px solid #aeb6ca;
		border-radius: 9px;
		background: var(--white);
		padding: 12px 14px;
		color: var(--ink);
		font-size: 15px;
		line-height: 24px;
	}
	.studio :global(.studio-form textarea) {
		resize: vertical;
	}
	@media (max-width: 1000px) {
		.studio .studio-shell {
			grid-template-columns: 202px minmax(0, 1fr);
			column-gap: 28px;
		}
		.sidebar {
			padding-inline-end: 16px;
		}
	}
	@media (max-width: 800px) {
		.studio .studio-shell {
			display: block;
			padding-block-start: 20px;
		}
		.sidebar {
			display: none;
		}
		.mobile-menu {
			display: block;
			border: 1px solid var(--line);
			border-radius: 10px;
			background: var(--white);
		}
		.mobile-menu nav {
			padding: 0 7px 7px;
		}
		summary {
			list-style: none;
			cursor: pointer;
			color: var(--ink);
			padding-inline: 13px;
		}
		summary::-webkit-details-marker {
			display: none;
		}
		.menu-chevron {
			margin-inline-start: auto;
		}
		details[open] .menu-chevron {
			transform: rotate(180deg);
		}
		.studio main {
			padding-block: 24px 48px;
		}
	}
	@media (max-width: 640px) {
		.studio {
			min-height: calc(100vh - 74px);
		}
		.studio :global(.studio-heading) {
			font-size: 30px;
		}
		.studio :global(.studio-panel) {
			padding: 18px;
		}
	}
</style>
