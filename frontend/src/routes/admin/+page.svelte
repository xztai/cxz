<script lang="ts">
	import CircleCheck from '@lucide/svelte/icons/circle-check';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import * as Card from '#lib/components/ui/card/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import LineRule from '#lib/components/LineRule.svelte';
</script>

<svelte:head
	><title>Кабинет администратора — etheyl</title><meta
		name="robots"
		content="noindex"
	/></svelte:head
>
<section class="workspace-page admin-page page-inset">
	<a class="back-link" href="/account"><ArrowLeft aria-hidden="true" />В личный кабинет</a>
	<div class="workspace-heading">
		<div>
			<h1>Кабинет администратора</h1>
			<p>etheyl / администрация</p>
		</div>
	</div>
	<LineRule />
	<Card.Root>
		<Card.Header>
			<Card.Title
				><h2 class="flex items-center gap-2">
					<CircleCheck aria-hidden="true" />Доступ подтверждён
				</h2></Card.Title
			>
			<Card.Description
				>Сервер проверил ваши права. Очередь жалоб, модерация и аудит появятся на шестом этапе MVP.</Card.Description
			>
		</Card.Header>
		<Card.Footer><Button href="/account" variant="outline">Вернуться в аккаунт</Button></Card.Footer
		>
	</Card.Root>
</section>
