<script lang="ts">
	import * as Alert from '#lib/components/ui/alert/index.js';

	let { error, id }: { error: string; id?: string } = $props();
</script>

{#if error}
	<Alert.Root variant="destructive" {id} role="alert">
		<Alert.Title>Не получилось</Alert.Title>
		<Alert.Description>{error}</Alert.Description>
	</Alert.Root>
{/if}
