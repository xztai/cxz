export function billingDate(value: string | null): string {
	return value
		? new Intl.DateTimeFormat('ru-RU', { dateStyle: 'long', timeStyle: 'short' }).format(
				new Date(value)
			)
		: 'Не оплачен';
}
export const paymentStatus = {
	queued: 'Готовим оплату',
	pending: 'Ожидает оплаты',
	succeeded: 'Оплачен',
	canceled: 'Не оплачен',
	review: 'Уточняем результат'
} as const;
