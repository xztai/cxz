<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { resolve } from '$app/paths';
	import { goto } from '$app/navigation';
	import { getSession } from '#lib/session.svelte.js';
	import { api, ApiError, errorMessage } from '#lib/api/client.js';
	import { emailError, nicknameError, passwordError } from '#lib/validation.js';
	import PasswordField from './PasswordField.svelte';
	import Icon from './Icon.svelte';
	const session = getSession();
	let email = $state('');
	let nickname = $state('');
	let password = $state('');
	let errors = $state({ email: '', nickname: '', password: '' });
	let busy = $state(false);
	let hydrated = $state(false);
	onMount(() => {
		hydrated = true;
	});
	let message = $state('');
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
			session.setVerification(email, true);
			await goto(resolve('/auth/check-email'));
		} catch (error) {
			message = errorMessage(error);
			if (error instanceof ApiError && error.code === 'email_exists') errors.email = message;
			if (error instanceof ApiError && error.code === 'nickname_exists') errors.nickname = message;
		} finally {
			busy = false;
		}
	}
</script>

<div id="registration" class="registration">
	<noscript><p class="feedback">Для регистрации включите JavaScript в браузере.</p></noscript>
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
		<button class="button button-primary register-button" type="submit" disabled={busy || !hydrated}
			>{busy ? 'Создаём аккаунт…' : 'Создать аккаунт'}<Icon name="arrow" /></button
		>
		<p class="registration-note">
			После регистрации подтвердите email.<br />Страницу автора можно настроить следующим шагом.
		</p>
	</form>
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
</style>
