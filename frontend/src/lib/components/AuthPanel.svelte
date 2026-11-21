<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { resolve } from '$app/paths';
	import { goto } from '$app/navigation';
	import { api, ApiError, errorMessage, type User } from '#lib/api/client.js';
	import { emailError, passwordError } from '#lib/validation.js';
	import PasswordField from './PasswordField.svelte';
	import Icon from './Icon.svelte';
	import { getSession } from '#lib/session.svelte.js';
	const session = getSession();
	let {
		mode,
		returnAuthor,
		returnPost
	}: {
		mode: 'login' | 'verify' | 'forgot' | 'reset';
		returnAuthor?: string;
		returnPost?: string;
	} = $props();
	let email = $state('');
	let password = $state('');
	let token = $state('');
	let busy = $state(false);
	let hydrated = $state(false);
	let ready = $state(false);
	let message = $state('');
	let success = $state(false);
	let confirmedUser = $state<User | null>(null);
	let user = $derived(mode === 'login' ? session.user : confirmedUser);
	let complete = $derived(mode === 'login' ? user !== null : success);
	let errors = $state({ email: '', password: '' });
	const titles = {
		login: 'С возвращением',
		verify: 'Подтвердите email',
		forgot: 'Восстановить пароль',
		reset: 'Новый пароль'
	};
	let title = $derived(
		complete
			? user
				? mode === 'verify'
					? 'Email подтверждён'
					: 'Вы вошли в аккаунт'
				: mode === 'reset'
					? 'Пароль обновлён'
					: 'Проверьте почту'
			: titles[mode]
	);
	onMount(() => {
		hydrated = true;
		if (mode !== 'verify' && mode !== 'reset') return;
		token = new URLSearchParams(window.location.hash.slice(1)).get('token') ?? '';
		// Keep the action token out of browser history, URLs and referrers after reading it.
		window.history.replaceState(window.history.state, '', window.location.pathname);
		if (!/^[A-Za-z0-9_-]{43}$/.test(token)) {
			message = 'В ссылке нет корректного кода. Запросите новое письмо.';
			return;
		}
		if (mode === 'reset') {
			ready = true;
			return;
		}
		void preview();
	});
	async function preview() {
		busy = true;
		message = '';
		try {
			email = (await api.preview(token)).email;
			ready = true;
		} catch (error) {
			message = errorMessage(error);
		} finally {
			busy = false;
		}
	}
	async function submit(event: SubmitEvent) {
		event.preventDefault();
		const form = event.currentTarget as HTMLFormElement;
		if (busy) return;
		message = '';
		errors = {
			email: mode === 'verify' || mode === 'reset' ? '' : emailError(email),
			password:
				mode === 'reset'
					? passwordError(password)
					: mode === 'login' && !password
						? 'Введите пароль.'
						: ''
		};
		if (Object.values(errors).some(Boolean)) {
			message = 'Проверьте выделенные поля.';
			await tick();
			form.querySelector<HTMLInputElement>('[aria-invalid="true"]')?.focus();
			return;
		}
		busy = true;
		try {
			if (mode === 'login') {
				session.setUser((await api.login({ email: email.trim(), password })).user);
			} else if (mode === 'verify') {
				confirmedUser = (await api.verify(token)).user;
				session.setUser(confirmedUser);
			} else if (mode === 'forgot') await api.forgot(email.trim());
			else {
				await api.reset(token, password);
				session.setUser(null);
			}
			success = true;
			password = '';
			token = '';
		} catch (error) {
			if (mode === 'login' && error instanceof ApiError && error.code === 'email_unverified') {
				password = '';
				session.setVerification(email);
				await goto(resolve('/auth/check-email'));
				return;
			}
			message = errorMessage(error);
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head
	><title>{title} — etheyl</title><meta name="robots" content="noindex" /><meta
		name="referrer"
		content="no-referrer"
	/></svelte:head
>
<div class="auth-panel">
	<noscript><p class="feedback">Для работы с аккаунтом включите JavaScript в браузере.</p></noscript
	>
	{#if complete}<span class="auth-symbol"
			><Icon name={user || mode === 'reset' ? 'check' : 'mail'} size={30} /></span
		>{/if}
	<h1>{title}</h1>
	<div aria-live="polite">
		{#if complete}
			{#if user}<p class="intro">
					Добро пожаловать, <strong>{user.nickname}</strong>. Теперь можно знакомиться с авторами и
					их работами.
				</p>
				<a
					class="button button-primary"
					href="{returnPost
						? resolve('/posts/[id]', { id: returnPost })
						: returnAuthor
							? resolve('/authors/[nickname]', { nickname: returnAuthor })
							: resolve('/authors')}{returnPost ? '#comments' : ''}"
					>{returnPost
						? 'Вернуться к посту'
						: returnAuthor
							? 'Вернуться к автору'
							: 'Найти авторов'}
					<Icon name="arrow" /></a
				>
			{:else if mode === 'reset'}<p class="intro">Используйте новый пароль для входа в аккаунт.</p>
				<a class="button button-primary" href={resolve('/login')}>Войти</a>
			{:else}<p class="intro">
					Если адрес подходит для этого действия, мы отправили письмо. Проверьте почту и папку
					«Спам».
				</p>
				<a class="button button-primary" href={resolve('/login')}>Перейти ко входу</a>{/if}
		{/if}
	</div>
	{#if !complete}
		{#if mode === 'login'}<p class="intro">Войдите, чтобы продолжить.</p>
		{:else if mode === 'verify'}<p class="intro">
				{busy && !ready
					? 'Проверяем ссылку…'
					: ready
						? `Подтвердите, что ${email} — ваш адрес.`
						: 'Для подтверждения откройте ссылку из письма.'}
			</p>
		{:else if mode === 'reset'}<p class="intro">Придумайте новый пароль для своего аккаунта.</p>
		{:else}<p class="intro">Укажите email аккаунта. Мы отправим ссылку для смены пароля.</p>{/if}
		<form method="POST" onsubmit={submit} novalidate aria-busy={busy}>
			{#if mode === 'login' || mode === 'forgot'}<div class="field">
					<label for="auth-email">Email</label><input
						id="auth-email"
						type="email"
						name="email"
						bind:value={email}
						autocomplete="email"
						required
						maxlength="254"
						disabled={busy || !hydrated}
						aria-invalid={!!errors.email}
						aria-describedby={errors.email ? 'auth-email-error' : undefined}
					/>{#if errors.email}<p class="field-error" id="auth-email-error">{errors.email}</p>{/if}
				</div>{/if}
			{#if mode === 'login' || (mode === 'reset' && ready)}<PasswordField
					bind:value={password}
					error={errors.password}
					autocomplete={mode === 'login' ? 'current-password' : 'new-password'}
					hint={mode !== 'login'}
					disabled={busy || !hydrated}
				/>{/if}
			<p class="feedback error-feedback" role="alert">{message}</p>
			{#if (mode !== 'verify' && mode !== 'reset') || ready}<button
					class="button button-primary"
					type="submit"
					disabled={busy || !hydrated}
					>{busy
						? 'Подождите…'
						: mode === 'login'
							? 'Войти'
							: mode === 'verify'
								? 'Подтвердить email и войти'
								: mode === 'reset'
									? 'Сохранить пароль'
									: 'Отправить письмо'}</button
				>{/if}
		</form>
		<div class="auth-links">
			{#if mode === 'login'}<a href={resolve('/auth/forgot')}>Забыли пароль?</a>
				<p>Ещё нет аккаунта? <a href={resolve('/#registration')}>Зарегистрироваться</a></p>
			{:else if mode === 'verify'}{#if token && !ready && !busy}<button
						type="button"
						class="retry-link"
						onclick={preview}>Проверить ссылку ещё раз</button
					>{/if}<a href={resolve('/auth/check-email')}>Запросить новое письмо</a>
			{:else if mode === 'reset'}<a href={resolve('/auth/forgot')}
					>Запросить ссылку для смены пароля</a
				>
			{:else}<a href={resolve('/login')}>Вернуться ко входу</a>{/if}
		</div>
	{/if}
</div>

<style>
	.auth-panel {
		width: min(420px, 100%);
		margin-inline: auto;
		padding-block: 12px;
	}
	.auth-panel h1 {
		font-size: 36px;
		margin-bottom: 20px;
	}
	.intro {
		color: var(--muted);
		font-size: 14px;
		line-height: 1.8;
		margin-bottom: 30px;
		overflow-wrap: anywhere;
	}
	form .button {
		width: 100%;
		margin-top: 8px;
	}
	.auth-links {
		margin-top: 26px;
		display: flex;
		flex-wrap: wrap;
		gap: 18px;
		font-size: 12px;
		line-height: 1.8;
	}
	.auth-links a,
	.retry-link {
		color: var(--blue-hover);
	}
	.auth-links p {
		margin: 0;
		color: var(--muted);
		width: 100%;
	}
	.auth-symbol {
		display: inline-flex;
		color: var(--blue);
		margin-bottom: 24px;
	}
	.retry-link {
		padding: 0;
		border: 0;
		background: none;
		font: inherit;
		text-decoration: underline;
		text-underline-offset: 4px;
		cursor: pointer;
	}
</style>
