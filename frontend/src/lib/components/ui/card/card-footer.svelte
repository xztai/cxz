<script lang="ts">
	import { cn, type WithElementRef } from '#lib/utils.js';
	import type { HTMLAttributes } from 'svelte/elements';

	let {
		ref = $bindable(null),
		class: className,
		children,
		...restProps
	}: WithElementRef<HTMLAttributes<HTMLDivElement>> = $props();
	function attachRef(element: HTMLElement) {
		ref = element;
		return () => {
			if (ref === element) ref = null;
		};
	}
</script>

<div
	{@attach attachRef}
	data-slot="card-footer"
	class={cn(
		'bg-muted/50 rounded-b-xl border-t p-(--card-spacing) mt-auto flex items-center',
		className
	)}
	{...restProps}
>
	{@render children?.()}
</div>
