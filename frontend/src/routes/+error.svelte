<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';

	const descriptions: Partial<Record<number, string>> = {
		403: 'Доступ запрещён',
		404: 'Страница не найдена'
	};
	const serverError = $derived(page.status >= 500 && page.status < 600);
	const description = $derived(
		descriptions[page.status] ?? (serverError ? 'Ошибка сервера' : 'Не удалось открыть страницу')
	);
	const retryHref = $derived(`./${page.url.pathname.split('/').at(-1)}${page.url.search}`);
</script>

<svelte:head>
	<title>{page.status} — {description} — etheyl</title>
</svelte:head>

<main class="error-page">
	<div>
		<h1>
			<span class="status-code">{page.status}</span>
			{description}
		</h1>
		<div class="actions">
			<a class="button button-primary" href={resolve('/')}>На главную</a>
			{#if serverError}
				<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- Retry relative to the current document URL. -->
				<a class="button button-outline" href={retryHref} data-sveltekit-reload>Повторить</a>
			{/if}
		</div>
	</div>
</main>

<style>
	.error-page {
		display: grid;
		min-height: 100svh;
		place-items: center;
		padding: 40px 20px;
		background: var(--surface);
		text-align: center;
	}

	h1 {
		margin: 0;
		font-size: 1.5rem;
		font-weight: 800;
		line-height: 1.35;
	}

	.status-code {
		display: block;
		margin-bottom: 12px;
		font-size: 6rem;
		font-variant-numeric: tabular-nums;
		letter-spacing: -0.04em;
		line-height: 1;
	}

	.actions {
		display: flex;
		flex-wrap: wrap;
		justify-content: center;
		gap: 12px;
		margin-top: 28px;
	}

	@media (max-width: 640px) {
		h1 {
			font-size: 1.25rem;
		}

		.status-code {
			font-size: 5rem;
		}
	}
</style>
