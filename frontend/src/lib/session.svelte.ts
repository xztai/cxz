import { createContext, onDestroy } from 'svelte';
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
	verified = $state(false);
	studioStatus = $state<'ready' | 'pending' | 'retry'>('ready');
	#revision = 0;
	#studioRevision = 0;
	#studioRecovery?: { ownerId: string | undefined };
	#lastStudioRecoveryOwner?: string;

	constructor(private invalidateStudio: () => Promise<void>) {}

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
		const changed = this.user?.id !== user?.id;
		this.user = user;
		this.verified = true;
		this.ready = true;
		if (user) {
			this.verification = null;
			this.saveVerification();
		}
		const recovery = this.#studioRecovery;
		this.#studioRecovery = undefined;
		if (changed) {
			this.#lastStudioRecoveryOwner = undefined;
		} else if (recovery) {
			// A repeated rejection with the same verified account should offer a manual retry.
			if (recovery.ownerId !== undefined && recovery.ownerId === this.#lastStudioRecoveryOwner) {
				this.studioStatus = 'retry';
				return;
			}
			this.#lastStudioRecoveryOwner = recovery.ownerId;
		}
		if (changed || recovery) void this.#reloadStudio();
	}

	async #reloadStudio() {
		const revision = ++this.#studioRevision;
		this.studioStatus = 'pending';
		try {
			await this.invalidateStudio();
			if (revision === this.#studioRevision) this.studioStatus = 'ready';
		} catch {
			if (revision === this.#studioRevision) this.studioStatus = 'retry';
		}
	}

	recheckStudio(ownerId?: string) {
		if (ownerId !== undefined && ownerId !== this.user?.id) return;
		if (this.#studioRecovery && this.studioStatus === 'pending') return;
		// A private request can fail while invalidation is still loading child data.
		this.#studioRevision++;
		this.#studioRecovery = { ownerId };
		this.studioStatus = 'pending';
		void this.refresh();
	}

	isStudioOwner(ownerId: string) {
		return this.studioStatus === 'ready' && this.user?.id === ownerId;
	}

	async refresh() {
		const revision = ++this.#revision;
		try {
			const { user } = await api.me();
			// A response started before login/logout must not replace the new session.
			if (revision === this.#revision) this.setUser(user);
		} catch {
			if (revision === this.#revision) {
				this.ready = true;
				if (this.#studioRecovery) this.studioStatus = 'retry';
			}
		}
	}
}

export const [getSession, setSession] = createContext<Session>();

export function studioEditorIsActive(ownerId: string) {
	const session = getSession();
	let mounted = true;
	onDestroy(() => (mounted = false));
	return () => mounted && session.isStudioOwner(ownerId);
}
