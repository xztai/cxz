<script lang="ts">
	import { goto, invalidate } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { tick, untrack } from 'svelte';
	import {
		ApiError,
		api,
		errorMessage,
		type Profile,
		type ProfileImageKind,
		type ProfileInput
	} from '#lib/api/client.js';
	import AuthorProfileHeader from '#lib/components/AuthorProfileHeader.svelte';
	import Icon from '#lib/components/Icon.svelte';
	import { getSession, studioEditorIsActive } from '#lib/session.svelte.js';
	import ProfileImageCrop from './ProfileImageCrop.svelte';
	import type { PageProps } from './$types';

	const imageTypes = ['image/jpeg', 'image/png', 'image/webp'];
	const maxImageBytes = 5 * 1024 * 1024;
	const imageKinds = ['cover', 'avatar'] as const;
	const imageCopy = {
		cover: {
			title: 'Обложка',
			empty: 'Обложка не загружена',
			saved: 'Обложка обновлена.',
			removed: 'Обложка удалена.'
		},
		avatar: {
			title: 'Аватар',
			empty: 'Аватар не загружен',
			saved: 'Аватар обновлён.',
			removed: 'Аватар удалён.'
		}
	} as const;

	let { data }: PageProps = $props();
	const session = getSession();
	const ownerId = untrack(() => (data.access.state === 'ready' ? data.access.ownerId : ''));
	const isActive = studioEditorIsActive(ownerId);
	let profile = $state.raw<Profile | null>(
		untrack(() => (data.access.state === 'ready' ? data.access.profile : null))
	);
	let displayName = $state(untrack(() => profile?.displayName ?? ''));
	let description = $state(untrack(() => profile?.description ?? ''));
	let nicknameDraft = $state<string>();
	const nickname = $derived(nicknameDraft ?? profile?.nickname ?? session.user?.nickname ?? '');
	const profilePath = $derived(
		validNickname(nickname)
			? resolve('/authors/[nickname]', { nickname: nickname.trim().toLowerCase() })
			: ''
	);

	const editHref = resolve('/studio/profile');
	const previewHref = resolve('/studio/profile?preview');
	const previewing = $derived(page.state.profilePreview || page.url.searchParams.has('preview'));
	const preview = $derived({
		displayName: displayName.trim() || 'Имя автора',
		nickname: nickname.trim().toLowerCase(),
		description: description.trim(),
		cover: profile?.cover ?? null,
		avatar: profile?.avatar ?? null
	});
	let returnFocus = false;

	let saving = $state(false);
	let saved = $state(false);
	let saveError = $state('');
	let fieldErrors = $state<Partial<Record<keyof ProfileInput, string>>>({});
	const inputs: Partial<Record<ProfileImageKind, HTMLInputElement>> = {};
	const uploadButtons: Partial<Record<ProfileImageKind, HTMLButtonElement>> = {};
	let choice = $state.raw<{ kind: ProfileImageKind; file: File } | null>(null);
	let changingImage = $state(false);
	let imageStatus = $state<Record<ProfileImageKind, { error: boolean; message: string }>>({
		cover: { error: false, message: '' },
		avatar: { error: false, message: '' }
	});

	function focusPreview(heading: HTMLElement) {
		heading.focus({ preventScroll: true });
		returnFocus = true;
	}

	function restorePreviewFocus(link: HTMLAnchorElement) {
		if (!returnFocus) return;
		link.focus({ preventScroll: true });
		returnFocus = false;
	}

	function validNickname(value: string) {
		return /^[A-Za-z0-9_-]{3,30}$/.test(value.trim());
	}

	function openPreview(event: MouseEvent) {
		if (event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey)
			return;
		event.preventDefault();
		void goto(previewHref, { shallow: true, state: { profilePreview: true }, reset: true });
	}

	function closePreview(event: MouseEvent) {
		if (
			!page.state.profilePreview ||
			event.button !== 0 ||
			event.metaKey ||
			event.ctrlKey ||
			event.shiftKey ||
			event.altKey
		)
			return;
		event.preventDefault();
		history.back();
	}

	function edited(field: keyof ProfileInput) {
		delete fieldErrors[field];
		saved = false;
		saveError = '';
	}

	async function focusField(form: HTMLFormElement, name: keyof ProfileInput) {
		await tick();
		const field = form.elements.namedItem(name);
		if (isActive() && field instanceof HTMLElement) field.focus();
	}

	async function save(event: SubmitEvent) {
		event.preventDefault();
		if (saving || changingImage || !isActive()) return;
		const form = event.currentTarget as HTMLFormElement;
		const input = {
			nickname: nickname.trim(),
			displayName: displayName.trim(),
			description: description.trim()
		};
		const errors: typeof fieldErrors = {};
		if (!validNickname(input.nickname))
			errors.nickname = 'Укажите от 3 до 30 латинских букв, цифр или знаков _ и -.';
		if (Array.from(input.displayName).length > 80 || !input.displayName)
			errors.displayName = 'Укажите имя от 1 до 80 символов.';
		if (Array.from(input.description).length > 5000)
			errors.description = 'Сократите описание до 5000 символов.';
		fieldErrors = errors;
		saved = false;
		saveError = '';
		const firstError = (Object.keys(errors) as (keyof ProfileInput)[])[0];
		if (firstError) {
			await focusField(form, firstError);
			return;
		}

		saving = true;
		try {
			const result = await api.saveProfile(ownerId, input);
			if (!isActive()) return;
			profile = result.profile;
			nicknameDraft = profile.nickname;
			displayName = profile.displayName;
			description = profile.description;
			saved = true;
			void session.refresh();
			void invalidate('studio:profile');
		} catch (error) {
			if (!isActive()) return;
			if (error instanceof ApiError && error.code === 'nickname_exists') {
				fieldErrors.nickname = 'Этот ник уже занят. Попробуйте другой.';
			} else {
				saveError = errorMessage(error);
			}
		} finally {
			saving = false;
		}
		if (fieldErrors.nickname) await focusField(form, 'nickname');
	}

	function pickImage(kind: ProfileImageKind, event: Event) {
		const input = event.currentTarget as HTMLInputElement;
		const file = input.files?.[0];
		input.value = '';
		if (!file || !profile || saving || changingImage) return;
		if (!imageTypes.includes(file.type)) {
			imageStatus[kind] = { error: true, message: 'Выберите изображение JPEG, PNG или WebP.' };
		} else if (file.size > maxImageBytes) {
			imageStatus[kind] = { error: true, message: 'Выберите файл размером до 5 МБ.' };
		} else {
			imageStatus[kind] = { error: false, message: '' };
			choice = { kind, file };
		}
	}

	function imageError(error: unknown) {
		if (error instanceof ApiError) {
			if (error.code === 'invalid_image')
				return 'Выберите исправное изображение JPEG, PNG или WebP до 5 МБ.';
			if (error.code === 'profile_required') return 'Сначала сохраните профиль.';
			if (error.code === 'storage_unavailable')
				return 'Хранилище изображений недоступно. Попробуйте чуть позже.';
		}
		return errorMessage(error);
	}

	async function changeImage(kind: ProfileImageKind, image?: Blob) {
		if (!profile || changingImage || saving || !isActive()) return;
		changingImage = true;
		imageStatus[kind] = { error: false, message: image ? 'Загружаем…' : 'Удаляем…' };
		try {
			const result = image
				? await api.uploadProfileImage(ownerId, kind, image)
				: await api.removeProfileImage(ownerId, kind);
			if (!isActive()) return;
			profile = result.profile;
			imageStatus[kind] = {
				error: false,
				message: image ? imageCopy[kind].saved : imageCopy[kind].removed
			};
			void invalidate('studio:profile');
		} catch (error) {
			if (!isActive()) return;
			const message = imageError(error);
			imageStatus[kind] = { error: true, message };
			if (image) throw new Error(message, { cause: error });
		} finally {
			changingImage = false;
		}
	}

	async function removeImage(kind: ProfileImageKind, trigger: HTMLButtonElement) {
		const { ownerDocument } = trigger;
		const wasFocused = ownerDocument.activeElement === trigger;
		await changeImage(kind);
		await tick();
		const active = ownerDocument.activeElement;
		if (
			!wasFocused ||
			!trigger.isConnected ||
			(active !== trigger && active !== ownerDocument.body)
		)
			return;
		(trigger.disabled ? uploadButtons[kind] : trigger)?.focus({ preventScroll: true });
	}
