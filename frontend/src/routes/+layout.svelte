<script lang="ts">
	import './layout.css';
	import { page } from '$app/state';
	import Logo from '#lib/components/Logo.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import type { LayoutProps } from './$types';
	let { children, data }: LayoutProps = $props();
	let isHome = $derived(page.url.pathname === '/');
	let isAuth = $derived(page.url.pathname.startsWith('/auth/'));
</script>

<svelte:head
	><link rel="icon" href="/favicon.svg" type="image/svg+xml" /><meta
		name="theme-color"
		content={isHome ? '#2448e5' : '#f8f9f5'}
	/></svelte:head
>
<a href="#main" class="skip-link">Перейти к содержимому</a>
<div class={['site-frame', { 'home-frame': isHome, 'auth-frame': isAuth }]}>
	<header class="site-header">
		<div class="header-content page-inset">
			<Logo />
			<nav aria-label="Главная навигация" class="header-nav">
				{#if !isAuth}<a href="/authors">Авторы</a><a href="/#about" class="nav-about">Для кого</a><a
						href="/#join"
						class="nav-about">Как присоединиться</a
					>{/if}
				{#if data.user}<Button href="/account" variant={isHome ? 'secondary' : 'default'}
						>Личный кабинет</Button
					>
				{:else if isAuth}<span class="auth-header-note">Место для вашего продолжения.</span>
				{:else}<a href="/auth/login" class="nav-login">Войти</a><Button
						href="/auth/register"
						variant="secondary">Создать аккаунт</Button
					>{/if}
			</nav>
		</div>
	</header>
	<main id="main" tabindex="-1">{@render children()}</main>
	<footer class="site-footer page-inset">
		<Logo /><span>Создавать. Поддерживать. Быть рядом.</span><span
			>© {new Date().getFullYear()} etheyl</span
		>
	</footer>
</div>
