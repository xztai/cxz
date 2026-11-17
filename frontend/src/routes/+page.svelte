<script lang="ts">
	import { asset, resolve } from '$app/paths';
	import SiteHeader from '#lib/components/SiteHeader.svelte';
	import RegistrationForm from '#lib/components/RegistrationForm.svelte';
	import Icon from '#lib/components/Icon.svelte';
	function focusRegistration(event: MouseEvent) {
		event.preventDefault();
		const target =
			document.getElementById('email') ??
			document.querySelector<HTMLElement>('.registration-success');
		target?.focus({ preventScroll: true });
		document.getElementById('registration')?.scrollIntoView({ block: 'start' });
	}
	const questions = [
		{
			question: 'Что нужно, чтобы начать?',
			answer:
				'Создайте аккаунт: укажите email, никнейм и пароль. Затем подтвердите адрес по ссылке из письма. Регистрация и создание публичной страницы автора — отдельные шаги.'
		},
		{
			question: 'Когда появится моя страница автора?',
			answer:
				'После регистрации и подтверждения email можно настроить профиль автора. Само создание аккаунта ещё не публикует страницу: для этого нужно сохранить профиль.'
		},
		{
			question: 'Чем открытая публикация отличается от публикации по подписке?',
			answer:
				'Открытая публикация знакомит посетителей с вашей работой. Для публикации по подписке вы задаёте уровень доступа — её содержимое доступно подписчикам подходящего уровня.'
		},
		{
			question: 'Что делать, если письмо не пришло?',
			answer:
				'Проверьте правильность email и папку «Спам». Запросить новое письмо можно после регистрации или на странице подтверждения email. Ссылка для подтверждения действует 24 часа.'
		}
	];
</script>

<svelte:head>
	<title>etheyl — создавайте и делитесь работами</title>
	<meta
		name="description"
		content="Публикуйте иллюстрации, 3D-работы и другие творческие проекты. Открывайте доступ по подписке и получайте поддержку на etheyl."
	/>
</svelte:head>

