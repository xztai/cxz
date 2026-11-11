<script lang="ts">
	import ArrowUpRight from '@lucide/svelte/icons/arrow-up-right';
	import ArrowRight from '@lucide/svelte/icons/arrow-right';
	import { Button } from '#lib/components/ui/button/index.js';
	import LineRule from '#lib/components/LineRule.svelte';
	import type { PageProps } from './$types';
	let { data }: PageProps = $props();
</script>

<svelte:head><title>etheyl — ближе к творчеству</title></svelte:head>
<section class="home-hero page-inset" aria-labelledby="home-title">
	<h1 id="home-title" class="hero-title">У творчества<br />есть <span>свои люди.</span></h1>
	<div class="hero-intro">
		<div>
			<p class="hero-description">
				Идеи обретают силу, когда ими делятся. Место для ваших историй, вашей аудитории и взаимной
				поддержки.
			</p>
			<p class="hero-audience">Независимым авторам и их людям</p>
		</div>
		<div class="hero-actions">
			{#if data.user}
				<Button href="/account" size="lg"
					>В личный кабинет <ArrowRight data-icon="inline-end" /></Button
				>
			{:else}
				<Button href="/auth/register" size="lg"
					>Присоединиться <ArrowUpRight data-icon="inline-end" /></Button
				>
				<a class="text-link text-sm" href="/auth/login">У меня есть аккаунт</a>
			{/if}
			<p class="hero-note">Один аккаунт — быть автором и поддерживать других.</p>
		</div>
	</div>
</section>
<section class="home-statement page-inset" aria-label="Пространство для своего">
	<LineRule />
	<div class="statement-content">
		<blockquote>Большие истории начинаются с маленького «продолжай».</blockquote>
		<p>Пространство для своего</p>
	</div>
</section>
<section id="about" class="home-roles" aria-labelledby="together">
	<div class="page-inset">
		<div class="roles-heading">
			<h2 id="together">Ближе.<br />И по-настоящему.</h2>
			<p>По обе стороны творчества.<br />Автор и аудитория.</p>
		</div>
		<article class="role-row">
			<div class="role-title">
				<h3>Создавайте своё</h3>
				<p>Для авторов</p>
			</div>
			<p class="role-description">
				Соберите рядом людей, которым откликается то, что вы делаете. Пишите, исследуйте и делитесь
				своим взглядом.
			</p>
			<Button href={data.user ? '/account' : '/auth/register'} variant="link"
				>Найти своих людей <ArrowRight data-icon="inline-end" /></Button
			>
		</article>
		<article class="role-row">
			<div class="role-title">
				<h3>Будьте частью истории</h3>
				<p>Для читателей и слушателей</p>
			</div>
			<p class="role-description">
				Поддерживайте тех, чьего следующего поста ждёте. За каждой любимой историей стоит человек.
			</p>
			<Button href={data.user ? '/account' : '/auth/register'} variant="link"
				>Присоединиться <ArrowRight data-icon="inline-end" /></Button
			>
		</article>
	</div>
</section>
