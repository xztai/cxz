<script lang="ts">
	import { goto } from '$app/navigation';
	import Mail from '@lucide/svelte/icons/mail';
	import { api, ApiError, checkError, errorMessage } from '#lib/api/client.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Spinner } from '#lib/components/ui/spinner/index.js';
	import * as Field from '#lib/components/ui/field/index.js';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import AuthCard from './AuthCard.svelte';
	import FormError from './FormError.svelte';
	import PasswordFields from './PasswordFields.svelte';
	import SubmitButton from './SubmitButton.svelte';

	let { mode }: { mode: 'login' | 'register' | 'forgot' } = $props();
	const copy = {
		login: {
			title: 'Продолжим?',
			description: 'Войдите, чтобы вернуться в свой аккаунт.',
			submit: 'Войти'
		},
		register: {
			title: 'Ваше начало здесь.',
			description: 'Один аккаунт для ваших идей и любимых авторов.',
			submit: 'Создать аккаунт'
		},
		forgot: {
			title: 'Вернём вам доступ.',
			description:
				'Введите почту от аккаунта. Отправим ссылку, по которой можно задать новый пароль.',
			submit: 'Отправить ссылку'
		}
	};
	let text = $derived(copy[mode]);
	let email = $state('');
	let nickname = $state('');
	let password = $state('');
	let confirmation = $state('');
	let pending = $state(false);
	let error = $state('');
	let result = $state<'form' | 'verification' | 'recovery'>('form');
	let mailSentAgain = $state(false);
	const errorId = $props.id();

	function backToLogin() {
		result = 'form';
		mailSentAgain = false;
		error = '';
	}

	async function resendConfirmation() {
		if (pending) return;
		pending = true;
		error = '';
		mailSentAgain = false;
		try {
			checkError((await api.POST('/auth/resend-verification', { body: { email } })).error);
			mailSentAgain = true;
		} catch (cause) {
			error = errorMessage(cause);
		} finally {
			pending = false;
		}
	}

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (pending) return;
		error = '';
		if (mode === 'register' && password !== confirmation) {
			error = 'Пароли не совпадают.';
			return;
		}
		pending = true;
		try {
			switch (mode) {
				case 'register':
					checkError(
						(await api.POST('/auth/register', { body: { email, nickname, password } })).error
					);
					result = 'verification';
					break;
				case 'login':
					checkError((await api.POST('/auth/login', { body: { email, password } })).error);
					password = '';
					await goto('/account', { invalidateAll: true });
					return;
				case 'forgot':
					checkError((await api.POST('/auth/forgot-password', { body: { email } })).error);
					result = 'recovery';
			}
			password = '';
			confirmation = '';
		} catch (cause) {
			if (mode === 'login' && cause instanceof ApiError && cause.code === 'email_unverified') {
				result = 'verification';
				password = '';
				return;
			}
			error = errorMessage(cause);
		} finally {
			pending = false;
		}
	}
</script>

<svelte:head
	><title>{text.title} — etheyl</title><meta name="robots" content="noindex" /></svelte:head
>
<AuthCard
	title={result === 'form' ? text.title : 'Проверьте почту.'}
	description={result === 'form'
		? text.description
		: result === 'verification'
			? `Подтвердите адрес ${email}, чтобы войти в аккаунт.`
			: 'Если адрес подходит для этого действия, письмо уже в пути. Загляните также в папку «Спам».'}
>
	{#if result !== 'form'}
		<div class="flex flex-col gap-4">
			<Alert.Root role="status">
				<Mail />
				<Alert.Title>{mailSentAgain ? 'Новое письмо отправлено' : 'Проверьте входящие'}</Alert.Title
				>
				<Alert.Description
					>Откройте ссылку в письме. Если письма пока нет, посмотрите в папке «Спам».</Alert.Description
				>
			</Alert.Root>
			<FormError {error} />
		</div>
	{:else}
		<form method="POST" onsubmit={submit} aria-busy={pending} class="flex flex-col gap-6">
			<Field.FieldGroup>
				{#if mode === 'register'}
					<Field.Field>
						<Field.FieldLabel for="nickname">Никнейм</Field.FieldLabel>
						<Input
							id="nickname"
							name="nickname"
							autocomplete="username"
							bind:value={nickname}
							minlength={3}
							maxlength={30}
							pattern={'[A-Za-z0-9_\\-]{3,30}'}
							title="От 3 до 30 символов: латинские буквы, цифры, _ и -"
							autocapitalize="none"
							spellcheck={false}
							required
							disabled={pending}
							placeholder="your_name"
							aria-describedby="nickname-help"
						/>
						<Field.FieldDescription id="nickname-help"
							>Ваш будущий адрес профиля. 3–30 символов: a–z, 0–9, _ и -. Без пробелов и точек.</Field.FieldDescription
						>
					</Field.Field>
				{/if}
				<Field.Field>
					<Field.FieldLabel for="email">Email</Field.FieldLabel>
					<Input
						id="email"
						name="email"
						type="email"
						autocomplete="email"
						bind:value={email}
						maxlength={254}
						required
						disabled={pending}
						placeholder="you@example.com"
					/>
				</Field.Field>
				{#if mode !== 'forgot'}<PasswordFields
						{mode}
						{pending}
						bind:password
						bind:confirmation
					/>{/if}
			</Field.FieldGroup>
			<FormError {error} id={errorId} />
			<SubmitButton {pending} label={text.submit} errorId={error ? errorId : undefined} />
		</form>
	{/if}
	{#snippet footer()}
		{#if result !== 'form'}
			{#if mode === 'login'}
				<Button onclick={backToLogin} class="w-full" size="lg">Вернуться ко входу</Button>
			{:else}
				<Button href="/auth/login" class="w-full" size="lg">Перейти ко входу</Button>
			{/if}
			{#if result === 'verification'}
				<Button onclick={resendConfirmation} variant="outline" class="w-full" disabled={pending}>
					{#if pending}<Spinner data-icon="inline-start" />{/if}Отправить письмо ещё раз
				</Button>
			{/if}
		{:else if mode === 'login'}
			<p class="text-center text-xs text-muted-foreground">
				Ещё нет аккаунта? <a class="text-link" href="/auth/register">Создать аккаунт</a>
			</p>
		{:else if mode === 'register'}
			<p class="text-center text-xs text-muted-foreground">
				Уже есть аккаунт? <a class="text-link" href="/auth/login">Войти</a>
			</p>
		{:else}
			<Button href="/auth/login" variant="link">Вернуться ко входу</Button>
		{/if}
	{/snippet}
</AuthCard>
