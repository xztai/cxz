<script lang="ts">
	import Icon from './Icon.svelte';
	let {
		value = $bindable(''),
		error = '',
		autocomplete = 'new-password',
		disabled = false,
		hint = true
	}: {
		value?: string;
		error?: string;
		autocomplete?: 'new-password' | 'current-password';
		disabled?: boolean;
		hint?: boolean;
	} = $props();
	let visible = $state(false);
</script>

<div class="field">
	<label for="password">Пароль</label>
	<div class="password-wrap">
		<input
			id="password"
			name="password"
			type={visible ? 'text' : 'password'}
			bind:value
			{autocomplete}
			{disabled}
			required
			aria-invalid={!!error}
			aria-describedby={error ? 'password-error' : hint ? 'password-hint' : undefined}
		/>
		<button
			class="password-toggle"
			type="button"
			onclick={() => (visible = !visible)}
			aria-label={visible ? 'Скрыть пароль' : 'Показать пароль'}
			aria-pressed={visible}
			{disabled}><Icon name={visible ? 'eye-off' : 'eye'} /></button
		>
	</div>
	{#if error}<p class="field-error" id="password-error">{error}</p>
	{:else if hint}<p class="field-hint" id="password-hint">
			Не менее 12 символов. Максимум 256 байт.
		</p>{/if}
</div>

<style>
	.password-wrap {
		position: relative;
	}
	input {
		padding-right: 52px;
	}
	.password-toggle {
		position: absolute;
		right: 4px;
		top: 4px;
		width: 44px;
		height: 42px;
		border: 0;
		display: grid;
		place-items: center;
		border-radius: 8px;
		background: transparent;
		color: var(--muted);
		cursor: pointer;
	}
	.password-toggle:hover {
		color: var(--blue);
		background: var(--blue-soft);
	}
</style>
