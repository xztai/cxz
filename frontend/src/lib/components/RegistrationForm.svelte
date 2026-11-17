<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { resolve } from '$app/paths';
	import { api, ApiError, errorMessage } from '#lib/api/client.js';
	import { emailError, nicknameError, passwordError } from '#lib/validation.js';
	import PasswordField from './PasswordField.svelte';
	import Icon from './Icon.svelte';
	let email = $state('');
	let nickname = $state('');
	let password = $state('');
	let errors = $state({ email: '', nickname: '', password: '' });
	let busy = $state(false);
	let hydrated = $state(false);
	onMount(() => {
		hydrated = true;
	});
	let success = $state(false);
	let message = $state('');
	let resendMessage = $state('');
	function focusSuccess(panel: HTMLDivElement) {
		panel.focus();
	}
	async function submit(event: SubmitEvent) {
		event.preventDefault();
		const form = event.currentTarget as HTMLFormElement;
		if (busy) return;
		errors = {
			email: emailError(email),
			nickname: nicknameError(nickname),
			password: passwordError(password)
		};
		message = '';
		if (Object.values(errors).some(Boolean)) {
			message = 'Проверьте выделенные поля.';
			await tick();
			form.querySelector<HTMLInputElement>('[aria-invalid="true"]')?.focus();
			return;
		}
		busy = true;
		try {
			await api.register({ email: email.trim(), nickname, password });
			password = '';
			success = true;
		} catch (error) {
			message = errorMessage(error);
			if (error instanceof ApiError && error.code === 'email_exists') errors.email = message;
			if (error instanceof ApiError && error.code === 'nickname_exists') errors.nickname = message;
		} finally {
			busy = false;
		}
	}
	async function resend() {
		busy = true;
		resendMessage = '';
		try {
			await api.resend(email.trim());
			resendMessage =
				'Если адрес подходит для подтверждения, письмо отправлено. Проверьте также папку «Спам».';
		} catch (error) {
			resendMessage = errorMessage(error);
		} finally {
			busy = false;
		}
	}
</script>

<div id="registration" class="registration">
	<noscript><p class="feedback">Для регистрации включите JavaScript в браузере.</p></noscript>
	{#if success}
		<div class="registration-success" role="status" tabindex="-1" {@attach focusSuccess}>
			<span class="success-icon"><Icon name="mail" size={26} /></span>
			<h2>Проверьте почту</h2>
			<p>
				Мы отправили письмо на <strong>{email}</strong>. Перейдите по ссылке, чтобы подтвердить
				адрес и завершить регистрацию.
			</p>
			<p class="field-hint">Не нашли письмо? Загляните в папку «Спам» или запросите новое.</p>
			<div class="success-actions">
				<button
					class="button button-outline"
					type="button"
					disabled={busy || !hydrated}
					onclick={resend}>{busy ? 'Отправляем…' : 'Отправить письмо ещё раз'}</button
				>
				<a href={resolve('/login')}>Перейти ко входу</a>
			</div>
			<p class="feedback" aria-live="polite">{resendMessage}</p>
		</div>
	{:else}
		<form method="POST" onsubmit={submit} novalidate aria-label="Создать аккаунт" aria-busy={busy}>
			<div class="field">
				<label for="email">Email</label>
				<input
					id="email"
					name="email"
					type="email"
					bind:value={email}
					autocomplete="email"
					placeholder="you@example.com"
					required
					maxlength="254"
					disabled={busy || !hydrated}
					aria-invalid={!!errors.email}
					aria-describedby={errors.email ? 'email-error' : undefined}
				/>
				{#if errors.email}<p class="field-error" id="email-error">{errors.email}</p>{/if}
			</div>
			<div class="field">
				<label for="nickname">Никнейм</label>
				<input
					id="nickname"
					name="nickname"
					bind:value={nickname}
					autocomplete="username"
					autocapitalize="none"
					spellcheck="false"
					placeholder="your_name"
					required
					minlength="3"
					maxlength="30"
					disabled={busy || !hydrated}
					aria-invalid={!!errors.nickname}
					aria-describedby={errors.nickname ? 'nickname-error' : 'nickname-hint'}
				/>
				{#if errors.nickname}<p class="field-error" id="nickname-error">{errors.nickname}</p>
				{:else}<p class="field-hint" id="nickname-hint">
						3–30 символов: латиница, цифры, _ и -.
					</p>{/if}
			</div>
			<PasswordField bind:value={password} error={errors.password} disabled={busy || !hydrated} />
			<p class="feedback error-feedback" role="alert">{message}</p>
			<button
				class="button button-primary register-button"
				type="submit"
				disabled={busy || !hydrated}
				>{busy ? 'Создаём аккаунт…' : 'Создать аккаунт'}<Icon name="arrow" /></button
			>
			<p class="registration-note">
				После регистрации подтвердите email.<br />Страницу автора можно настроить следующим шагом.
			</p>
		</form>
	{/if}
</div>

<style>
	.registration {
		max-width: 420px;
		scroll-margin-top: 28px;
	}
	.register-button {
		width: 100%;
		justify-content: space-between;
		padding-inline: 22px;
	}
	.registration-note {
		color: var(--muted);
		font-size: 12px;
		line-height: 1.7;
		margin: 14px 0 0;
	}
	.registration-success {
		padding: 24px;
		border-radius: 16px;
		background: var(--blue-soft);
	}
	.registration-success h2 {
		font-size: 26px;
		margin: 18px 0 12px;
	}
	.registration-success p {
		line-height: 1.7;
		overflow-wrap: anywhere;
	}
	.registration-success .button {
		font-size: 13px;
		max-width: 100%;
	}
	.success-actions {
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: 16px;
		margin-top: 20px;
	}
	.success-icon {
		color: var(--blue);
	}
</style>
