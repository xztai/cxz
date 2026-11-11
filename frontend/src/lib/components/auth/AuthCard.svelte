<script lang="ts">
	import type { Snippet } from 'svelte';
	import * as Card from '#lib/components/ui/card/index.js';

	let {
		title,
		description,
		children,
		footer,
		heading
	}: {
		title: string;
		description: string;
		children: Snippet;
		footer: Snippet;
		heading?: Snippet;
	} = $props();
</script>

<Card.Root class="auth-card w-full">
	<Card.Header>
		<Card.Title><h1>{title}</h1></Card.Title>
		<Card.Description>{description}</Card.Description>
		{@render heading?.()}
	</Card.Header>
	<Card.Content>{@render children()}</Card.Content>
	<Card.Footer class="flex-col gap-3">{@render footer()}</Card.Footer>
</Card.Root>
