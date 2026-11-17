<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { resolve } from '$app/paths';
	import { api, ApiError, errorMessage } from '#lib/api/client.js';
	import { getSession } from '#lib/session.svelte.js';
	import { emailError } from '#lib/validation.js';
	import Icon from './Icon.svelte';
	import PasswordField from './PasswordField.svelte';

	const session = getSession();
	let hydrated = $state(false);
	let now = $state(0);
	let busy = $state(false);
	let editing = $state(false);
	let newEmail = $state('');
	let password = $state('');
	let errors = $state({ email: '', password: '' });
	let message = $state('');
	let notice = $state('');
	let pending = $derived(session.verification);
	let remaining = $derived(Math.max(0, Math.ceil(((pending?.resendAt ?? 0) - now) / 1000)));
	let title = $derived(editing ? 'Изменить email' : 'Проверьте почту');

	onMount(() => {
		session.restoreVerification();
		now = Date.now();
		hydrated = true;
		const timer = window.setInterval(() => (now = Date.now()), 1000);
		return () => window.clearInterval(timer);
	});

	function startEditing() {
		newEmail = pending?.email ?? '';
		password = '';
		errors = { email: '', password: '' };
		message = '';
		notice = '';
		editing = true;
	}

	function cancelEditing() {
		password = '';
		message = '';
		editing = false;
	}

	async function resend() {
		if (busy || !pending || Date.now() < pending.resendAt) return;
		busy = true;
		message = '';
		notice = '';
		try {
			await api.resend(pending.email);
			session.setVerification(pending.email, true);
			now = Date.now();
			notice = 'Письмо отправлено повторно. Проверьте также папку «Спам».';
		} catch (error) {
			message = errorMessage(error);
			if (error instanceof ApiError && error.status === 429) {
				session.waitForResend(Math.max(60, error.retryAfter || 900));
				now = Date.now();
			}
		} finally {
			busy = false;
		}
	}

	async function changeEmail(event: SubmitEvent) {
		event.preventDefault();
		if (busy || !pending) return;
		const form = event.currentTarget as HTMLFormElement;
		errors = {
			email:
				emailError(newEmail) ||
				(newEmail.trim().toLowerCase() === pending.email ? 'Укажите другой email.' : ''),
			password: password ? '' : 'Введите текущий пароль.'
		};
		message = '';
		if (Object.values(errors).some(Boolean)) {
			await tick();
			form.querySelector<HTMLInputElement>('[aria-invalid="true"]')?.focus();
			return;
		}
		busy = true;
		try {
			await api.changeUnverifiedEmail({
				email: pending.email,
				newEmail: newEmail.trim(),
				password
			});
			password = '';
			session.setVerification(newEmail, true);
			now = Date.now();
			editing = false;
			notice =
				'Email изменён. Мы отправили письмо на новый адрес. Прежняя ссылка больше не действует.';
		} catch (error) {
			message = errorMessage(error);
			if (error instanceof ApiError && error.code === 'email_exists') errors.email = message;
			if (error instanceof ApiError && error.code === 'credentials') {
				errors.password = 'Неверный пароль. Попробуйте ещё раз.';
			}
		} finally {
			password = '';
			busy = false;
		}
	}
</script>

<svelte:document onvisibilitychange={() => (now = Date.now())} />

<svelte:head>
	<title>{title} — etheyl</title>
	<meta name="robots" content="noindex" />
	<meta name="referrer" content="no-referrer" />
</svelte:head>

