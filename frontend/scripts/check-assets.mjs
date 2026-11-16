import { readdir, readFile } from 'node:fs/promises';
import { resolve } from 'node:path';
import { gzipSync } from 'node:zlib';

// All shipped assets are counted, including optional chart dependencies. Route
// JS grows with feed/discussion; LayerChart has a separate bounded lazy budget.
const budgets = { fonts: 150_000, coreJsGzip: 160_000, chartJsGzip: 180_000, cssGzip: 20_000 };
const sizes = { fonts: 0, coreJsGzip: 0, chartJsGzip: 0, cssGzip: 0 };
const manifest = JSON.parse(
	await readFile('.svelte-kit/output/client/.vite/manifest.json', 'utf8')
);
const chartEntry = 'src/lib/components/analytics/PaymentsChart.svelte';
if (!manifest[chartEntry]?.isDynamicEntry) throw new Error('Analytics chart must be a lazy entry');
function dependencies(entries, skip) {
	const seen = new Set();
	const files = new Set();
	function visit(key) {
		if (seen.has(key) || key === skip) return;
		seen.add(key);
		const item = manifest[key];
		if (!item) return;
		files.add(item.file);
		for (const child of [...(item.imports ?? []), ...(item.dynamicImports ?? [])]) visit(child);
	}
	entries.forEach(visit);
	return files;
}
const chartFiles = dependencies([chartEntry]);
const coreFiles = dependencies(
	Object.keys(manifest).filter((key) => manifest[key].isEntry),
	chartEntry
);
if (coreFiles.has(manifest[chartEntry].file)) throw new Error('Chart was included eagerly');
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
	} else if (path.endsWith('.js')) {
		const file = path.slice(resolve('build/client').length + 1).replaceAll('\\', '/');
		sizes[chartFiles.has(file) && !coreFiles.has(file) ? 'chartJsGzip' : 'coreJsGzip'] +=
			gzipSync(bytes).length;
	} else sizes.cssGzip += gzipSync(bytes).length;
}
for (const [kind, bytes] of Object.entries(sizes)) {
	console.log(`${kind}: ${bytes} B / ${budgets[kind]} B`);
	if (bytes > budgets[kind]) throw new Error(`${kind} asset budget exceeded`);
}
console.log(`Font files: ${fontFiles} (WOFF2, Latin and Cyrillic)`);
