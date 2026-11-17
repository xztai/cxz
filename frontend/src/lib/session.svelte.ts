import { createContext } from 'svelte';
import { api, type User } from '#lib/api/client.js';

type Verification = {
	email: string;
	resendAt: number;
	sent: boolean;
};
const verificationKey = 'etheyl.verification';

export class Session {
	user = $state.raw<User | null>(null);
	ready = $state(false);
	verification = $state.raw<Verification | null>(null);
	#revision = 0;

	restoreVerification() {
		if (this.verification) return;
		try {
			const saved: unknown = JSON.parse(sessionStorage.getItem(verificationKey) ?? 'null');
			if (
				saved &&
				typeof saved === 'object' &&
				'email' in saved &&
				typeof saved.email === 'string' &&
				'resendAt' in saved &&
				typeof saved.resendAt === 'number' &&
				Number.isFinite(saved.resendAt) &&
				'sent' in saved &&
				typeof saved.sent === 'boolean'
			) {
				this.verification = { email: saved.email, resendAt: saved.resendAt, sent: saved.sent };
			}
		} catch {
			// The current page still works when browser storage is unavailable.
		}
	}

	setVerification(email: string, sent = false, delay = 60) {
		this.restoreVerification();
		const previous = this.verification;
		this.verification = {
			email: email.trim().toLowerCase(),
			resendAt: sent
				? Date.now() + delay * 1000
				: previous?.email === email.trim().toLowerCase()
					? previous.resendAt
					: 0,
			sent: sent || (previous?.email === email.trim().toLowerCase() && previous.sent)
		};
		this.saveVerification();
	}

	waitForResend(seconds: number) {
		if (!this.verification) return;
		this.verification = { ...this.verification, resendAt: Date.now() + seconds * 1000 };
		this.saveVerification();
	}

	private saveVerification() {
		try {
			if (this.verification)
				sessionStorage.setItem(verificationKey, JSON.stringify(this.verification));
			else sessionStorage.removeItem(verificationKey);
		} catch {
			// Keep the state in this browser tab's component context as a fallback.
		}
	}

	setUser(user: User | null) {
		this.#revision++;
		this.user = user;
		this.ready = true;
		if (user) {
			this.verification = null;
			this.saveVerification();
		}
	}

	async refresh() {
		const revision = ++this.#revision;
		try {
			const { user } = await api.me();
			// A response started before login/logout must not replace the new session.
			if (revision === this.#revision) this.setUser(user);
		} catch {
			if (revision === this.#revision) this.ready = true;
		}
	}
}

export const [getSession, setSession] = createContext<Session>();
