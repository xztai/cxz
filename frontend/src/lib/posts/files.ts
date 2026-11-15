const number = new Intl.NumberFormat('ru-RU', { maximumFractionDigits: 1 });

export function fileSize(bytes: number): string {
	if (bytes < 1024) return `${bytes} Б`;
	if (bytes < 1024 * 1024) return `${number.format(bytes / 1024)} КБ`;
	return `${number.format(bytes / 1024 / 1024)} МБ`;
}
