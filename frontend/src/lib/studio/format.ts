const postDate = new Intl.DateTimeFormat('ru-RU', {
	day: 'numeric',
	month: 'short',
	year: 'numeric'
});

const rubles = new Intl.NumberFormat('ru-RU', {
	style: 'currency',
	currency: 'RUB',
	minimumFractionDigits: 0,
	maximumFractionDigits: 2
});

export function formatPostDate(publishedAt: string) {
	return postDate.format(new Date(publishedAt));
}

export function formatRubles(kopecks: number) {
	return rubles.format(kopecks / 100);
}
