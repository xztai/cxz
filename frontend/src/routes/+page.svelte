<script lang="ts">
	import ArrowRight from '@lucide/svelte/icons/arrow-right';
	import ArrowDown from '@lucide/svelte/icons/arrow-down';
	import Plus from '@lucide/svelte/icons/plus';
	import { Button } from '#lib/components/ui/button/index.js';
	import CreatorCollage from '#lib/components/CreatorCollage.svelte';
	import type { PageProps } from './$types';
	let { data }: PageProps = $props();
	let destination = $derived(data.user ? '/account' : '/auth/register');
	const questions = [
		{
			question: 'Что уже можно делать на etheyl?',
			answer:
				'Создать аккаунт, подтвердить почту и войти в личный кабинет. Публикации, подписки и платежи мы добавим позже. Сейчас — самое начало.'
		},
		{
			question: 'Нужно выбирать: автор или подписчик?',
			answer:
				'Нет. Один аккаунт подойдёт для обеих ролей. В личном кабинете можно переключаться между режимами автора и подписчика.'
		},
		{
			question: 'Что понадобится для регистрации?',
			answer:
				'Email, свободный никнейм и пароль от 12 символов. Затем останется подтвердить адрес по ссылке из письма.'
		}
	];
</script>

<svelte:head>
	<title>etheyl — делайте своё, собирайте своих</title>
	<meta
		name="description"
		content="etheyl — будущая платформа подписок для независимых авторов и их аудитории. Начните с одного аккаунта для творчества и поддержки."
	/>
</svelte:head>

<section class="home-hero" aria-labelledby="hero-title">
	<div class="hero-inner page-inset">
		<h1 id="hero-title">Делайте своё.<br /><span>Собирайте своих.</span></h1>
		<div class="hero-intro">
			<p>Большие идеи начинаются с вас.<br />А продолжаются с теми, кому не всё равно.</p>
			<p class="hero-definition">
				Строим etheyl — платформу подписок для независимых авторов и их аудитории.
			</p>
			<Button href={destination} variant="secondary" size="lg"
				>{data.user ? 'В личный кабинет' : 'Присоединиться'}<ArrowRight
					data-icon="inline-end"
				/></Button
			>
			<span class="hero-status"><span class="status-dot"></span>Сейчас открыта регистрация</span>
		</div>
		<CreatorCollage />
		<a class="hero-scroll" href="#about">Здесь всё начинается <ArrowDown aria-hidden="true" /></a>
	</div>
</section>

<section id="about" class="about-section page-inset" aria-labelledby="about-title">
	<div class="about-heading">
		<h2 id="about-title">За каждым «продолжай»<br />стоит человек.</h2>
		<p>
			Тот, кто ждёт следующую главу. Слушает до конца. Замечает детали. Мы делаем место для этой
			связи — между вашим делом и вашими людьми.
		</p>
	</div>
	<div class="audience-grid">
		<article class="audience-author">
			<div class="audience-symbol" aria-hidden="true">
				<svg viewBox="0 0 100 100"><path d="M50 5v90M5 50h90M18 18l64 64M18 82l64-64" /></svg>
			</div>
			<div>
				<h3>У вас есть своё дело.</h3>
				<p>
					Тексты, музыка, рисунки, исследования — и люди, которым это интересно. Будущий etheyl
					поможет оставаться с ними на связи через публикации и подписки.
				</p>
			</div>
			<a href={destination}>Я создаю <ArrowRight aria-hidden="true" /></a>
		</article>
		<article class="audience-supporter">
			<div class="audience-symbol orbit-symbol" aria-hidden="true">
				<svg viewBox="0 0 100 100"
					><ellipse cx="50" cy="50" rx="45" ry="23" transform="rotate(-35 50 50)" /><ellipse
						cx="50"
						cy="50"
						rx="45"
						ry="23"
						transform="rotate(35 50 50)"
					/></svg
				>
			</div>
			<div>
				<h3>Вам близко то, что делают другие.</h3>
				<p>
					У любимого творчества есть продолжение, когда рядом его аудитория. Мы хотим сделать
					поддержку авторов частью этой истории.
				</p>
			</div>
			<a href={destination}>Я поддерживаю <ArrowRight aria-hidden="true" /></a>
		</article>
	</div>
	<p class="shared-account">А если и то, и другое — тем лучше. Аккаунт всё равно один.</p>
</section>

<section id="join" class="joining-section" aria-labelledby="join-title">
	<div class="joining-inner page-inset">
		<div class="joining-heading">
			<h2 id="join-title">Начнём<br />со знакомства.</h2>
			<p>
				etheyl ещё растёт. Пока мы готовим инструменты для творчества и поддержки, вы уже можете
				создать свой аккаунт.
			</p>
			<Button href={destination} size="lg"
				>{data.user ? 'Открыть кабинет' : 'Создать аккаунт'}<ArrowRight
					data-icon="inline-end"
				/></Button
			>
		</div>
		<ol class="joining-steps">
			<li>
				<span class="step-number">1</span>
				<div>
					<h3>Выберите свой никнейм</h3>
					<p>Он станет основой адреса вашего будущего профиля.</p>
				</div>
			</li>
			<li>
				<span class="step-number">2</span>
				<div>
					<h3>Подтвердите почту</h3>
					<p>Отправим письмо со ссылкой. Один переход — и адрес подтверждён.</p>
				</div>
			</li>
			<li>
				<span class="step-number">3</span>
				<div>
					<h3>Освойтесь в аккаунте</h3>
					<p>Два режима — автор и подписчик. Публикации и подписки появятся позже.</p>
				</div>
			</li>
		</ol>
	</div>
</section>

<section class="faq-section page-inset" aria-labelledby="faq-title">
	<h2 id="faq-title">Кстати, о главном.</h2>
	<div class="faq-list">
		{#each questions as item (item.question)}<details>
				<summary>{item.question}<Plus aria-hidden="true" /></summary>
				<p>{item.answer}</p>
			</details>{/each}
	</div>
</section>

<section class="closing-section page-inset">
	<p>У творчества есть автор.<br />У продолжения — <span>вы.</span></p>
	<Button href={destination} variant="secondary" size="lg"
		>{data.user ? 'Вернуться в аккаунт' : 'Стать частью etheyl'}<ArrowRight
			data-icon="inline-end"
		/></Button
	>
</section>
