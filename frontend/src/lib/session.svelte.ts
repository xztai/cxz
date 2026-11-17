import { createContext } from 'svelte';
import { api, type User } from '#lib/api/client.js';

export class Session {
	user = $state.raw<User | null>(null);
	ready = $state(false);
	#revision = 0;

	setUser(user: User | null) {
		this.#revision++;
		this.user = user;
		this.ready = true;
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
