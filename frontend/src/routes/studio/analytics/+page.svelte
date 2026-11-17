<script lang="ts">
	import AnalyticsMetrics from '../AnalyticsMetrics.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
</script>

<svelte:head><title>Аналитика — etheyl</title></svelte:head>

<AnalyticsMetrics analytics={data.analytics} dependency="studio:analytics" heading="h1" />
