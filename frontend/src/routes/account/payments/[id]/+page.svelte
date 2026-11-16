<script lang="ts">
	import PaymentStatus from '#lib/components/billing/PaymentStatus.svelte';
	import type { PageProps } from './$types';
	let { data }: PageProps = $props();
</script>

{#key data.payment.id}<PaymentStatus initial={data.payment} />{/key}
