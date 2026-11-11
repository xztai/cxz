import assert from 'node:assert/strict';
import { test } from 'node:test';
import { requestWithTimeout } from '../../src/lib/api/request.ts';

test('every fetch receives a fresh timeout and preserves caller cancellation', async (t) => {
	const signals: AbortSignal[] = [];
	t.mock.method(globalThis, 'fetch', (_request: Request, init: RequestInit) => {
		const signal = init.signal!;
		signals.push(signal);
		return new Promise<Response>((resolve, reject) => {
			if (signal.aborted) reject(signal.reason);
			else {
				signal.addEventListener('abort', () => reject(signal.reason), { once: true });
				resolve(new Response('{}'));
			}
		});
	});
	const controller = new AbortController();
	await requestWithTimeout(new Request('https://example.test/csrf', { signal: controller.signal }));
	controller.abort();
	await assert.rejects(
		requestWithTimeout(new Request('https://example.test/mutation', { signal: controller.signal }))
	);
	await requestWithTimeout(new Request('https://example.test/next'));
	assert.equal(signals.length, 3);
	assert.equal(signals[0].aborted, true);
	assert.equal(signals[2].aborted, false);
	assert.notEqual(signals[0], signals[2]);
});

test('a stalled fetch is cancelled when its timeout expires', async (t) => {
	// Retain the native scheduling implementation; shorten only the requested delay.
	const timeout = AbortSignal.timeout.bind(AbortSignal);
	const delays: number[] = [];
	t.mock.method(AbortSignal, 'timeout', (delay: number) => {
		delays.push(delay);
		return timeout(20);
	});
	t.mock.method(
		globalThis,
		'fetch',
		(_request: Request, init: RequestInit) =>
			new Promise<Response>((_resolve, reject) => {
				init.signal!.addEventListener('abort', () => reject(init.signal!.reason), { once: true });
			})
	);
	// Native AbortSignal timeout timers are unref'ed, so keep the test event loop alive.
	const keepAlive = setInterval(() => {}, 100);
	try {
		await assert.rejects(requestWithTimeout(new Request('https://example.test/stalled')), {
			name: 'TimeoutError'
		});
		assert.deepEqual(delays, [10_000]);
	} finally {
		clearInterval(keepAlive);
	}
});
