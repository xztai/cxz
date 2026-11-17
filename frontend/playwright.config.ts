import { defineConfig } from '@playwright/test';

const previewURL = 'http://127.0.0.1:4173';
const externalURL = process.env.E2E_BASE_URL;

export default defineConfig({
	use: { baseURL: externalURL || previewURL },
	webServer: externalURL
		? undefined
		: {
				command: 'pnpm build && pnpm preview --host 127.0.0.1 --port 4173 --strictPort',
				url: previewURL
			},
	testMatch: '**/*.e2e.{ts,js}'
});
