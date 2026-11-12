<script lang="ts">
	import { goto } from '$app/navigation';
	import BookOpen from '@lucide/svelte/icons/book-open';
	import PenLine from '@lucide/svelte/icons/pen-line';
	import CircleCheck from '@lucide/svelte/icons/circle-check';
	import { api, checkError, errorMessage } from '#lib/api/client.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Spinner } from '#lib/components/ui/spinner/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Tabs from '#lib/components/ui/tabs/index.js';
	import * as Empty from '#lib/components/ui/empty/index.js';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import LineRule from '#lib/components/LineRule.svelte';
	import type { PageProps } from './$types';
	let { data }: PageProps = $props();
	let pending = $state(false);
	let error = $state('');
	async function logout() {
		if (pending) return;
		pending = true;
		error = '';
		try {
			checkError((await api.POST('/auth/logout')).error);
			await goto('/auth/login', { invalidateAll: true });
		} catch (cause) {
			error = errorMessage(cause);
		} finally {
			pending = false;
		}
	}
</script>

<svelte:head
	><title>Личный кабинет — etheyl</title><meta name="robots" content="noindex" /></svelte:head
>
<section class="workspace-page page-inset">
	<div class="workspace-heading">
		<div>
			<h1>Здравствуйте, {data.user.nickname}.</h1>
			<p>Всё начинается с вас. Ваше личное пространство.</p>
		</div>
		<Button variant="outline" onclick={logout} disabled={pending}
			>{#if pending}<Spinner data-icon="inline-start" />{/if}Выйти</Button
		>
	</div>
	<LineRule />
	{#if error}
		<Alert.Root variant="destructive" role="alert"
			><Alert.Title>Не удалось выйти</Alert.Title><Alert.Description>{error}</Alert.Description
			></Alert.Root
		>
	{/if}
	<div class="workspace-grid">
		<Card.Root>
			<Card.Header
				><Card.Title><h2>Ваш аккаунт</h2></Card.Title><Card.Description
					>Один аккаунт для всех ваших интересов.</Card.Description
				></Card.Header
			>
			<Card.Content>
				<dl class="account-details">
					<div>
						<dt>Никнейм</dt>
						<dd>{data.user.nickname}</dd>
					</div>
					<div>
						<dt>Email</dt>
						<dd>{data.user.email}</dd>
					</div>
				</dl>
				<div class="mt-6">
					<Badge variant="outline"
						><span class="verified-badge"><CircleCheck aria-hidden="true" />Email подтверждён</span
						></Badge
					>
				</div>
			</Card.Content>
			<Card.Footer class="flex flex-col items-start gap-2"
				><Button href="/account/profile" variant="outline">Профиль и настройки</Button><Button
					href="/auth/forgot"
					variant="link">Изменить пароль через почту</Button
				></Card.Footer
			>
		</Card.Root>
		<div class="workspace-content">
			<Tabs.Root value="reader" class="flex flex-col gap-6">
				<Tabs.List variant="line" aria-label="Режим кабинета">
					<Tabs.Trigger value="reader">Я подписчик</Tabs.Trigger><Tabs.Trigger value="creator"
						>Я автор</Tabs.Trigger
					>
				</Tabs.List>
				<Tabs.Content value="reader">
					<Empty.Root>
						<Empty.Header>
							<Empty.Media variant="icon"><BookOpen aria-hidden="true" /></Empty.Media>
							<Empty.Title><h2>Здесь будут ваши истории</h2></Empty.Title>
							<Empty.Description
								>Находите интересных людей в каталоге авторов. Оформление подписок появится позже.</Empty.Description
							>
						</Empty.Header>
						<Empty.Content
							><Button href="/authors" variant="outline">Найти автора</Button></Empty.Content
						>
					</Empty.Root>
				</Tabs.Content>
				<Tabs.Content value="creator">
					<Empty.Root>
						<Empty.Header>
							<Empty.Media variant="icon"><PenLine aria-hidden="true" /></Empty.Media>
							<Empty.Title><h2>Место для вашего голоса</h2></Empty.Title>
							<Empty.Description
								>Оформите публичную страницу и создайте уровни подписки. Ваш профиль появится в
								каталоге авторов.</Empty.Description
							>
						</Empty.Header>
						<Empty.Content
							><Button href="/account/profile">Оформить профиль</Button><Button
								href="/account/tiers"
								variant="outline">Уровни подписки</Button
							></Empty.Content
						>
					</Empty.Root>
				</Tabs.Content>
			</Tabs.Root>
		</div>
	</div>
	{#if data.user.isAdmin}
		<section class="admin-access" aria-labelledby="admin-access-title">
			<div>
				<h2 id="admin-access-title">Доступ администратора</h2>
				<p>Ваш аккаунт создан через CLI и имеет права администратора.</p>
			</div>
			<Button href="/admin" variant="outline">Кабинет администратора</Button>
		</section>
	{/if}
</section>
