<script lang="ts">
	import { afterNavigate, goto, replaceState } from '$app/navigation';
	import { page } from '$app/state';
	import CircleCheck from '@lucide/svelte/icons/circle-check';
	import { api, checkError, errorMessage } from '#lib/api/client.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Field from '#lib/components/ui/field/index.js';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import AuthCard from './AuthCard.svelte';
	import FormError from './FormError.svelte';
	import PasswordFields from './PasswordFields.svelte';
	import SubmitButton from './SubmitButton.svelte';

	let { mode }: { mode: 'verify' | 'reset' } = $props();
	const copy = {
		verify: {
			title: 'Это ваша почта?',
			description: 'Подтвердите адрес ниже. После этого откроется ваш аккаунт.',
			submit: 'Подтвердить email'
		},
		reset: {
			title: 'Новый пароль — и вы дома.',
			description: 'Выберите пароль, который ещё не использовали для этого аккаунта.',
			submit: 'Сохранить пароль'
		}
	};
	let text = $derived(copy[mode]);
	let password = $state('');
	let confirmation = $state('');
	let token = $state('');
	let pending = $state(false);
	let ready = $state(false);
	let error = $state('');
	let done = $state(false);
	let verificationEmail = $state('');
	let checkingToken = $state(false);
	const errorId = $props.id();

	async function previewVerification(value: string) {
		checkingToken = true;
		verificationEmail = '';
		try {
			const result = await api.POST('/auth/verify-email/preview', { body: { token: value } });
			if (token !== value) return;
			checkError(result.error);
			verificationEmail = result.data?.email ?? '';
		} catch (cause) {
			if (token === value) error = errorMessage(cause);
		} finally {
			if (token === value) checkingToken = false;
		}
	}

	function readToken() {
		const fragment = new URLSearchParams(window.location.hash.slice(1));
		if (fragment.has('token')) {
			token = fragment.get('token') ?? '';
			verificationEmail = '';
			checkingToken = false;
			replaceState(window.location.pathname, page.state);
			error = '';
			done = false;
			if (mode === 'verify' && token) void previewVerification(token);
		}
		ready = true;
	}
	afterNavigate(readToken);

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (pending || checkingToken || !token || (mode === 'verify' && !verificationEmail)) return;
		error = '';
		if (mode === 'reset' && password !== confirmation) {
			error = 'Пароли не совпадают.';
			return;
		}
		pending = true;
		try {
			if (mode === 'verify') {
				checkError((await api.POST('/auth/verify-email', { body: { token } })).error);
				token = '';
				await goto('/account', { invalidateAll: true });
				return;
			}
			checkError((await api.POST('/auth/reset-password', { body: { token, password } })).error);
			token = '';
			password = '';
			confirmation = '';
			done = true;
		} catch (cause) {
			error = errorMessage(cause);
		} finally {
			pending = false;
		}
	}
</script>

<svelte:window onhashchange={readToken} />
<svelte:head
	><title>{text.title} — etheyl</title><meta name="robots" content="noindex" /></svelte:head
>
<AuthCard
	title={done ? 'Доступ восстановлен.' : text.title}
	description={done
		? 'Пароль изменён. Все предыдущие сеансы завершены.'
		: checkingToken
			? 'Проверяем ссылку…'
			: text.description}
>
	{#snippet heading()}
		{#if mode === 'verify' && verificationEmail}<p
				class="break-all"
				data-testid="verification-email"
			>
				{verificationEmail}
			</p>{/if}
	{/snippet}
	{#if done}
		<Alert.Root role="status">
			<CircleCheck />
			<Alert.Title>Можно войти в аккаунт</Alert.Title>
			<Alert.Description>Используйте email и пароль на странице входа.</Alert.Description>
		</Alert.Root>
	{:else}
		<form
			method="POST"
			onsubmit={submit}
			aria-busy={pending || checkingToken}
			class="flex flex-col gap-6"
		>
			{#if mode === 'reset'}<Field.FieldGroup
					><PasswordFields {mode} {pending} bind:password bind:confirmation /></Field.FieldGroup
				>{/if}
			{#if ready && !token}<Alert.Root variant="destructive">
					<Alert.Title>В ссылке нет кода</Alert.Title>
					<Alert.Description
						>Откройте полную ссылку из письма или запросите новую.</Alert.Description
					>
				</Alert.Root>{/if}
			<FormError {error} id={errorId} />
			<SubmitButton
				pending={pending || checkingToken}
				disabled={!ready || !token || (mode === 'verify' && !verificationEmail)}
				label={text.submit}
				errorId={error ? errorId : undefined}
			/>
		</form>
	{/if}
	{#snippet footer()}
		{#if done}<Button href="/auth/login" class="w-full" size="lg">Перейти ко входу</Button>
		{:else if mode === 'reset'}<Button href="/auth/forgot" variant="link"
				>Запросить новую ссылку</Button
			>
		{:else}<Button href="/auth/login" variant="link">Вернуться ко входу</Button>{/if}
	{/snippet}
</AuthCard>
