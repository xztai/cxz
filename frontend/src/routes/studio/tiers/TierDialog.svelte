<script module lang="ts">
	type RetireAction = 'delete' | 'archive';
	export type TierDialogAction = { type: 'create' } | { type: 'edit' | RetireAction; tier: Tier };
</script>

<script lang="ts">
	import { tick } from 'svelte';
	import { errorMessage, type Tier, type TierInput } from '#lib/api/client.js';

	let {
		action,
		onsave,
		onretire,
		onclose
	}: {
		action: TierDialogAction;
		onsave: (input: TierInput, tier?: Tier) => Promise<void>;
		onretire: (tier: Tier, action: RetireAction) => Promise<void>;
		onclose: () => void;
	} = $props();

	type Field = 'name' | 'description' | 'price';
	let dialog: HTMLDialogElement;
	let pending = $state(false);
	let requestError = $state('');
	let fieldErrors = $state<Partial<Record<Field, string>>>({});
	const tier = $derived(action.type === 'create' ? undefined : action.tier);
	const titles: Record<TierDialogAction['type'], string> = {
		create: 'Новый уровень',
		edit: 'Изменить уровень',
		delete: 'Удалить уровень?',
		archive: 'Архивировать уровень?'
	};
	const confirming = $derived(action.type === 'delete' || action.type === 'archive');
	const usedPrice = $derived(action.type === 'edit' && action.tier.used);
	const price = $derived(
		tier
			? (tier.priceKopecks / 100).toFixed(tier.priceKopecks % 100 === 0 ? 0 : 2).replace('.', ',')
			: ''
	);

	function openDialog(element: HTMLDialogElement) {
		dialog = element;
		element.showModal();
		return () => {
			if (element.open) element.close();
		};
	}

	function close() {
		if (pending) return;
		dialog.close();
		onclose();
	}

	function getPriceKopecks(value: string) {
		const parts = /^(\d+)(?:[,.](\d{1,2}))?$/.exec(value.trim());
		if (!parts) return NaN;
		return Number(parts[1]) * 100 + Number((parts[2] ?? '').padEnd(2, '0'));
	}

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (pending) return;
		const form = event.currentTarget as HTMLFormElement;
		requestError = '';

		const selected = action;
		if (selected.type === 'delete' || selected.type === 'archive') {
			const { tier, type } = selected;
			await run(() => onretire(tier, type));
			return;
		}

		const fields = new FormData(form);
		const name = String(fields.get('name')).trim();
		const description = String(fields.get('description')).trim();
		const priceKopecks = getPriceKopecks(String(fields.get('price')));
		const errors: typeof fieldErrors = {};
		if (!name) errors.name = 'Введите название уровня.';
		else if ([...name].length > 80) errors.name = 'Сократите название до 80 символов.';
		if ([...description].length > 5000) errors.description = 'Сократите описание до 5000 символов.';
		if (!Number.isSafeInteger(priceKopecks) || priceKopecks > 100_000_000) {
			errors.price = 'Укажите цену от 0 до 1 000 000 ₽ с точностью до копейки.';
		}
		fieldErrors = errors;

		const firstError = Object.keys(errors)[0];
		if (firstError) {
			await tick();
			(form.elements.namedItem(firstError) as HTMLElement).focus();
			return;
		}

		await run(() => onsave({ name, description, priceKopecks }, tier));
	}

	async function run(save: () => Promise<void>) {
		pending = true;
		try {
			await save();
		} catch (error) {
			requestError = errorMessage(error);
			return;
		} finally {
			pending = false;
		}
		close();
	}
</script>

<dialog
	{@attach openDialog}
	aria-labelledby="tier-dialog-title"
	aria-describedby={confirming ? 'tier-retire-detail' : undefined}
	oncancel={(event) => {
		event.preventDefault();
		close();
	}}
