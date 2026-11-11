<script lang="ts">
	import { page } from '$app/state';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Empty from '#lib/components/ui/empty/index.js';
	let isMissingPage = $derived(page.error?.status === 404);
</script>

<svelte:head
	><title>{isMissingPage ? 'Страница не найдена' : 'Не получилось'} — etheyl</title></svelte:head
>
<section class="error-page page-inset">
	<Empty.Root class="mx-auto max-w-lg"
		><Empty.Header
			><Empty.Title><h1>{page.error?.status ?? 500}</h1></Empty.Title><Empty.Description
				>{isMissingPage
					? 'Страница не найдена. Проверьте адрес или вернитесь на главную.'
					: (page.error?.message ?? 'Произошла ошибка. Попробуйте позже.')}</Empty.Description
			></Empty.Header
		><Empty.Content
			><Button href="/"><ArrowLeft data-icon="inline-start" />На главную</Button></Empty.Content
		></Empty.Root
	>
</section>
