export function passwordError(value: string): string {
	if ([...value].length < 12) return 'Пароль должен содержать не менее 12 символов.';
	if (new TextEncoder().encode(value).length > 256)
		return 'Пароль слишком длинный: максимум 256 байт в UTF-8.';
	if (!value.isWellFormed()) return 'Пароль содержит некорректный символ.';
	return '';
}
export function emailError(value: string): string {
	return value.length <= 254 && /^[^\s@]+@[^\s@]+$/.test(value.trim())
		? ''
		: 'Укажите корректный email.';
}
export function nicknameError(value: string): string {
	return /^[A-Za-z0-9_-]{3,30}$/.test(value)
		? ''
		: 'От 3 до 30 латинских букв, цифр, знаков _ или -.';
}
