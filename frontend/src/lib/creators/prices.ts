export function rublesToKopecks(value: string): number | null {
	const match = /^(0|[1-9]\d{0,6})(?:[.,](\d{1,2}))?$/.exec(value.trim());
	if (!match) return null;
	const amount = Number(match[1]) * 100 + Number((match[2] ?? '').padEnd(2, '0'));
	return amount <= 100_000_000 ? amount : null;
}
export function rubles(kopecks: number): string {
	return new Intl.NumberFormat('ru-RU', {
		style: 'currency',
		currency: 'RUB',
		maximumFractionDigits: 2
	}).format(kopecks / 100);
}
