<script lang="ts">
	import { resolve } from '$app/paths';
	import { invalidate } from '$app/navigation';
	import { api, errorMessage } from '#lib/api/client.js';
	import { getSession } from '#lib/session.svelte.js';
	const session = getSession();
	let busy = $state(false);
	let message = $state('');
	async function logout() {
		if (busy) return;
		busy = true;
		message = '';
		try {
			await api.logout();
			session.setUser(null);
			await invalidate('studio:profile');
		} catch (error) {
			message = errorMessage(error);
		} finally {
			busy = false;
		}
	}
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
			{#if session.user}<a href={resolve('/studio')}>Студия</a>{/if}
		</nav>
		<div class="account-menu">
			{#if session.user}
				<span class="account-name">@{session.user.nickname}</span>
				<button
					class="button button-outline login-link"
					type="button"
					disabled={busy}
					onclick={logout}>{busy ? 'Выходим…' : 'Выйти'}</button
				>
			{:else if session.ready}
				<a class="button button-outline login-link" href={resolve('/login')}>Войти</a>
			{/if}
		</div>
	</div>
	{#if message}<p class="shell feedback error-feedback" role="alert">{message}</p>{/if}
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
	.account-menu {
		margin-left: auto;
		display: flex;
		align-items: center;
		gap: 12px;
		min-width: 92px;
	}
	.account-name {
		font-size: 14px;
		font-weight: 600;
		overflow-wrap: anywhere;
	}
	.login-link {
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
			gap: 12px;
			font-size: 12px;
		}
		.how-link {
			display: none;
		}
		.account-menu {
			margin-left: 0;
			min-width: 0;
			flex: none;
			flex-wrap: wrap;
			justify-content: flex-end;
		}
		.account-name {
			display: none;
		}
		.login-link {
			padding: 8px 14px;
		}
	}
	@media (max-width: 380px) {
		.header-inner {
			gap: 8px;
		}
		.wordmark {
			font-size: 26px;
		}
		nav {
			gap: 8px;
		}
		.login-link {
			padding-inline: 11px;
		}
	}
</style>
