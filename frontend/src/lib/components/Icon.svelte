<script lang="ts">
	let {
		name,
		size = 20
	}: { name: 'arrow' | 'eye' | 'eye-off' | 'lock' | 'check' | 'mail' | 'plus'; size?: number } =
		$props();
	const paths = {
		plus: 'M12 5v14M5 12h14',
		arrow: 'M4 12h16m-6-6 6 6-6 6',
		eye: 'M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7S2 12 2 12Zm13 0a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z',
		'eye-off':
			'm3 3 18 18M10.6 5.1 12 5c6.5 0 10 7 10 7a20 20 0 0 1-3 3.8M6.3 6.3A20 20 0 0 0 2 12s3.5 7 10 7a11 11 0 0 0 5-1.2M10 10a3 3 0 0 0 4 4',
		lock: 'M7 10V7a5 5 0 0 1 10 0v3M5 10h14v11H5V10Zm7 5v2',
		check: 'm5 12 4 4L19 6',
		mail: 'M3 5h18v14H3V5Zm0 1 9 7 9-7'
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
