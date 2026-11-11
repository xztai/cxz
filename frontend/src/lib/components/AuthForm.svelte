<script lang="ts">
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import CredentialsForm from './auth/CredentialsForm.svelte';
	import TokenForm from './auth/TokenForm.svelte';
	import WingMark from './WingMark.svelte';
	let { mode }: { mode: 'login' | 'register' | 'forgot' | 'verify' | 'reset' } = $props();
</script>

<section class="auth-layout page-inset">
	<a href="/" class="back-link"><ArrowLeft aria-hidden="true" />На главную</a>
	{#if mode === 'verify' || mode === 'reset'}
		<TokenForm {mode} />
	{:else}
		<CredentialsForm {mode} />
	{/if}
	<p class="auth-note">
		<WingMark /><span>Ваше пространство. Ваш ритм.<br />Ближе к тому, что любите.</span>
	</p>
</section>
