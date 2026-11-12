<script lang="ts">
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import CredentialsForm from './auth/CredentialsForm.svelte';
	import TokenForm from './auth/TokenForm.svelte';
	import CreatorCollage from './CreatorCollage.svelte';
	let { mode }: { mode: 'login' | 'register' | 'forgot' | 'verify' | 'reset' } = $props();
</script>

<section class="auth-layout">
	<aside class="auth-story" aria-label="О etheyl">
		<div>
			<h2>У всего, что вы<br />делаете, может<br />быть <span>продолжение.</span></h2>
			<p>И люди, которые его ждут.</p>
		</div>
		<CreatorCollage compact />
		<p class="auth-story-foot">Один аккаунт. Создавайте и поддерживайте.</p>
	</aside>
	<div class="auth-main">
		<div class="auth-form-wrap">
			<a href="/" class="back-link"><ArrowLeft aria-hidden="true" />На главную</a>
			{#if mode === 'login' || mode === 'register'}<nav
					class="auth-switch"
					aria-label="Доступ к аккаунту"
				>
					<a href="/auth/login" aria-current={mode === 'login' ? 'page' : undefined}>Вход</a><a
						href="/auth/register"
						aria-current={mode === 'register' ? 'page' : undefined}>Регистрация</a
					>
				</nav>{/if}
			{#if mode === 'verify' || mode === 'reset'}<TokenForm {mode} />{:else}<CredentialsForm
					{mode}
				/>{/if}
		</div>
	</div>
</section>
