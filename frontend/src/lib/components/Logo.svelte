<script lang="ts">
	import WingMark from './WingMark.svelte';
	let { compact = false }: { compact?: boolean } = $props();
</script>

<a href="/" aria-label="etheyl — на главную" class="brand inline-flex items-center gap-2.5">
	<WingMark />
	{#if !compact}<span>etheyl</span>{/if}
</a>
