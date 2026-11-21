<script lang="ts">
	import type { PostFile } from '#lib/api/client.js';
	import Icon from '#lib/components/Icon.svelte';

	let { files }: { files: PostFile[] } = $props();
	let images = $derived(files.filter(({ kind }) => kind === 'image'));
	let recordings = $derived(files.filter(({ kind }) => kind === 'audio' || kind === 'video'));
	let attachments = $derived(files.filter(({ kind }) => kind === 'attachment'));
	const number = new Intl.NumberFormat('ru-RU', { maximumFractionDigits: 1 });

	function sizeLabel(bytes: number) {
		if (bytes < 1024) return `${number.format(bytes)} Б`;
		if (bytes < 1024 ** 2) return `${number.format(bytes / 1024)} КБ`;
		return `${number.format(bytes / 1024 ** 2)} МБ`;
	}
</script>

{#if images.length}
	<ul class="gallery" aria-label="Галерея изображений">
		{#each images as image (image.id)}
			<li>
				<a href={image.url} target="_blank" rel="external noopener">
					<img
						src={image.thumbnailUrl ?? image.url}
						alt={image.filename}
						loading="lazy"
						decoding="async"
					/>
					<span class="sr-only"> (откроется в новой вкладке)</span>
				</a>
			</li>
		{/each}
	</ul>
{/if}

{#if recordings.length}
	<div class="recordings">
		{#each recordings as file (file.id)}
			<figure>
				<figcaption id="post-media-{file.id}">{file.filename}</figcaption>
				{#if file.kind === 'audio'}
					<audio src={file.url} controls preload="metadata" aria-labelledby="post-media-{file.id}"
					></audio>
				{:else}
					<!-- svelte-ignore a11y_media_has_caption (The media API does not supply a caption track.) -->
					<video
						src={file.url}
						controls
						preload="metadata"
						playsinline
						aria-labelledby="post-media-{file.id}"
					></video>
				{/if}
			</figure>
		{/each}
	</div>
{/if}

{#if attachments.length}
	<section class="attachments" aria-labelledby="attachment-heading">
		<h2 id="attachment-heading">Вложения</h2>
		<ul class="files">
			{#each attachments as file (file.id)}
				<li>
					<a class="file" href={file.url} download={file.filename} rel="external">
						<span class="file-icon"><Icon name="post" /></span>
						<span class="file-details">
							<span class="sr-only">Скачать </span>
							<strong>{file.filename}</strong>
							<span class="file-size">{sizeLabel(file.sizeBytes)}</span>
						</span>
						<span class="file-download"><Icon name="download" /></span>
					</a>
				</li>
			{/each}
		</ul>
	</section>
{/if}

<style>
	.gallery,
	.recordings,
	.attachments {
		margin-top: 28px;
	}
	.gallery {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		align-items: start;
		gap: 12px;
	}
	.gallery li:only-child {
		grid-column: 1 / -1;
	}
	.gallery a {
		display: block;
		border-radius: var(--radius);
	}
	.gallery img {
		width: 100%;
		aspect-ratio: 3 / 2;
		object-fit: contain;
		border-radius: inherit;
		background: var(--surface);
	}
	.gallery a:hover {
		outline: 2px solid var(--blue);
		outline-offset: 2px;
	}
	.recordings {
		display: grid;
		gap: 20px;
	}
	figcaption {
		margin-bottom: 8px;
		font-size: 14px;
		font-weight: 700;
		overflow-wrap: anywhere;
	}
	audio,
	video {
		display: block;
		width: 100%;
	}
	video {
		aspect-ratio: 16 / 9;
		border-radius: var(--radius);
		background: var(--ink);
	}
	h2 {
		margin-bottom: 10px;
		font-size: 19px;
		font-weight: 800;
	}
	.files {
		border-top: 1px solid var(--line);
	}
	.file {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 12px 10px;
		border-bottom: 1px solid var(--line);
		font-size: 14px;
	}
	.file:hover {
		background: var(--surface);
		color: var(--blue-hover);
	}
	.file-details {
		display: grid;
		flex: 1;
		gap: 2px;
		min-width: 0;
		overflow-wrap: anywhere;
	}
	.file-size {
		color: var(--muted);
		font-size: 12px;
		font-variant-numeric: tabular-nums;
	}
	.file-icon,
	.file-download {
		display: grid;
		flex: none;
		place-items: center;
		width: 36px;
		height: 36px;
		color: var(--blue-hover);
	}
	.file-icon {
		border-radius: 8px;
		background: var(--blue-soft);
	}
</style>
