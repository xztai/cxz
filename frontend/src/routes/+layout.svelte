<script lang="ts">
	import './layout.css';
	import '@fontsource-variable/manrope';
	import favicon from '#lib/assets/favicon.svg';
	import type { LayoutProps } from './$types';
	import { onMount } from 'svelte';
	import { Session, setSession } from '#lib/session.svelte.js';

	let { children }: LayoutProps = $props();
	const session = new Session();
	setSession(session);
	onMount(() => {
		void session.refresh();
	});
</script>

<svelte:window
	onfocus={() => void session.refresh()}
	onpageshow={(event) => {
		if (event.persisted) void session.refresh();
	}}
/>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

{@render children()}