<SiteHeader />
<main id="main">
	<section class="hero shell" aria-labelledby="hero-heading">
		<div class="hero-offer">
			<h1 id="hero-heading">Создавайте. Делитесь.<br /><span>Получайте поддержку.</span></h1>
			<p class="hero-description">Публикуйте работы и открывайте доступ по подписке.</p>
			<RegistrationForm />
		</div>
		<figure class="creator-demo" aria-labelledby="demo-caption">
			<figcaption id="demo-caption">
				<span class="demo-dot" aria-hidden="true"></span>Демонстрационный пример
			</figcaption>
			<div class="creator-page">
				<img
					class="creator-cover"
					src={asset('artwork/garden.webp')}
					alt="Иллюстрация: цветущий сад в оранжерее с видом на сказочный город"
					width="1440"
					height="960"
					fetchpriority="high"
				/>
				<div class="creator-intro">
					<img
						class="creator-avatar"
						src={asset('artwork/ceramic-cat.webp')}
						alt=""
						width="72"
						height="72"
					/>
					<div>
						<h2>Творческая мастерская</h2>
						<p>Иллюстрации, 3D и немного волшебства</p>
					</div>
				</div>
				<div class="demo-section-heading">
					<span>Работы и процесс</span><span>Иллюстрация / 3D</span>
				</div>
				<div class="demo-works">
					<div class="demo-work">
						<img
							src={asset('artwork/garden.webp')}
							alt="Открытый скетчбук и цветы в солнечной оранжерее"
							width="720"
							height="480"
						/>
						<div class="work-info">
							<span class="access-label">Открытая публикация</span>
							<h3>Там, где расцветают истории</h3>
						</div>
					</div>
					<div class="demo-work">
						<img
							src={asset('artwork/ceramic-cat.webp')}
							alt="3D-рендер керамического кота с синим цветочным узором и зелёным ростком"
							width="720"
							height="480"
						/>
						<div class="work-info">
							<span class="access-label subscriber"><Icon name="lock" size={12} />По подписке</span>
							<h3>От эскиза к 3D-персонажу</h3>
						</div>
					</div>
				</div>
				<p class="demo-note">
					Вымышленная мастерская. Пример того, как можно представить свои работы.
				</p>
			</div>
			<p class="demo-outside-note">
				Ваше творчество. Своя страница. Место для тех, кому близки ваши работы.
			</p>
		</figure>
	</section>

	<section class="setup" id="how-it-works" aria-labelledby="setup-heading">
		<div class="shell">
			<div class="section-intro">
				<h2 id="setup-heading">От первой идеи —<br />к своей странице</h2>
				<p>Начните с аккаунта.<br />Остальное — шаг за шагом.</p>
			</div>
			<ol class="setup-steps">
				<li>
					<span class="step-number">1</span>
					<h3>Создайте аккаунт</h3>
					<p>Выберите никнейм и подтвердите email по ссылке из письма.</p>
				</li>
				<li>
					<span class="step-number">2</span>
					<h3>Оформите страницу</h3>
					<p>Расскажите о себе и своём творчестве. Сохраните профиль автора.</p>
				</li>
				<li>
					<span class="step-number">3</span>
					<h3>Поделитесь работами</h3>
					<p>Публикуйте материалы открыто или настройте доступ по подписке.</p>
				</li>
			</ol>
		</div>
	</section>

	<section class="examples shell" aria-labelledby="examples-heading">
		<h2 id="examples-heading">У каждой работы<br />есть продолжение</h2>
		<p class="examples-intro">Покажите, что вы создаёте. Пригласите заглянуть глубже.</p>
		<article class="example-row">
			<figure class="example-image">
				<img
					src={asset('artwork/garden.webp')}
					alt="Авторская демонстрационная иллюстрация солнечной оранжереи с цветами и скетчбуком"
					width="1440"
					height="960"
					loading="lazy"
				/>
				<figcaption>Демонстрационный пример · Иллюстрация</figcaption>
			</figure>
			<div class="example-copy">
				<h3>От готовой иллюстрации<br />до первого наброска</h3>
				<p>
					Открытая работа помогает познакомиться с вашим миром. А подписка может стать приглашением
					в творческий процесс.
				</p>
				<dl>
					<div>
						<dt>
							<span class="access-mark" aria-hidden="true"><Icon name="check" size={16} /></span
							>Открыто для всех
						</dt>
						<dd>Готовая иллюстрация и история её создания.</dd>
					</div>
					<div>
						<dt>
							<span class="access-mark" aria-hidden="true"><Icon name="lock" size={16} /></span>По
							подписке
						</dt>
						<dd>Эскизы, этапы работы и разбор деталей.</dd>
					</div>
				</dl>
			</div>
		</article>
		<article class="example-row example-reverse">
			<figure class="example-image">
				<img
					src={asset('artwork/ceramic-cat.webp')}
					alt="Демонстрационная 3D-модель керамического кота с расписной глазурью"
					width="1000"
					height="667"
					loading="lazy"
				/>
				<figcaption>Демонстрационный пример · 3D-работа</figcaption>
			</figure>
			<div class="example-copy">
				<h3>За объёмом —<br />ваши находки и решения</h3>
				<p>
					У 3D-персонажа есть история: от простой формы до света и материалов. Делитесь результатом
					и тем, как вы к нему пришли.
				</p>
				<dl>
					<div>
						<dt>
							<span class="access-mark" aria-hidden="true"><Icon name="check" size={16} /></span
							>Открыто для всех
						</dt>
						<dd>Финальный рендер и знакомство с проектом.</dd>
					</div>
					<div>
						<dt>
							<span class="access-mark" aria-hidden="true"><Icon name="lock" size={16} /></span>По
							подписке
						</dt>
						<dd>Разбор моделирования, материалов и освещения.</dd>
					</div>
				</dl>
			</div>
		</article>
		<p class="examples-note">
			Это примеры публикаций. Вы сами выбираете, о чём рассказать и какой доступ открыть.
		</p>
	</section>

	<section class="faq shell" id="questions" aria-labelledby="faq-heading">
		<h2 id="faq-heading">Перед началом</h2>
		<div class="faq-list">
			{#each questions as item (item.question)}<details>
					<summary>{item.question}<span aria-hidden="true"><Icon name="plus" /></span></summary>
					<p>{item.answer}</p>
					{#if item.question === 'Что делать, если письмо не пришло?'}<a
							href={resolve('/auth/resend')}>Запросить новое письмо</a
						>{/if}
				</details>{/each}
		</div>
	</section>

	<section class="closing">
		<div class="shell closing-inner">
			<div>
				<h2>Дайте творчеству<br />своё место.</h2>
				<p>Начните с аккаунта на etheyl.</p>
			</div>
			<a class="button button-primary" href="#registration" onclick={focusRegistration}
				>Создать аккаунт<Icon name="arrow" /></a
			>
		</div>
	</section>
</main>
<footer class="site-footer shell">
	<a class="footer-wordmark" href={resolve('/')}>etheyl.</a>
	<p>Творчество, которым хочется делиться.</p>
	<nav aria-label="Навигация в подвале">
		<a href="#how-it-works">Как это работает</a><a href={resolve('/authors')}>Найти авторов</a><a
			href={resolve('/login')}>Войти</a
		>
	</nav>
</footer>

<style>
	.hero {
		display: grid;
		grid-template-columns: 0.91fr 1.09fr;
		gap: 48px;
		align-items: center;
		padding-block: 50px 64px;
	}
	h1 {
		margin-bottom: 22px;
		font-size: clamp(38px, 3.2vw, 46px);
		line-height: 1.12;
		font-weight: 800;
		letter-spacing: -0.04em;
	}
	h1 span {
		color: var(--blue);
	}
	.hero-description {
		color: var(--muted);
		font-size: 18px;
		line-height: 1.65;
		max-width: 360px;
		margin-bottom: 30px;
	}
	.creator-demo {
		margin: 0;
		align-self: center;
		min-width: 0;
	}
	.creator-demo > figcaption {
		display: flex;
		align-items: center;
		gap: 8px;
		color: #4d5878;
		font-size: 11px;
		font-weight: 600;
		margin: 0 0 12px 4px;
	}
	.demo-dot {
		width: 6px;
		height: 6px;
		background: var(--blue);
		border-radius: 50%;
	}
	.creator-page {
		background: var(--white);
		border-radius: 16px;
		padding: 10px;
		box-shadow:
			0 16px 60px -16px #202e5a30,
			0 3px 12px #202e5a0c;
	}
	.creator-cover {
		width: 100%;
		height: 192px;
		border-radius: 9px;
		object-fit: cover;
		object-position: center 38%;
	}
	.creator-intro {
		display: flex;
		align-items: center;
		gap: 13px;
		padding: 0 14px 18px;
		margin-top: -20px;
		position: relative;
	}
	.creator-avatar {
		flex-shrink: 0;
		width: 68px;
		height: 68px;
		object-fit: cover;
		border: 4px solid white;
		border-radius: 15px;
	}
	.creator-intro h2 {
		margin: 22px 0 4px;
		font-size: 18px;
		font-weight: 800;
		line-height: 1.25;
		letter-spacing: -0.02em;
	}
	.creator-intro p {
		font-size: 12px;
		color: var(--muted);
		margin: 0;
	}
	.demo-section-heading {
		display: flex;
		justify-content: space-between;
		gap: 10px;
		padding: 12px 8px;
		border-top: 1px solid var(--line);
		font-size: 11px;
		font-weight: 700;
	}
	.demo-section-heading span:last-child {
		color: var(--muted);
		font-size: 10px;
		font-weight: 400;
	}
	.demo-works {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 10px;
	}
	.demo-work {
		min-width: 0;
	}
	.demo-work img {
		width: 100%;
		height: 205px;
		object-fit: cover;
		border-radius: 8px;
	}
	.demo-work:first-child img {
		object-position: 35% center;
	}
	.work-info {
		padding: 10px 5px 6px;
	}
	.access-label {
		display: flex;
		align-items: center;
		gap: 5px;
		font-size: 11px;
		color: var(--muted);
	}
	.subscriber {
		color: var(--blue-hover);
	}
	.work-info h3 {
		margin: 5px 0 0;
		font-size: 14px;
		line-height: 1.5;
		font-weight: 700;
	}
	.demo-note {
		border-top: 1px solid var(--line);
		margin: 10px 5px 0;
		padding: 12px 0 4px;
		font-size: 11px;
		line-height: 1.5;
		color: var(--muted);
	}
	.demo-outside-note {
		text-align: center;
		color: var(--muted);
		font-size: 10px;
		line-height: 1.6;
		max-width: 350px;
		margin: 20px auto 0;
	}
	.setup {
		background: var(--surface);
		border-block: 1px solid var(--line);
		padding-block: 64px;
		scroll-margin-top: 0;
	}
	.section-intro {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 28px;
		margin-bottom: 44px;
	}
	.section-intro h2,
	.examples > h2,
	.faq > h2 {
		font-size: clamp(30px, 3.3vw, 44px);
		font-weight: 800;
		letter-spacing: -0.035em;
		line-height: 1.18;
		margin: 0;
	}
	.section-intro > p {
		color: var(--muted);
		font-size: 15px;
		line-height: 1.75;
		margin: 0;
	}
	.setup-steps {
		list-style: none;
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		padding: 0;
		margin: 0;
		gap: 48px;
	}
	.setup-steps li {
		border-top: 1px solid #ccd3e8;
		padding-top: 22px;
	}
	.step-number {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 32px;
		height: 32px;
		border-radius: 50%;
		background: #e4eaff;
		color: var(--blue-hover);
		font-size: 14px;
		font-weight: 700;
		margin-bottom: 19px;
	}
	.setup-steps h3 {
		font-size: 19px;
		line-height: 1.4;
		font-weight: 800;
		margin-bottom: 10px;
	}
	.setup-steps p {
		font-size: 13px;
		line-height: 1.8;
		max-width: 270px;
		color: var(--muted);
		margin: 0;
	}
	.examples {
		padding-block: 96px 72px;
	}
	.examples-intro {
		margin: 18px 0 48px;
		color: var(--muted);
		font-size: 16px;
		line-height: 1.7;
	}
	.example-row {
		display: grid;
		grid-template-columns: 1.08fr 0.92fr;
		align-items: center;
		gap: 80px;
		margin-bottom: 60px;
	}
	.example-image {
		margin: 0;
		min-width: 0;
	}
	.example-image img {
		width: 100%;
		aspect-ratio: 3/2;
		object-fit: cover;
		border-radius: var(--radius);
	}
	.example-image figcaption {
		color: var(--muted);
		font-size: 10px;
		margin-top: 12px;
	}
	.example-copy h3 {
		font-size: 29px;
		line-height: 1.3;
		font-weight: 800;
		letter-spacing: -0.025em;
		margin-bottom: 18px;
	}
	.example-copy > p {
		font-size: 14px;
		line-height: 1.8;
		color: var(--muted);
		max-width: 400px;
	}
	.example-copy dl {
		margin: 26px 0 0;
	}
	.example-copy dl > div + div {
		margin-top: 22px;
	}
	.example-copy dt {
		display: flex;
		align-items: center;
		gap: 10px;
		font-size: 12px;
		font-weight: 800;
	}
	.access-mark {
		display: inline-flex;
		color: var(--blue);
	}
	.example-copy dd {
		margin: 7px 0 0 26px;
		color: var(--muted);
		font-size: 12px;
		line-height: 1.7;
	}
	.example-reverse .example-image {
		grid-column: 2;
		grid-row: 1;
	}
	.example-reverse .example-copy {
		grid-column: 1;
		grid-row: 1;
	}
	.examples-note {
		font-size: 12px;
		color: var(--muted);
		text-align: center;
		margin: -12px 0 0;
		line-height: 1.7;
	}
	.faq {
		display: grid;
		grid-template-columns: 0.8fr 1.2fr;
		gap: 80px;
		padding-block: 35px 90px;
	}
	.faq-list {
		border-top: 1px solid var(--line);
	}
	details {
		border-bottom: 1px solid var(--line);
		padding: 22px 0;
	}
	summary {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 20px;
		list-style: none;
		cursor: pointer;
		font-size: 14px;
		line-height: 1.65;
		font-weight: 700;
	}
	summary::-webkit-details-marker {
		display: none;
	}
	summary span {
		display: inline-flex;
		flex-shrink: 0;
		color: var(--blue);
	}
	details[open] summary span {
		transform: rotate(45deg);
	}
	details p {
		margin: 15px 26px 0 0;
		font-size: 13px;
		line-height: 1.85;
		color: var(--muted);
	}
	details a {
		display: inline-block;
		margin-top: 16px;
		color: var(--blue);
		font-size: 13px;
	}
	.closing {
		background: var(--blue-soft);
		padding-block: 58px;
	}
	.closing-inner {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 30px;
	}
	.closing h2 {
		font-size: clamp(32px, 3.5vw, 46px);
		font-weight: 800;
		line-height: 1.15;
		letter-spacing: -0.035em;
		margin-bottom: 14px;
	}
	.closing p {
		color: var(--muted);
		font-size: 14px;
		margin: 0;
	}
	.closing .button {
		min-width: 230px;
		justify-content: space-between;
	}
	.site-footer {
		display: flex;
		gap: 24px;
		align-items: center;
		padding-block: 32px;
	}
	.footer-wordmark {
		font-size: 25px;
		font-weight: 800;
		letter-spacing: -0.04em;
		text-decoration: none;
	}
	.site-footer p {
		margin: 0;
		color: var(--muted);
		font-size: 10px;
	}
	.site-footer nav {
		margin-left: auto;
		display: flex;
		gap: 24px;
		font-size: 11px;
	}
	.site-footer nav a {
		text-decoration: none;
	}
	.site-footer nav a:hover {
		color: var(--blue);
	}
	@media (max-width: 1100px) {
		.hero {
			gap: 32px;
			padding-block: 50px 60px;
		}
		h1 {
			font-size: 36px;
		}
		.creator-cover {
			height: 170px;
		}
		.demo-work img {
			height: 175px;
		}
		.creator-intro h2 {
			font-size: 16px;
		}
		.example-row,
		.faq {
			gap: 45px;
		}
		.example-copy h3 {
			font-size: 25px;
		}
	}
	@media (max-width: 950px) {
		.hero {
			grid-template-columns: 1fr;
			gap: 44px;
			max-width: 580px;
		}
		h1 {
			font-size: clamp(40px, 6.3vw, 52px);
		}
		.hero-description {
			max-width: 400px;
		}
		.hero-offer :global(.registration) {
			max-width: none;
		}
		.creator-cover {
			height: 180px;
		}
		.demo-work img {
			height: 220px;
		}
		.section-intro {
			align-items: flex-start;
		}
		.setup-steps {
			gap: 24px;
		}
		.setup-steps h3 {
			font-size: 16px;
		}
		.example-row {
			gap: 32px;
		}
		.example-copy h3 {
			font-size: 23px;
		}
		.faq {
			grid-template-columns: 1fr;
			gap: 30px;
		}
		.site-footer {
			flex-wrap: wrap;
		}
		.site-footer nav {
			width: 100%;
			margin: 0;
		}
	}
	@media (max-width: 560px) {
		.hero {
			padding-block: 38px 44px;
			gap: 36px;
		}
		h1 {
			font-size: clamp(36px, 8vw, 44px);
		}
		.hero-description {
			font-size: 16px;
			margin-bottom: 26px;
		}
		.creator-cover {
			height: 150px;
		}
		.creator-intro {
			padding-inline: 5px;
			gap: 8px;
		}
		.creator-avatar {
			width: 56px;
			height: 56px;
			border-radius: 12px;
		}
		.creator-intro h2 {
			font-size: 17px;
		}
		.creator-intro p {
			font-size: 11px;
		}
		.demo-works {
			grid-template-columns: 1fr;
			gap: 18px;
		}
		.demo-work img {
			height: 210px;
		}
		.work-info h3 {
			font-size: 14px;
		}
		.access-label {
			font-size: 11px;
		}
		.demo-note {
			font-size: 11px;
		}
		.demo-section-heading {
			font-size: 12px;
		}
		.demo-section-heading span:last-child {
			font-size: 10px;
		}
		.setup {
			padding-block: 42px;
		}
		.section-intro {
			flex-direction: column;
			gap: 18px;
			margin-bottom: 30px;
		}
		.section-intro > p {
			font-size: 14px;
		}
		.setup-steps {
			grid-template-columns: 1fr;
			gap: 26px;
		}
		.setup-steps li {
			position: relative;
			padding: 22px 0 0 48px;
		}
		.step-number {
			position: absolute;
			left: 0;
			top: 24px;
		}
		.setup-steps h3 {
			font-size: 18px;
		}
		.setup-steps p {
			max-width: 330px;
		}
		.examples {
			padding-block: 54px 40px;
		}
		.examples-intro {
			margin-bottom: 30px;
			font-size: 14px;
		}
		.example-row {
			grid-template-columns: 1fr;
			gap: 24px;
			margin-bottom: 42px;
		}
		.example-reverse .example-image,
		.example-reverse .example-copy {
			grid-column: auto;
			grid-row: auto;
		}
		.example-copy h3 {
			font-size: 26px;
		}
		.example-copy > p {
			max-width: none;
		}
		.examples-note {
			font-size: 11px;
			margin: 0;
			text-align: left;
		}
		.faq {
			padding-block: 25px 54px;
		}
		.closing {
			padding-block: 40px;
		}
		.closing-inner {
			flex-direction: column;
			align-items: flex-start;
			gap: 26px;
		}
		.closing .button {
			width: 100%;
		}
		.site-footer {
			gap: 12px;
		}
		.site-footer p {
			width: 100%;
		}
		.site-footer nav {
			flex-wrap: wrap;
			gap: 18px;
			margin-top: 12px;
		}
	}
</style>
