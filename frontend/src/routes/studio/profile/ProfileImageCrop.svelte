<script lang="ts">
	let {
		file,
		kind,
		upload,
		onclose
	}: {
		file: File;
		kind: 'cover' | 'avatar';
		upload: (image: Blob) => Promise<void>;
		onclose: () => void;
	} = $props();

	const id = $props.id();
	const aspect = $derived(kind === 'cover' ? 4 : 1);
	let dialog: HTMLDialogElement;
	let initialFocus: Element | null;
	let source = $state.raw<ImageBitmap>();
	let loading = $state(true);
	let pending = $state(false);
	let message = $state('');
	let zoom = $state(1);
	let center = $state.raw({ x: 0, y: 0 });
	let drag: { id: number; x: number; y: number } | undefined;

	const percent = $derived(`${Math.round(zoom * 100)}%`);
	const crop = $derived.by(() => {
		if (!source) return;
		const width = Math.min(source.width, source.height * aspect) / zoom;
		const height = width / aspect;
		return {
			x: clamp(center.x - width / 2, source.width - width),
			y: clamp(center.y - height / 2, source.height - height),
			width,
			height
		};
	});

	const arrows = new Map([
		['ArrowLeft', [-1, 0]],
		['ArrowRight', [1, 0]],
		['ArrowUp', [0, -1]],
		['ArrowDown', [0, 1]]
	]);

	function clamp(value: number, max: number) {
		return Math.max(0, Math.min(value, max));
	}

	function openDialog(element: HTMLDialogElement) {
		let active = true;
		let decoded: ImageBitmap | undefined;
		dialog = element;
		element.showModal();
		initialFocus = element.ownerDocument.activeElement;
		void decode(file);
		return () => {
			active = false;
			decoded?.close();
			element.close();
		};

		async function decode(selected: File) {
			try {
				const bitmap = await createImageBitmap(selected);
				if (!active) {
					bitmap.close();
					return;
				}
				const { width, height } = bitmap;
				if (width > 8192 || height > 8192 || width * height > 16_000_000) {
					message = 'Выберите изображение до 16 Мп и до 8192 пикселей по каждой стороне.';
					bitmap.close();
					return;
				}
				center = { x: width / 2, y: height / 2 };
				source = decoded = bitmap;
			} catch {
				if (active) message = 'Не удалось открыть изображение. Выберите другой файл.';
			} finally {
				if (active) loading = false;
			}
		}
	}

	function focusFrame(frame: HTMLElement) {
		if (frame.ownerDocument.activeElement === initialFocus) frame.focus({ preventScroll: true });
	}

	function drawPreview(canvas: HTMLCanvasElement) {
		if (!source) return;
		const scale = Math.min(1, 2048 / Math.max(source.width, source.height));
		canvas.width = Math.max(1, Math.round(source.width * scale));
		canvas.height = Math.max(1, Math.round(source.height * scale));
		canvas.getContext('2d')?.drawImage(source, 0, 0, canvas.width, canvas.height);
	}

	function moveImage(horizontal: number, vertical: number) {
		if (!source || !crop) return;
		const { x, y, width, height } = crop;
		center = {
			x: width / 2 + clamp(x - horizontal * width, source.width - width),
			y: height / 2 + clamp(y - vertical * height, source.height - height)
		};
	}

	function startDrag(event: PointerEvent & { currentTarget: HTMLElement }) {
		if (event.button !== 0 || drag || pending) return;
		drag = { id: event.pointerId, x: event.clientX, y: event.clientY };
		event.currentTarget.setPointerCapture(event.pointerId);
	}

	function dragImage(event: PointerEvent & { currentTarget: HTMLElement }) {
		if (!drag || drag.id !== event.pointerId || pending) return;
		const { width, height } = event.currentTarget.getBoundingClientRect();
		moveImage((event.clientX - drag.x) / width, (event.clientY - drag.y) / height);
		drag.x = event.clientX;
		drag.y = event.clientY;
	}

	function stopDrag(event: PointerEvent) {
		if (drag?.id === event.pointerId) drag = undefined;
	}

	function moveWithKeys(event: KeyboardEvent) {
		const direction = arrows.get(event.key);
		if (!direction || pending) return;
		event.preventDefault();
		const [horizontal, vertical] = direction;
		const step = event.shiftKey ? 0.01 : 0.04;
		moveImage(horizontal * step, vertical * step);
	}

	function changeZoom(event: Event & { currentTarget: HTMLInputElement }) {
		if (pending || !crop) return;
		center = { x: crop.x + crop.width / 2, y: crop.y + crop.height / 2 };
		zoom = event.currentTarget.valueAsNumber;
	}

	function close(event?: Event) {
		event?.preventDefault();
		if (pending) return;
		dialog.close();
		onclose();
	}

	async function encodeCrop() {
		if (!source || !crop) return null;
		const canvas = document.createElement('canvas');
		const targetHeight = kind === 'cover' ? 400 : 512;
		canvas.height = Math.max(1, Math.min(targetHeight, Math.floor(crop.height)));
		canvas.width = canvas.height * aspect;
		const context = canvas.getContext('2d');
		if (!context) return null;
		context.imageSmoothingQuality = 'high';
		context.drawImage(
			source,
			crop.x,
			crop.y,
			crop.width,
			crop.height,
			0,
			0,
			canvas.width,
			canvas.height
		);
		const type = file.type === 'image/jpeg' ? 'image/jpeg' : 'image/png';
		return new Promise<Blob | null>((resolve) => canvas.toBlob(resolve, type, 0.92));
	}

	async function apply() {
		if (pending || !source) return;
		pending = true;
		message = '';
		try {
			const image = await encodeCrop().catch(() => null);
			if (!image || image.size > 5 * 1024 * 1024) {
				message = 'Не удалось подготовить кадр. Попробуйте ещё раз.';
				return;
			}
			await upload(image);
		} catch (error) {
			message =
				error instanceof Error
					? error.message
					: 'Не удалось загрузить изображение. Повторите попытку.';
			return;
		} finally {
			pending = false;
		}
		close();
	}
