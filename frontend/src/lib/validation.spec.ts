import { describe, expect, it } from 'vitest';
import { emailError, nicknameError, passwordError } from './validation';

describe('API input boundaries', () => {
	it('counts Unicode characters rather than UTF-16 code units', () => {
		expect(passwordError('🌿'.repeat(11))).not.toBe('');
		expect(passwordError('🌿'.repeat(12))).toBe('');
	});
	it('applies the UTF-8 byte limit without trimming or normalizing', () => {
		expect(passwordError('я'.repeat(128))).toBe('');
		expect(passwordError('я'.repeat(129))).not.toBe('');
		expect(passwordError(' '.repeat(12))).toBe('');
		expect(passwordError('a'.repeat(256))).toBe('');
		expect(passwordError('a'.repeat(257))).not.toBe('');
	});
	it('rejects malformed Unicode', () => {
		expect(passwordError('a'.repeat(12) + '\ud800')).not.toBe('');
	});
	it('accepts only the API nickname alphabet and length', () => {
		expect(nicknameError('Art_3-D')).toBe('');
		for (const nickname of ['ab', 'a'.repeat(31), 'автор', 'art.name', 'art name']) {
			expect(nicknameError(nickname)).not.toBe('');
		}
	});
	it('validates email and its API length limit', () => {
		expect(emailError('artist@example.com')).toBe('');
		expect(emailError('artist@')).not.toBe('');
		expect(emailError('a'.repeat(250) + '@example.com')).not.toBe('');
	});
});
