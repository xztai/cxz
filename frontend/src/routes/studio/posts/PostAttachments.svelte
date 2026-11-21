<script lang="ts">
	import { onDestroy, untrack } from 'svelte';
	import { SvelteSet } from 'svelte/reactivity';
	import { api, ApiError, errorMessage, type PostFile } from '#lib/api/client.js';
	import Icon from '#lib/components/Icon.svelte';
	import { studioEditorIsActive } from '#lib/session.svelte.js';

	type PendingUpload = {
		id: number;
		filename: string;
		sizeBytes: number;
		kind: PostFile['kind'];
		progress: number | null;
		error?: string;
	};

	let {
		postId,
		files,
		ownerId,
		disabled,
		busy = $bindable(false),
		onFilesChange
	}: {
		postId?: string;
		files: PostFile[];
		ownerId: string;
		disabled: boolean;
		busy?: boolean;
		onFilesChange: (update: (previous: PostFile[]) => PostFile[]) => void;
	} = $props();

	const fileTypes = {
		image: { label: 'Изображение', accept: '.jpg,.jpeg,.png,.gif,.webp' },
		audio: { label: 'Аудио', accept: 'audio/*' },
		video: { label: 'Видео', accept: 'video/*' },
		attachment: { label: 'Вложение', accept: undefined }
	} satisfies Record<PostFile['kind'], { label: string; accept: string | undefined }>;
	const kinds = Object.keys(fileTypes) as PostFile['kind'][];
	const number = new Intl.NumberFormat('ru-RU', { maximumFractionDigits: 1 });
	let kind = $state<PostFile['kind']>('attachment');
	let dragDepth = $state(0);
	let uploads = $state<PendingUpload[]>([]);
	let detachingId = $state<string>();
	let message = $state('');
	let detachError = $state('');
	const failedThumbnails = new SvelteSet<string>();
	let nextUploadId = 0;
	let uploadController: AbortController | undefined;
	const locked = $derived(!postId || disabled || busy);
	const isActive = studioEditorIsActive(untrack(() => ownerId));
	onDestroy(() => uploadController?.abort());

	function sizeLabel(bytes: number) {
		if (bytes < 1024) return `${number.format(bytes)} Б`;
		if (bytes < 1024 ** 2) return `${number.format(bytes / 1024)} КБ`;
		return `${number.format(bytes / 1024 ** 2)} МБ`;
	}

	function removeUpload(id: number) {
		uploads = uploads.filter((item) => item.id !== id);
	}

	async function addFiles(selected: File[]) {
		if (!postId || locked || !isActive() || selected.length === 0) return;
		const id = postId;
		const controller = new AbortController();
		uploadController = controller;
		busy = true;
		message = '';
		uploads.push(
			...selected.map((file) => ({
				id: ++nextUploadId,
				filename: file.name,
				sizeBytes: file.size,
				kind,
				progress: null
			}))
		);
		const queued = uploads.slice(-selected.length);
		let added = 0;
		try {
			for (const [index, file] of selected.entries()) {
				if (!isActive()) return;
				const item = queued[index];
				if (file.size === 0 || file.size > 100 * 1024 ** 2) {
					item.error = file.size === 0 ? 'Файл пуст.' : 'Размер файла не должен превышать 100 МБ.';
					continue;
				}
				item.progress = 0;
				try {
					const { media } = await api.uploadPostMedia(
						ownerId,
						id,
						item.kind,
						file,
						(fraction) => (item.progress = fraction),
						controller.signal
					);
					if (!isActive()) return;
					onFilesChange((previous) => [...previous, media]);
					removeUpload(item.id);
					added++;
				} catch (error) {
					if (isActive()) {
						item.error =
							error instanceof ApiError && error.status === 413
								? 'Размер файла не должен превышать 100 МБ.'
								: errorMessage(error);
					}
				}
			}
			if (added) message = added === 1 ? 'Файл прикреплён.' : `Прикреплено файлов: ${added}.`;
		} finally {
			uploadController = undefined;
			busy = false;
		}
	}

	function selectFiles(event: Event & { currentTarget: HTMLInputElement }) {
		const selected = Array.from(event.currentTarget.files ?? []);
		event.currentTarget.value = '';
		void addFiles(selected);
	}

	function dragOver(event: DragEvent) {
		if (!event.dataTransfer?.types.includes('Files')) return;
		event.preventDefault();
		event.dataTransfer.dropEffect = locked ? 'none' : 'copy';
	}

	function dropFiles(event: DragEvent) {
		event.preventDefault();
		dragDepth = 0;
		void addFiles(Array.from(event.dataTransfer?.files ?? []));
	}

	async function detach(file: PostFile) {
		if (!postId || locked || !isActive()) return;
		busy = true;
		detachingId = file.id;
		message = '';
		detachError = '';
		try {
			await api.detachPostMedia(ownerId, postId, file.id);
			if (!isActive()) return;
			onFilesChange((previous) => previous.filter((item) => item.id !== file.id));
			message = 'Файл откреплён.';
		} catch (error) {
			if (isActive()) detachError = errorMessage(error);
		} finally {
			detachingId = undefined;
			busy = false;
		}
	}