</script>

<dialog {@attach openDialog} aria-labelledby="{id}-title" oncancel={close}>
	<div class="dialog-body" aria-busy={pending}>
		<h2 id="{id}-title">{kind === 'cover' ? 'Кадрирование обложки' : 'Кадрирование аватара'}</h2>
		{#if source && crop}
			<div class="workspace">
				<!-- svelte-ignore a11y_no_noninteractive_tabindex, a11y_no_noninteractive_element_interactions (The crop widget needs focus for two-axis arrow key input.) -->
				<div
					class={['crop-frame', { avatar: kind === 'avatar' }]}
					role="application"
					aria-label="Положение изображения"
					aria-describedby="{id}-instructions"
					aria-disabled={pending}
					tabindex="0"
					{@attach focusFrame}
					onkeydown={moveWithKeys}
					onpointerdown={startDrag}
					onpointermove={dragImage}
					onlostpointercapture={stopDrag}
				>
					<canvas
						{@attach drawPreview}
						aria-hidden="true"
						style:left={`${(-crop.x / crop.width) * 100}%`}
						style:top={`${(-crop.y / crop.height) * 100}%`}
						style:width={`${(source.width / crop.width) * 100}%`}
						style:height={`${(source.height / crop.height) * 100}%`}
					></canvas>
				</div>
			</div>
			<p id="{id}-instructions" class="instructions">
				Перетащите изображение или сдвиньте его стрелками.
			</p>
			<div class="zoom">
				<label for="{id}-zoom">Масштаб</label>
				<input
					id="{id}-zoom"
					type="range"
					min="1"
					max="3"
					step="0.01"
					value={zoom}
					aria-valuetext={percent}
					disabled={pending}
					oninput={changeZoom}
				/>
				<span aria-hidden="true">{percent}</span>
			</div>
		{:else if loading}
			<p class="instructions" role="status">Открываем изображение…</p>
		{/if}
		{#if message}<p class="feedback error-feedback" role="alert">{message}</p>{/if}
		<div class="actions">
			<button type="button" class="button button-outline" disabled={pending} onclick={close}>
				Отмена
			</button>
			{#if loading || source}
				<button
					type="button"
					class="button button-primary"
					disabled={!source}
					aria-disabled={pending}
					onclick={apply}
				>
					{pending ? 'Загружаем…' : 'Применить и загрузить'}
				</button>
			{/if}
		</div>
	</div>
</dialog>

<style>
	dialog {
		inset: 0;
		width: min(600px, calc(100% - 24px));
		max-height: calc(100% - 24px);
		margin: auto;
		padding: 24px;
		border: 1px solid var(--line);
		border-radius: 16px;
		background: var(--white);
		color: var(--ink);
	}
	dialog::backdrop {
		background: #171a246e;
	}
	.dialog-body {
		display: grid;
		gap: 16px;
	}
	h2 {
		margin: 0;
		font-size: 22px;
		font-weight: 800;
		line-height: 1.3;
		letter-spacing: -0.025em;
	}
	.workspace {
		display: grid;
		place-items: center;
		overflow: hidden;
		padding: 20px;
		border-radius: 12px;
		background: var(--surface);
	}
	.crop-frame {
		position: relative;
		width: 100%;
		aspect-ratio: 4;
		cursor: grab;
		touch-action: none;
	}
	.crop-frame.avatar {
		width: min(256px, 100%);
		aspect-ratio: 1;
	}
	.crop-frame::after {
		position: absolute;
		inset: 0;
		border: 1px solid var(--white);
		border-radius: 7px;
		box-shadow: 0 0 0 100vmax #171a246e;
		content: '';
		pointer-events: none;
	}
	.crop-frame.avatar::after {
		border-radius: 18%;
	}
	.crop-frame:focus-visible {
		outline: 0;
	}
	.crop-frame:focus-visible::after {
		outline: 3px solid var(--blue);
		outline-offset: 2px;
	}
	.crop-frame:active {
		cursor: grabbing;
	}
	.crop-frame[aria-disabled='true'] {
		cursor: wait;
	}
	canvas {
		position: absolute;
		display: block;
		pointer-events: none;
	}
	.instructions,
	.feedback {
		margin: 0;
		font-size: 13px;
		line-height: 1.5;
	}
	.instructions {
		color: var(--muted);
	}
	.zoom {
		display: grid;
		grid-template-columns: auto minmax(0, 1fr) 4ch;
		align-items: center;
		gap: 12px;
		font-size: 13px;
		font-weight: 700;
	}
	.zoom input {
		width: 100%;
		min-height: 28px;
		margin: 0;
		accent-color: var(--blue);
		cursor: pointer;
	}
	.zoom span {
		text-align: end;
		font-variant-numeric: tabular-nums;
	}
	.actions {
		display: flex;
		flex-wrap: wrap;
		justify-content: flex-end;
		gap: 8px;
	}
	.actions .button {
		min-height: 44px;
		padding: 9px 16px;
		font-size: 14px;
	}
	.actions .button[aria-disabled='true'] {
		background: var(--blue);
		opacity: 0.65;
		cursor: wait;
	}
	@media (max-width: 420px) {
		dialog {
			padding: 16px;
		}
		.workspace {
			padding: 12px;
		}
		.actions .button {
			flex: auto;
		}
	}
</style>
