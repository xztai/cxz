<script lang="ts">
	import { untrack } from 'svelte';
	import { invalidateAll } from '$app/navigation';
	import { api, checkError, errorMessage, ApiError } from '#lib/api/client.js';
	import { requestWithTimeout } from '#lib/api/request.js';
	import type { components } from '#lib/api/schema.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Textarea } from '#lib/components/ui/textarea/index.js';
	import { Spinner } from '#lib/components/ui/spinner/index.js';
	import * as Field from '#lib/components/ui/field/index.js';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import type { PageProps } from './$types';
	let { data }: PageProps = $props();
	let profile = $state.raw(untrack(() => data.profile));
	let nickname = $state(untrack(() => data.profile?.nickname ?? data.user.nickname));
	let displayName = $state(untrack(() => data.profile?.displayName ?? ''));
	let description = $state(untrack(() => data.profile?.description ?? ''));
	let pending = $state('');
	let failure = $state('');
	let message = $state('');
	let imageFeedback = $state<{ kind: 'avatar' | 'cover'; error: string; message: string } | null>(
		null
	);
	async function save(event: SubmitEvent) {
		event.preventDefault();
		if (pending) return;
		pending = 'profile';
		failure = '';
		message = '';
		try {
			const result = await api.PUT('/me/profile', { body: { nickname, displayName, description } });
			checkError(result.error);
			if (!result.data) throw new Error('Missing profile');
			profile = result.data.profile;
			nickname = profile.nickname;
			displayName = profile.displayName;
			description = profile.description;
			await invalidateAll();
			message = 'Профиль сохранён и доступен в каталоге.';
		} catch (cause) {
			failure = errorMessage(cause);
		} finally {
			pending = '';
		}
	}
	async function upload(event: Event, kind: 'avatar' | 'cover') {
		const input = event.currentTarget as HTMLInputElement;
		const file = input.files?.[0];
		if (!file || pending) return;
		failure = '';
		message = '';
		imageFeedback = null;
		if (file.size > 5 * 1024 * 1024) {
			imageFeedback = { kind, error: 'Размер изображения не должен превышать 5 МБ.', message: '' };
			input.value = '';
			return;
		}
		pending = kind;
		try {
			const csrf = await api.GET('/auth/csrf');
			checkError(csrf.error);
			if (!csrf.data) throw new Error('Missing CSRF token');
			const response = await requestWithTimeout(
				new Request(`/api/v1/me/profile/images/${kind}`, {
					method: 'PUT',
					credentials: 'same-origin',
					headers: { 'Content-Type': file.type, 'X-CSRF-Token': csrf.data.token },
					body: file
				})
			);
			const result = (await response.json()) as {
				profile: components['schemas']['Profile'];
			} & Partial<components['schemas']['Error']>;
			if (!response.ok)
				throw new ApiError(
					result.error?.code ?? 'upload',
					result.error?.message ?? 'Не удалось загрузить изображение.'
				);
			profile = result.profile;
			imageFeedback = { kind, error: '', message: 'Изображение сохранено.' };
		} catch (cause) {
			imageFeedback = { kind, error: errorMessage(cause), message: '' };
		} finally {
			pending = '';
			input.value = '';
		}
	}
	async function remove(kind: 'avatar' | 'cover') {
		if (pending) return;
		pending = kind;
		failure = '';
		message = '';
		imageFeedback = null;
		try {
			const result = await api.DELETE('/me/profile/images/{kind}', { params: { path: { kind } } });
			checkError(result.error);
			if (result.data) profile = result.data.profile;
			imageFeedback = { kind, error: '', message: 'Изображение удалено.' };
		} catch (cause) {
			imageFeedback = { kind, error: errorMessage(cause), message: '' };
		} finally {
			pending = '';
		}
	}
</script>

<svelte:head
	><title>Профиль и настройки — etheyl</title><meta name="robots" content="noindex" /></svelte:head
