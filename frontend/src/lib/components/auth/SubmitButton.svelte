<script lang="ts">
	import ArrowRight from '@lucide/svelte/icons/arrow-right';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Spinner } from '#lib/components/ui/spinner/index.js';

	let {
		pending,
		disabled = false,
		errorId,
		label
	}: {
		pending: boolean;
		disabled?: boolean;
		errorId?: string;
		label: string;
	} = $props();
</script>

<Button
	type="submit"
	size="lg"
	class="w-full"
	disabled={pending || disabled}
	aria-describedby={errorId}
>
	{#if pending}<Spinner data-icon="inline-start" />Подождите…{:else}{label}<ArrowRight
			data-icon="inline-end"
		/>{/if}
</Button>
