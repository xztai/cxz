<script lang="ts">
	import './layout.css';
	import { afterNavigate, invalidate } from '$app/navigation';
	import { studioAccountFailure } from '#lib/api/client.js';
	import '@fontsource-variable/manrope';
	import favicon from '#lib/assets/favicon.svg';
	import type { LayoutProps } from './$types';
	import { onMount } from 'svelte';
	import { Session, setSession } from '#lib/session.svelte.js';

	let { children }: LayoutProps = $props();
	const session = new Session(() => invalidate('studio:profile'));
	setSession(session);
	onMount(() => {
		const recoverStudio = (event: Event) => {
			const ownerId = (event as CustomEvent<string | null>).detail ?? session.user?.id;
			if (ownerId !== undefined) session.recheckStudio(ownerId);
		};
		window.addEventListener(studioAccountFailure, recoverStudio);
		void session.refresh();
		return () => window.removeEventListener(studioAccountFailure, recoverStudio);
	});
	afterNavigate(({ type }) => {
		if (type === 'popstate') void session.refresh();
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