>
<section class="creator-workspace page-inset">
	<a href="/account" class="back-link">← Личный кабинет</a>
	<div class="workspace-heading">
		<div>
			<h1>Профиль и настройки</h1>
			<p>Расскажите о себе. Это увидят ваши будущие подписчики.</p>
		</div>
		{#if profile}<Button href={`/authors/${profile.nickname}`} variant="outline"
				>Открыть страницу</Button
			>{/if}
	</div>
	<nav class="creator-nav" aria-label="Настройки автора">
		<a href="/account/profile" aria-current="page">Профиль</a>
		<a href="/account/tiers">Уровни подписки</a>
		<a href="/account/posts">Публикации</a>
	</nav>
	{#if failure}<Alert.Root variant="destructive" role="alert"
			><Alert.Title>Не удалось сохранить</Alert.Title><Alert.Description
				>{failure}</Alert.Description
			></Alert.Root
		>{/if}
	{#if message}<p class="save-message" role="status">{message}</p>{/if}
	<div class="settings-grid">
		<form method="POST" onsubmit={save} class="creator-form">
			<h2>Как вас представлять</h2>
			<Field.FieldGroup>
				<Field.Field
					><Field.FieldLabel for="nickname">Уникальный адрес</Field.FieldLabel><Input
						id="nickname"
						name="nickname"
						bind:value={nickname}
						required
						minlength={3}
						maxlength={30}
						pattern="[A-Za-z0-9_\-]+"
						disabled={!!pending}
						aria-describedby="nickname-help"
					/><Field.FieldDescription id="nickname-help"
						>/authors/{nickname || 'ваш_никнейм'} · 3–30 латинских букв, цифр, _ или -. Смена адреса изменит
						никнейм аккаунта; прежняя ссылка перестанет работать.</Field.FieldDescription
					></Field.Field
				>
				<Field.Field
					><Field.FieldLabel for="displayName">Имя автора</Field.FieldLabel><Input
						id="displayName"
						name="displayName"
						bind:value={displayName}
						required
						maxlength={80}
						disabled={!!pending}
						placeholder="Как вас знает ваша аудитория"
					/></Field.Field
				>
				<Field.Field
					><Field.FieldLabel for="description">Описание</Field.FieldLabel><Textarea
						id="description"
						name="description"
						bind:value={description}
						maxlength={5000}
						rows={7}
						disabled={!!pending}
						placeholder="Что вы создаёте и о чём хотите рассказывать"
					/></Field.Field
				>
				<Button type="submit" disabled={!!pending}
					>{#if pending === 'profile'}<Spinner data-icon="inline-start" />{/if}Сохранить профиль</Button
				>
			</Field.FieldGroup>
			<p class="form-note">После сохранения профиль появится в каталоге авторов.</p>
			<div class="account-settings-note">
				<h2>Доступ к аккаунту</h2>
				<p>{data.user.email}</p>
				<Button href="/auth/forgot" variant="link">Изменить пароль через почту</Button>
			</div>
		</form>
		<section class="profile-images" aria-labelledby="images-title">
			<h2 id="images-title">Аватар и обложка</h2>
			<p class="form-note">
				JPEG, PNG или WebP до 5 МБ. До 8192 пикселей по стороне и 16 Мп. Для обложки подойдёт
				горизонтальный кадр.
			</p>
			{#if !profile}<p class="form-note">
					Сначала сохраните профиль, затем добавьте изображения.
				</p>{/if}
			{#each ['avatar', 'cover'] as value (value)}
				{@const kind = value as 'avatar' | 'cover'}
				<div class="image-setting">
					{#if profile?.[kind]}<img
							class={{ 'avatar-preview': kind === 'avatar', 'cover-preview': kind === 'cover' }}
							src={profile[kind].url}
							alt={kind === 'avatar' ? 'Ваш аватар' : 'Ваша обложка'}
						/>{/if}
					<Field.Field
						data-invalid={(imageFeedback?.kind === kind && !!imageFeedback.error) || undefined}
						><Field.FieldLabel for={`image-${kind}`}
							>{kind === 'avatar' ? 'Аватар' : 'Обложка'}</Field.FieldLabel
						><Input
							id={`image-${kind}`}
							aria-invalid={(imageFeedback?.kind === kind && !!imageFeedback.error) || undefined}
							aria-describedby={imageFeedback?.kind === kind ? `image-feedback-${kind}` : undefined}
							type="file"
							accept="image/png,image/jpeg,image/webp"
							disabled={!profile || !!pending}
							onchange={(event) => upload(event, kind)}
						/>
						{#if imageFeedback?.kind === kind}
							{#if imageFeedback.error}<Field.FieldError id={`image-feedback-${kind}`}
									>{imageFeedback.error}</Field.FieldError
								>
							{:else}<Field.FieldDescription id={`image-feedback-${kind}`} role="status"
									>{imageFeedback.message}</Field.FieldDescription
								>{/if}
						{/if}
					</Field.Field>
					{#if pending === kind}<p role="status" class="flex items-center gap-2">
							<Spinner />Сохраняем изображение…
						</p>{/if}
					{#if profile?.[kind]}<Button
							variant="ghost"
							onclick={() => remove(kind)}
							disabled={!!pending}>Удалить {kind === 'avatar' ? 'аватар' : 'обложку'}</Button
						>{/if}
				</div>
			{/each}
		</section>
	</div>
</section>
