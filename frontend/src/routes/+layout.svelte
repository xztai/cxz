<script lang="ts">
	import './layout.css';
	import Logo from '#lib/components/Logo.svelte';
	import LineRule from '#lib/components/LineRule.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import type { LayoutProps } from './$types';
	let { children, data }: LayoutProps = $props();
</script>

<svelte:head>
	<link rel="icon" href="/favicon.svg" type="image/svg+xml" />
	<meta name="theme-color" content="#282031" />
	<meta
		name="description"
		content="etheyl — пространство для авторов и тех, кто ценит их творчество."
	/>
</svelte:head>

<a href="#main" class="skip-link">Перейти к содержимому</a>
<div class="site-frame">
	<header class="site-header">
		<div class="header-content">
			<Logo />
			<nav aria-label="Главная навигация" class="header-nav">
				<a href="/#about" class="about-link">О пространстве</a>
				{#if data.user}<Button href="/account" variant="outline">Личный кабинет</Button>
				{:else}<Button href="/auth/login" variant="ghost">Войти</Button><Button
						href="/auth/register">Начать</Button
					>{/if}
			</nav>
		</div>
	</header>
	<main id="main" class="flex flex-1 flex-col" tabindex="-1">{@render children()}</main>
	<footer class="site-footer">
		<LineRule />
		<div class="footer-content">
			<span>© {new Date().getFullYear()} etheyl</span><span>Творчество начинается с доверия.</span>
		</div>
	</footer>
</div>
