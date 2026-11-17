<script lang="ts">
	import { resolve } from '$app/paths';
</script>

<a class="skip-link" href="#main">Перейти к содержимому</a>
<header class="site-header">
	<div class="shell header-inner">
		<a class="wordmark" href={resolve('/')} aria-label="etheyl — главная"
			>etheyl<span aria-hidden="true">.</span></a
		>
		<nav aria-label="Основная навигация">
			<a class="how-link" href={resolve('/#how-it-works')}>Как это работает</a>
			<a href={resolve('/authors')}>Найти авторов</a>
		</nav>
		<a class="button button-outline login-link" href={resolve('/login')}>Войти</a>
	</div>
</header>

<style>
	.site-header {
		border-bottom: 1px solid var(--line);
		background: var(--white);
	}
	.header-inner {
		min-height: 88px;
		display: flex;
		align-items: center;
		gap: 64px;
	}
	.wordmark {
		color: var(--ink);
		font-size: 36px;
		font-weight: 800;
		letter-spacing: -0.04em;
		text-decoration: none;
	}
	.wordmark span {
		color: var(--blue);
	}
	nav {
		display: flex;
		align-items: center;
		gap: 34px;
		font-size: 14px;
		font-weight: 600;
	}
	nav a {
		text-decoration: none;
	}
	nav a:hover {
		color: var(--blue);
	}
	.login-link {
		margin-left: auto;
		min-height: 42px;
		padding: 9px 25px;
		font-size: 14px;
	}
	@media (max-width: 640px) {
		.header-inner {
			min-height: 74px;
			gap: 18px;
		}
		.wordmark {
			font-size: 29px;
		}
		nav {
			margin-left: auto;
			font-size: 12px;
		}
		.how-link {
			display: none;
		}
		.login-link {
			margin-left: 0;
			padding: 8px 14px;
		}
	}
</style>
