import { readdir, readFile } from 'node:fs/promises';
import { resolve } from 'node:path';
import { gzipSync } from 'node:zlib';

// These are budgets for all shipped client assets, not initial-page transfer sizes.
const budgets = { fonts: 150_000, jsGzip: 120_000, cssGzip: 20_000 };
const sizes = { fonts: 0, jsGzip: 0, cssGzip: 0 };
let fontFiles = 0;
for (const entry of await readdir('build/client', { recursive: true, withFileTypes: true })) {
	if (!entry.isFile()) continue;
	const path = resolve(entry.parentPath, entry.name);
	if (/\.woff$/.test(path)) throw new Error(`Unexpected WOFF fallback: ${entry.name}`);
	if (!/\.(woff2|js|css)$/.test(path)) continue;
	const bytes = await readFile(path);
	if (path.endsWith('.woff2')) {
		sizes.fonts += bytes.length;
		fontFiles++;
	} else if (path.endsWith('.js')) sizes.jsGzip += gzipSync(bytes).length;
	else sizes.cssGzip += gzipSync(bytes).length;
}
for (const [kind, bytes] of Object.entries(sizes)) {
	console.log(`${kind}: ${bytes} B / ${budgets[kind]} B`);
	if (bytes > budgets[kind]) throw new Error(`${kind} asset budget exceeded`);
}
console.log(`Font files: ${fontFiles} (WOFF2, Latin and Cyrillic)`);