</script>

<svelte:head
	><title>{previewing ? 'Предпросмотр профиля' : 'Профиль автора'} — etheyl</title></svelte:head
>

{#if previewing}
	<div class="preview-toolbar">
		<h2 tabindex="-1" {@attach focusPreview}>Предпросмотр профиля</h2>
		<a class="studio-link" href={editHref} onclick={closePreview}>Вернуться к редактированию</a>
	</div>
	<AuthorProfileHeader profile={preview} />
{:else}
	<div class="editor-heading">
		<h1 class="studio-heading">{profile ? 'Профиль автора' : 'Создать профиль'}</h1>
		<a class="studio-link" href={previewHref} onclick={openPreview} {@attach restorePreviewFocus}
			>Как это видят читатели</a
		>
	</div>

	<div class="studio-form studio-panel">
		{#each imageKinds as kind (kind)}
			{@const copy = imageCopy[kind]}
			{@const image = profile?.[kind]}
			<section
				class={['media-section', { 'avatar-section': kind === 'avatar' }]}
				aria-labelledby={kind + '-title'}
			>
				<h2 id={kind + '-title'}>{copy.title}</h2>
				<div class={['image-slot', { 'avatar-slot': kind === 'avatar' }]}>
					{#if image}
						<img src={image.url} alt={copy.title + ' профиля'} />
					{:else if kind === 'cover'}
						<span>Обложка не загружена</span>
					{:else}
						<span role="img" aria-label={copy.empty}><Icon name="profile" size={27} /></span>
					{/if}
				</div>
				<div class="media-controls">
					<p class="media-hint" id={kind + '-hint'}>JPEG, PNG или WebP, до 5&nbsp;МБ</p>
					<div class="media-actions">
						<input
							type="file"
							hidden
							accept={imageTypes.join(',')}
							{@attach (input) => {
								inputs[kind] = input;
								return () => {
									delete inputs[kind];
								};
							}}
							onchange={(event) => pickImage(kind, event)}
						/>
						<button
							type="button"
							class="button button-outline"
							aria-label={kind === 'cover' ? 'Загрузить обложку' : 'Загрузить аватар'}
							aria-describedby={kind + '-hint'}
							disabled={!profile || saving || changingImage}
							{@attach (button) => {
								uploadButtons[kind] = button;
								return () => {
									delete uploadButtons[kind];
								};
							}}
							onclick={() => inputs[kind]?.click()}>Загрузить</button
						>
						<button
							type="button"
							class="button button-outline remove-button"
							aria-label={kind === 'cover' ? 'Удалить обложку' : 'Удалить аватар'}
							disabled={!profile || !image || saving || changingImage}
							onclick={(event) => void removeImage(kind, event.currentTarget)}>Удалить</button
						>
					</div>
				</div>
				<p class={['media-status', { 'error-feedback': imageStatus[kind].error }]} role="status">
					{imageStatus[kind].message}
				</p>
			</section>
		{/each}
		{#if !profile}<p class="first-save-hint">
				Сначала сохраните профиль, чтобы загрузить обложку и аватар.
			</p>{/if}

		<form aria-busy={saving} novalidate onsubmit={save}>
			<fieldset disabled={saving}>
				<div class="field">
					<label for="nickname">Ник</label>
					<input
						id="nickname"
						name="nickname"
						value={nickname}
						oninput={(event) => {
							nicknameDraft = event.currentTarget.value;
							edited('nickname');
						}}
						aria-describedby={fieldErrors.nickname
							? 'nickname-error nickname-hint'
							: 'nickname-hint'}
						aria-invalid={fieldErrors.nickname ? true : undefined}
						autocapitalize="none"
						spellcheck="false"
						required
					/>
					{#if fieldErrors.nickname}<p id="nickname-error" class="field-error" role="alert">
							{fieldErrors.nickname}
						</p>{/if}
					<p id="nickname-hint" class="field-hint">
						{#if profilePath}
							<span class="profile-path">{profilePath}</span>
							{#if profile && nickname.trim().toLowerCase() !== profile.nickname}
								<span>Адрес изменится после сохранения.</span>
							{/if}
						{:else}
							От 3 до 30 латинских букв, цифр, знаков _ или -.
						{/if}
					</p>
				</div>
				<div class="field">
					<label for="display-name">Отображаемое имя</label>
					<input
						id="display-name"
						name="displayName"
						bind:value={displayName}
						oninput={() => edited('displayName')}
						aria-invalid={fieldErrors.displayName ? true : undefined}
						aria-describedby={fieldErrors.displayName ? 'display-name-error' : undefined}
						required
					/>
					{#if fieldErrors.displayName}<p id="display-name-error" class="field-error" role="alert">
							{fieldErrors.displayName}
						</p>{/if}
				</div>
				<div class="field">
					<label for="description">Описание</label>
					<textarea
						id="description"
						name="description"
						rows="4"
						bind:value={description}
						oninput={() => edited('description')}
						aria-invalid={fieldErrors.description ? true : undefined}
						aria-describedby={fieldErrors.description
							? 'description-error description-hint'
							: 'description-hint'}></textarea>
					{#if fieldErrors.description}<p id="description-error" class="field-error" role="alert">
							{fieldErrors.description}
						</p>{/if}
					<p id="description-hint" class="field-hint">Необязательное, до 5000 символов.</p>
				</div>
				{#if !profile}<p class="publication-hint">Первое сохранение опубликует профиль.</p>{/if}
				<div class="save-row">
					<button type="submit" class="button button-primary" disabled={saving || changingImage}
						>{saving ? 'Сохраняем…' : 'Сохранить'}</button
					>
					<p class="save-status" role="status">{saved ? 'Сохранено' : ''}</p>
				</div>
			</fieldset>
			{#if saveError}<p class="feedback error-feedback" role="alert">{saveError}</p>{/if}
		</form>
	</div>
{/if}

{#if choice}
	{@const selected = choice}
	<ProfileImageCrop
		file={selected.file}
		kind={selected.kind}
		upload={(image) => changeImage(selected.kind, image)}
		onclose={() => (choice = null)}
	/>
{/if}

<style>
	.editor-heading {
		display: flex;
		max-width: 640px;
		flex-wrap: wrap;
		align-items: baseline;
		justify-content: space-between;
		gap: 10px 20px;
	}
	a.studio-link {
		color: var(--blue-hover);
		font-size: 14px;
		font-weight: 750;
		text-decoration: underline;
		text-decoration-color: currentColor;
		text-underline-offset: 4px;
	}
	a.studio-link:hover {
		color: var(--blue);
	}
	.preview-toolbar {
		display: flex;
		flex-wrap: wrap;
		align-items: baseline;
		justify-content: space-between;
		gap: 10px 24px;
		margin-block-end: 30px;
	}
	.preview-toolbar h2 {
		margin: 0;
		font-size: 16px;
		font-weight: 800;
	}
	.media-section h2 {
		margin: 0 0 10px;
		font-size: 15px;
		font-weight: 800;
	}
	.image-slot {
		display: grid;
		aspect-ratio: 4 / 1;
		place-items: center;
		position: relative;
		overflow: hidden;
		border-radius: var(--radius);
		background: var(--surface);
		color: var(--muted);
		font-size: 13px;
	}
	.image-slot img {
		position: absolute;
		inset: 0;
		width: 100%;
		height: 100%;
		object-fit: cover;
	}
	.media-controls {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		justify-content: space-between;
		gap: 8px 16px;
		margin-top: 10px;
	}
	.media-hint,
	.first-save-hint,
	.publication-hint,
	.media-status,
	.save-status {
		font-size: 13px;
		line-height: 1.55;
	}
	.media-hint,
	.first-save-hint,
	.publication-hint,
	.save-status {
		color: var(--muted);
	}
	.media-actions,
	.save-row {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 8px 12px;
	}
	.media-actions .button {
		min-height: 42px;
		padding: 8px 13px;
		font-size: 13px;
	}
	.remove-button:not(:disabled):hover {
		border-color: var(--error);
		background: var(--white);
		color: var(--error);
	}
	.media-actions .button:disabled {
		cursor: default;
	}
	.media-actions .button:disabled:hover {
		border-color: #bdc3d3;
		background: var(--white);
	}
	.media-status:not(:empty) {
		margin-block-start: 8px;
	}
	.avatar-section {
		display: grid;
		grid-template-columns: 88px minmax(0, 1fr);
		column-gap: 16px;
		margin-block-start: 22px;
	}
	.avatar-section h2,
	.avatar-section .media-status {
		grid-column: 1 / -1;
	}
	.avatar-section .media-controls {
		align-self: center;
		margin: 0;
	}
	.avatar-slot {
		aspect-ratio: 1;
		border-radius: 16px;
	}
	.first-save-hint {
		margin-block-start: 12px;
	}
	form {
		margin-block-start: 22px;
		padding-block-start: 22px;
		border-block-start: 1px solid var(--line);
	}
	.field-hint,
	.field-error {
		font-size: 12px;
	}
	.profile-path {
		margin-inline-end: 6px;
		overflow-wrap: anywhere;
	}
	textarea[aria-invalid='true'] {
		border-color: var(--error);
	}
	.publication-hint {
		margin-block-end: 10px;
	}
	@media (max-width: 480px) {
		.avatar-section {
			grid-template-columns: 64px minmax(0, 1fr);
			column-gap: 10px;
		}
		.avatar-section .media-controls {
			justify-content: flex-start;
		}
		.media-actions {
			gap: 8px;
		}
		.media-actions .button {
			padding-inline: 10px;
		}
	}
	@media (max-width: 360px) {
		.avatar-section {
			grid-template-columns: 56px minmax(0, 1fr);
		}
	}
</style>
