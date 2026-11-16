<script lang="ts">
	import { cn, type WithElementRef } from '#lib/utils.js';
	import type { HTMLOptgroupAttributes } from 'svelte/elements';

	let {
		ref = $bindable(null),
		class: className,
		children,
		...restProps
	}: WithElementRef<HTMLOptgroupAttributes> = $props();
</script>

<optgroup
	{@attach (node) => {
		ref = node;
		return () => {
			ref = null;
		};
	}}
	data-slot="native-select-optgroup"
	class={cn('bg-[Canvas] text-[CanvasText]', className)}
	{...restProps}
>
	{@render children?.()}
</optgroup>