<div class="check-email-panel">
	<noscript><p class="feedback">Для работы с аккаунтом включите JavaScript в браузере.</p></noscript
	>
	<span class="mail-symbol"><Icon name="mail" size={30} /></span>
	<h1>{title}</h1>
	{#if hydrated && pending}
		{#if editing}
			<p class="intro">
				Текущий адрес: <strong>{pending.email}</strong>. Укажите новый email и пароль аккаунта.
			</p>
			<form method="POST" onsubmit={changeEmail} novalidate aria-busy={busy}>
				<div class="field">
					<label for="new-email">Новый email</label>
					<input
						id="new-email"
						name="newEmail"
						type="email"
						bind:value={newEmail}
						autocomplete="email"
						required
						maxlength="254"
						disabled={busy}
						aria-invalid={!!errors.email}
						aria-describedby={errors.email ? 'new-email-error' : undefined}
						{@attach (input) => input.focus()}
					/>
					{#if errors.email}<p class="field-error" id="new-email-error">{errors.email}</p>{/if}
				</div>
				<PasswordField
					bind:value={password}
					error={errors.password}
					autocomplete="current-password"
					hint={false}
					disabled={busy}
				/>
				<p class="field-hint">Никнейм и пароль сохранятся. Подтвердить нужно будет новый адрес.</p>
				<p class="feedback error-feedback" role="alert">{message}</p>
				<button class="button button-primary" type="submit" disabled={busy}>
					{busy ? 'Сохраняем…' : 'Сохранить и отправить письмо'}
				</button>
				<button
					class="text-button cancel-button"
					type="button"
					disabled={busy}
					onclick={cancelEditing}
				>
					Отмена
				</button>
			</form>
		{:else}
			<p class="intro">
				{#if pending.sent}Мы отправили письмо на <strong>{pending.email}</strong>.
				{:else}Адрес <strong>{pending.email}</strong> ещё не подтверждён.{/if}
				Перейдите по ссылке из письма, чтобы подтвердить email и войти в аккаунт.
			</p>
			<p class="help">Не нашли письмо? Проверьте папку «Спам» или отправьте его ещё раз.</p>
			<button
				class="button button-primary"
				type="button"
				disabled={busy || remaining > 0}
				aria-describedby={remaining > 0 ? 'resend-timer' : undefined}
				onclick={resend}>{busy ? 'Отправляем…' : 'Отправить письмо ещё раз'}</button
			>
			{#if remaining > 0}
				<p class="timer" id="resend-timer">Повторная отправка через {remaining} сек.</p>
			{/if}
			<p class="feedback" role="status">{notice}</p>
			<p class="feedback error-feedback" role="alert">{message}</p>
			<div class="links">
				<button class="text-button" type="button" disabled={busy} onclick={startEditing}>
					Изменить email
				</button>
				<a href={resolve('/login')}>Перейти ко входу</a>
			</div>
		{/if}
	{:else if hydrated}
		<p class="intro">
			Войдите с email и паролем аккаунта, чтобы запросить новое письмо подтверждения.
		</p>
		<a class="button button-primary" href={resolve('/login')}>Перейти ко входу</a>
	{:else}
		<p class="intro">Открываем информацию о подтверждении…</p>
	{/if}
</div>

<style>
	.check-email-panel {
		width: min(420px, 100%);
		margin-inline: auto;
		padding-block: 12px;
	}
	h1 {
		margin-bottom: 20px;
	}
	.mail-symbol {
		display: inline-flex;
		color: var(--blue);
		margin-bottom: 24px;
	}
	.intro,
	.help {
		color: var(--muted);
		font-size: 14px;
		line-height: 1.8;
		margin-bottom: 24px;
		overflow-wrap: anywhere;
	}
	.intro strong {
		color: var(--ink);
	}
	.button {
		width: 100%;
	}
	button:disabled {
		cursor: default;
	}
	.timer {
		margin: 12px 0 0;
		color: var(--muted);
		font-size: 12px;
		font-variant-numeric: tabular-nums;
	}
	.links {
		display: flex;
		flex-wrap: wrap;
		gap: 12px 24px;
		margin-top: 24px;
		font-size: 13px;
		line-height: 1.8;
		color: var(--blue-hover);
	}
	.text-button {
		padding: 0;
		border: 0;
		background: none;
		color: var(--blue-hover);
		font: inherit;
		text-decoration: underline;
		text-underline-offset: 4px;
		cursor: pointer;
	}
	form .button {
		margin-top: 20px;
	}
	.cancel-button {
		display: block;
		margin: 20px auto 0;
		font-size: 13px;
	}
</style>
