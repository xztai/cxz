<script lang="ts">
	let { name, size = 20 }: { name: keyof typeof paths; size?: number } = $props();
	const paths = {
		plus: 'M12 5v14M5 12h14',
		arrow: 'M4 12h16m-6-6 6 6-6 6',
		eye: 'M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7S2 12 2 12Zm13 0a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z',
		'eye-off':
			'm3 3 18 18M10.6 5.1 12 5c6.5 0 10 7 10 7a20 20 0 0 1-3 3.8M6.3 6.3A20 20 0 0 0 2 12s3.5 7 10 7a11 11 0 0 0 5-1.2M10 10a3 3 0 0 0 4 4',
		lock: 'M7 10V7a5 5 0 0 1 10 0v3M5 10h14v11H5V10Zm7 5v2',
		check: 'm5 12 4 4L19 6',
		mail: 'M3 5h18v14H3V5Zm0 1 9 7 9-7',
		overview: 'M3 3h7v7H3zm11 0h7v7h-7zM3 14h7v7H3zm11 0h7v7h-7z',
		profile: 'M16 8a4 4 0 1 1-8 0 4 4 0 0 1 8 0ZM4 21v-2a5 5 0 0 1 5-5h6a5 5 0 0 1 5 5v2',
		layers: 'm12 3 9 5-9 5-9-5zm-9 9 9 5 9-5m-18 4 9 5 9-5',
		post: 'M6 3h9l4 4v14H6zm8 0v5h5m-9 4h5m-5 4h5',
		analytics: 'M4 3v17h17M9 16v-4m5 4V8m5 8V4',
		menu: 'M4 6h16M4 12h16M4 18h16',
		chevron: 'm6 9 6 6 6-6',
		external: 'M13 5h6v6m0-6-9 9m6-1v6H5V8h6',
		wallet: 'M20 8H5a2 2 0 0 1 0-4h12v4M3 6v12a2 2 0 0 0 2 2h15V8m0 4h-5a2 2 0 0 0 0 4h5',
		users:
			'M15 7a3 3 0 1 1-6 0 3 3 0 0 1 6 0ZM6 20v-2a4 4 0 0 1 4-4h4a4 4 0 0 1 4 4v2M4 18H2v-2a4 4 0 0 1 4-4m12 0a4 4 0 0 1 4 4v2h-2'
	};
</script>

<svg
	width={size}
	height={size}
	viewBox="0 0 24 24"
	fill="none"
	stroke="currentColor"
	stroke-width="1.7"
	stroke-linecap="round"
	stroke-linejoin="round"
	aria-hidden="true"><path d={paths[name]} /></svg
>
