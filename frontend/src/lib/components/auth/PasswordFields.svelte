<script lang="ts">
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
</script>

<Field.Field>
	<Field.FieldLabel for="password">{mode === 'reset' ? 'Новый пароль' : 'Пароль'}</Field.FieldLabel>
	<Input
		id="password"
		name="password"
		type="password"
		autocomplete={mode === 'login' ? 'current-password' : 'new-password'}
		bind:value={password}
		minlength={mode === 'login' ? 1 : 12}
		maxlength={256}
		required
		disabled={pending}
		aria-describedby={mode === 'login' ? undefined : 'password-help'}
	/>
	{#if mode !== 'login'}<Field.FieldDescription id="password-help"
			>От 12 символов. Можно использовать пробелы и кириллицу.</Field.FieldDescription
		>{/if}
</Field.Field>
{#if mode !== 'login'}
	<Field.Field data-invalid={(confirmation !== '' && password !== confirmation) || undefined}
		><Field.FieldLabel for="confirmation">Повторите пароль</Field.FieldLabel><Input
			id="confirmation"
			name="confirmation"
			type="password"
			autocomplete="new-password"
			bind:value={confirmation}
			required
			disabled={pending}
			aria-invalid={(confirmation !== '' && password !== confirmation) || undefined}
		/></Field.Field
	>
{/if}
