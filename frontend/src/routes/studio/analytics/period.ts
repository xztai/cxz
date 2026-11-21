export const presets = [7, 30, 90] as const;
export const minDate = '2000-01-01';
export const maxDate = '9998-12-31';

const dayMs = 86_400_000;

function dateOnly(date: Date) {
	return date.toISOString().slice(0, 10);
}

function validDate(value: string | null): value is string {
	if (!value || !/^\d{4}-\d{2}-\d{2}$/.test(value) || value < minDate || value > maxDate) {
		return false;
	}
	const date = new Date(value);
	return Number.isFinite(date.getTime()) && dateOnly(date) === value;
}

export function validPeriod(from: string, to: string) {
	return (
		validDate(from) &&
		validDate(to) &&
		from <= to &&
		Date.parse(to) - Date.parse(from) <= 365 * dayMs
	);
}

export function analyticsPeriod(params: URLSearchParams) {
	const from = params.get('from');
	const to = params.get('to');
	if (from && to && validPeriod(from, to)) {
		return { period: { from, to }, preset: null, invalidPeriod: false };
	}

	const invalidPeriod = from !== null || to !== null;
	const days = params.get('days');
	const preset = (!invalidPeriod && presets.find((value) => String(value) === days)) || 30;
	const today = dateOnly(new Date());
	return {
		period: { from: dateOnly(new Date(Date.parse(today) - (preset - 1) * dayMs)), to: today },
		preset,
		invalidPeriod
	};
}