</script>

<section class="attachments studio-panel" aria-labelledby="attachments-heading">
	<h2 id="attachments-heading">Файлы</h2>
	{#if !postId}
		<p id="attachments-locked" class="locked-hint">
			<Icon name="lock" size={16} />
			<span>Сохраните пост, чтобы прикрепить файлы.</span>
		</p>
	{/if}
	<fieldset disabled={locked} aria-describedby={!postId ? 'attachments-locked' : undefined}>
		<div class="field file-type">
			<label for="media-kind">Тип файла</label>
			<select id="media-kind" bind:value={kind}>
				{#each kinds as type (type)}
					<option value={type}>{fileTypes[type].label}</option>
				{/each}
			</select>
		</div>
		<label
			class={['dropzone', { dragging: dragDepth > 0 && !locked, locked }]}
			ondragenter={(event) => {
				if (event.dataTransfer?.types.includes('Files')) dragDepth++;
			}}
			ondragleave={() => (dragDepth = Math.max(0, dragDepth - 1))}
			ondragover={dragOver}
			ondrop={dropFiles}
		>
			<input
				class="sr-only"
				type="file"
				multiple
				accept={fileTypes[kind].accept}
				aria-label="Выбрать файлы"
				aria-describedby="file-drop-hint file-size-limit"
				onchange={selectFiles}
			/>
			<Icon name="upload" size={23} />
			<span class="drop-action">Выбрать файлы</span>
			<span id="file-drop-hint" class="drop-hint">или перетащить сюда</span>
			<span id="file-size-limit" class="drop-limit">До 100 МБ на файл</span>
		</label>
	</fieldset>

	{#if files.length || uploads.length}
		<ul class="file-list" aria-label="Файлы поста">
			{#each files as file (file.id)}
				<li class="file-row">
					<div class="thumbnail">
						<Icon name="post" size={20} />
						{#if file.thumbnailUrl && !failedThumbnails.has(file.id)}
							<img
								src={file.thumbnailUrl}
								alt=""
								loading="lazy"
								onerror={() => failedThumbnails.add(file.id)}
							/>
						{/if}
					</div>
					<div class="file-details">
						<p class="filename">{file.filename}</p>
						<p class="file-meta">{fileTypes[file.kind].label} · {sizeLabel(file.sizeBytes)}</p>
						<button
							type="button"
							class="file-action studio-link"
							aria-label={`Открепить ${file.filename}`}
							disabled={locked}
							onclick={() => detach(file)}
							>{detachingId === file.id ? 'Открепляем…' : 'Открепить'}</button
						>
					</div>
				</li>
			{/each}
			{#each uploads as item (item.id)}
				<li class="file-row">
					<div class="thumbnail"><Icon name="post" size={20} /></div>
					<div class="file-details">
						<p class="filename">{item.filename}</p>
						<p class="file-meta">{fileTypes[item.kind].label} · {sizeLabel(item.sizeBytes)}</p>
						{#if item.error}
							<p class="field-error" role="alert">{item.error}</p>
							<button
								type="button"
								class="file-action studio-link"
								aria-label={`Убрать ${item.filename} из списка загрузок`}
								onclick={() => removeUpload(item.id)}>Убрать</button
							>
						{:else if item.progress === null}
							<p class="transfer-label">В очереди</p>
						{:else}
							<progress max="1" value={item.progress} aria-label={`Загрузка ${item.filename}`}
							></progress>
							<p class="transfer-label">
								{item.progress === 1
									? 'Обработка файла…'
									: `Загрузка · ${Math.floor(item.progress * 100)}%`}
							</p>
						{/if}
					</div>
				</li>
			{/each}
		</ul>
	{/if}
	{#if detachError}<p class="feedback error-feedback" role="alert">{detachError}</p>{/if}
	<p class="feedback" role="status">{message}</p>
</section>

<style>
	.attachments {
		min-width: 0;
	}
	h2 {
		font-size: 19px;
		font-weight: 800;
		line-height: 1.4;
		margin-block-end: 16px;
	}
	.locked-hint {
		display: flex;
		align-items: start;
		gap: 8px;
		margin-block-end: 16px;
		color: var(--muted);
		font-size: 13px;
		line-height: 1.55;
	}
	.locked-hint :global(svg) {
		flex-shrink: 0;
		margin-top: 2px;
	}
	fieldset {
		min-width: 0;
	}
	.file-type {
		margin-block-end: 14px;
	}
	select {
		width: 100%;
		min-height: 44px;
		padding: 8px 11px;
		border: 1px solid #aeb6ca;
		border-radius: 9px;
		background: var(--white);
		color: var(--ink);
		font-size: 14px;
	}
	select:disabled {
		border-color: var(--line);
		background: var(--surface);
		color: var(--muted);
		opacity: 1;
	}
	.dropzone {
		display: grid;
		justify-items: center;
		gap: 3px;
		padding: 20px 12px;
		border: 1px dashed #aeb6ca;
		border-radius: 10px;
		background: var(--surface);
		color: var(--blue-hover);
		text-align: center;
		cursor: pointer;
	}
	.dropzone:focus-within {
		outline: 3px solid var(--blue);
		outline-offset: 3px;
	}
	.dropzone:not(.locked):hover,
	.dropzone.dragging {
		border-color: var(--blue);
		background: var(--blue-soft);
	}
	.dropzone.locked {
		color: var(--muted);
		cursor: not-allowed;
	}
	.drop-action {
		margin-block-start: 4px;
		font-size: 14px;
		font-weight: 800;
	}
	.drop-hint {
		color: var(--muted);
		font-size: 13px;
	}
	.drop-limit {
		margin-block-start: 5px;
		color: var(--muted);
		font-size: 11px;
	}
	.file-list {
		margin-block-start: 16px;
	}
	.file-row {
		display: grid;
		grid-template-columns: 44px minmax(0, 1fr);
		gap: 10px;
		padding-block: 12px 4px;
		border-block-start: 1px solid var(--line);
	}
	.file-row:last-child {
		padding-block-end: 0;
	}
	.thumbnail {
		position: relative;
		display: grid;
		place-items: center;
		aspect-ratio: 1;
		border-radius: 8px;
		background: var(--surface);
		color: var(--muted);
	}
	.thumbnail img {
		position: absolute;
		inset: 0;
		width: 100%;
		height: 100%;
		border-radius: inherit;
		object-fit: cover;
	}
	.file-details {
		overflow-wrap: anywhere;
	}
	.filename {
		font-size: 13px;
		font-weight: 800;
		line-height: 1.5;
	}
	.file-meta,
	.transfer-label {
		margin-block-start: 2px;
		color: var(--muted);
		font-size: 12px;
		font-variant-numeric: tabular-nums;
	}
	.file-action {
		min-height: 44px;
		font-size: 13px;
		cursor: pointer;
	}
	progress {
		display: block;
		width: 100%;
		height: 6px;
		margin-block: 8px 4px;
		accent-color: var(--blue);
	}
	.feedback {
		margin-block-end: 0;
	}
	@media (max-width: 520px) {
		.attachments.studio-panel {
			padding: 16px;
		}
	}
</style>