>
	<form novalidate aria-busy={pending} onsubmit={submit}>
		<h2 id="tier-dialog-title">{titles[action.type]}</h2>

		{#if action.type === 'delete'}
			<p id="tier-retire-detail">
				Уровень <strong>«{action.tier.name}»</strong> будет удалён безвозвратно.
			</p>
		{:else if action.type === 'archive'}
			<p id="tier-retire-detail">
				Уровень <strong>«{action.tier.name}»</strong> исчезнет с публичной страницы. Доступ у действующих
				подписчиков сохранится.
			</p>
		{:else}
			<div class="field">
				<label for="tier-name">Название</label>
				<input
					id="tier-name"
					name="name"
					value={tier?.name ?? ''}
					readonly={pending}
					required
					aria-invalid={fieldErrors.name ? true : undefined}
					aria-describedby="tier-name-hint"
				/>
				<p id="tier-name-hint" class={fieldErrors.name ? 'field-error' : 'field-hint'}>
					{fieldErrors.name ?? 'До 80 символов'}
				</p>
			</div>
			<div class="field">
				<label for="tier-description">Описание <span>· необязательно</span></label>
				<textarea
					id="tier-description"
					name="description"
					rows="4"
					value={tier?.description ?? ''}
					readonly={pending}
					aria-invalid={fieldErrors.description ? true : undefined}
					aria-describedby="tier-description-hint"></textarea>
				<p
					id="tier-description-hint"
					class={fieldErrors.description ? 'field-error' : 'field-hint'}
				>
					{fieldErrors.description ?? 'До 5000 символов'}
				</p>
			</div>
			<div class="field">
				<label for="tier-price">Стоимость в месяц, ₽</label>
				<input
					id="tier-price"
					name="price"
					inputmode="decimal"
					value={price}
					readonly={pending}
					required
					aria-invalid={fieldErrors.price ? true : undefined}
					aria-describedby={usedPrice ? 'tier-price-hint tier-used-price' : 'tier-price-hint'}
				/>
				<p id="tier-price-hint" class={fieldErrors.price ? 'field-error' : 'field-hint'}>
					{fieldErrors.price ?? 'От 0 до 1 000 000 ₽. Например, 150,50.'}
				</p>
				{#if usedPrice}
					<p id="tier-used-price" class="price-policy">
						Новая цена применится к новым подписчикам, действующие продлеваются по прежней
					</p>
				{/if}
			</div>
		{/if}

		{#if requestError}
			<p class="feedback error-feedback" role="alert">{requestError}</p>
		{/if}

		<div class="dialog-actions">
			<button type="button" class="button button-outline" aria-disabled={pending} onclick={close}>
				Отмена
			</button>
			{#if action.type === 'delete'}
				<button type="submit" class="button button-danger" aria-disabled={pending}>
					{pending ? 'Удаляем…' : 'Удалить безвозвратно'}
				</button>
			{:else if action.type === 'archive'}
				<button type="submit" class="button button-primary" aria-disabled={pending}>
					{pending ? 'Архивируем…' : 'Архивировать'}
				</button>
			{:else}
				<button type="submit" class="button button-primary" aria-disabled={pending}>
					{pending ? 'Сохраняем…' : action.type === 'edit' ? 'Сохранить' : 'Создать уровень'}
				</button>
			{/if}
		</div>
	</form>
</dialog>

<style>
	dialog {
		inset: 0;
		width: min(480px, calc(100% - 32px));
		max-height: calc(100% - 32px);
		margin: auto;
		padding: 24px;
		border: 1px solid var(--line);
		border-radius: 16px;
		background: var(--white);
		color: var(--ink);
		overflow-wrap: anywhere;
	}

	dialog::backdrop {
		background: #171a246e;
	}

	h2 {
		margin: 0 0 20px;
		font-size: 22px;
		font-weight: 800;
		line-height: 1.3;
		letter-spacing: -0.025em;
	}

	#tier-retire-detail {
		font-size: 15px;
		line-height: 1.6;
	}

	label span {
		color: var(--muted);
		font-weight: 500;
	}

	textarea {
		display: block;
		width: 100%;
		border: 1px solid #aeb6ca;
		border-radius: 9px;
		background: var(--white);
		padding: 12px 14px;
		font-size: 15px;
		line-height: 1.6;
		resize: vertical;
	}

	textarea:focus {
		border-color: var(--blue);
		outline: 2px solid var(--blue);
		outline-offset: 1px;
	}

	textarea[aria-invalid='true'] {
		border-color: var(--error);
	}

	.field-hint,
	.field-error {
		font-size: 12px;
	}

	.price-policy {
		margin: 9px 0 0;
		color: var(--muted);
		font-size: 13px;
		line-height: 1.55;
	}

	.dialog-actions {
		display: flex;
		flex-wrap: wrap;
		justify-content: flex-end;
		gap: 8px;
		margin-top: 24px;
	}

	.button-danger {
		background: var(--error);
		color: var(--white);
	}

	.button-danger:hover {
		background: #8e1d2c;
	}

	.button[aria-disabled='true'] {
		opacity: 0.65;
		cursor: wait;
	}

	@media (max-width: 440px) {
		dialog {
			padding: 20px;
		}

		input,
		textarea {
			font-size: 16px;
		}

		.dialog-actions > button {
			flex: 1 0 auto;
			padding-inline: 12px;
		}
	}
</style>
