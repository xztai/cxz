<script lang="ts">
	import Eye from '@lucide/svelte/icons/eye';
	import EyeOff from '@lucide/svelte/icons/eye-off';
	import { Input } from '#lib/components/ui/input/index.js';
	import * as Field from '#lib/components/ui/field/index.js';
	let {
		mode,
		pending,
		password = $bindable(''),
		confirmation = $bindable('')
	}: {
		mode: 'login' | 'register' | 'reset';
		pending: boolean;
		password: string;
		confirmation?: string;
	} = $props();
	let visible = $state(false);
</script>

<Field.Field>
	<div class="password-label-row">
		<Field.FieldLabel for="password"
			>{mode === 'reset' ? 'Новый пароль' : 'Пароль'}</Field.FieldLabel
		>
		{#if mode === 'login'}<a href="/auth/forgot" class="forgot-link">Не помню пароль</a>{/if}
	</div>
	<div class="password-control">
		<Input
			id="password"
			name="password"
			type={visible ? 'text' : 'password'}
			autocomplete={mode === 'login' ? 'current-password' : 'new-password'}
			bind:value={password}
			minlength={mode === 'login' ? 1 : 12}
			maxlength={256}
			required
			disabled={pending}
			aria-describedby={mode === 'login' ? undefined : 'password-help'}
		/>
		<button
			type="button"
			class="password-toggle"
			onclick={() => (visible = !visible)}
			aria-label={visible ? 'Скрыть пароль' : 'Показать пароль'}
			aria-pressed={visible}
			disabled={pending}
			>{#if visible}<EyeOff aria-hidden="true" />{:else}<Eye aria-hidden="true" />{/if}</button
		>
	</div>
	{#if mode !== 'login'}<Field.FieldDescription id="password-help"
			>Не менее 12 символов. Пробелы и кириллица тоже подойдут.</Field.FieldDescription
		>{/if}
</Field.Field>
{#if mode !== 'login'}
	<Field.Field data-invalid={(confirmation !== '' && password !== confirmation) || undefined}>
		<Field.FieldLabel for="confirmation">Повторите пароль</Field.FieldLabel>
		<Input
			id="confirmation"
			name="confirmation"
			type={visible ? 'text' : 'password'}
			autocomplete="new-password"
			bind:value={confirmation}
			required
			disabled={pending}
			aria-invalid={(confirmation !== '' && password !== confirmation) || undefined}
			aria-describedby={confirmation !== '' && password !== confirmation
				? 'confirmation-help'
				: undefined}
		/>
		{#if confirmation !== '' && password !== confirmation}<Field.FieldDescription
				id="confirmation-help">Пароли должны совпадать.</Field.FieldDescription
			>{/if}
	</Field.Field>
{/if}
