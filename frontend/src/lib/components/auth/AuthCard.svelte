<script lang="ts">
	import type { Snippet } from 'svelte';
	let {
		title,
		description,
		children,
		footer,
		heading
	}: { title: string; description: string; children: Snippet; footer: Snippet; heading?: Snippet } =
		$props();
</script>

<div class="auth-card">
	<header class="auth-card-heading">
		<h1>{title}</h1>
		<p>{description}</p>
		{@render heading?.()}
	</header>
	<div class="auth-card-content">{@render children()}</div>
	<footer class="auth-card-footer">{@render footer()}</footer>
</div>
