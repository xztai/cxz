<script lang="ts">
	let { compact = false }: { compact?: boolean } = $props();
</script>

<div
	class={['creator-collage', { 'collage-compact': compact }]}
	aria-label="Визуальные эскизы творческих проектов"
>
	<div class="collage-orbit" aria-hidden="true"></div>
	<article class="creative-piece photo-piece">
		<div class="piece-topline"><span>фотодневник</span><span>№ 07</span></div>
		<img
			src="/images/coast.jpg"
			width="900"
			height="675"
			alt="Скалистый берег и тихий океан на плёночной фотографии"
			fetchpriority={compact ? 'auto' : 'high'}
		/>
		<h2>Медленные<br />маршруты</h2>
		<p>Замечать больше. Спешить меньше.</p>
	</article>
	<article class="creative-piece sound-piece">
		<span>разговоры о важном</span>
		<div class="sound-art" aria-hidden="true"><i></i><i></i><i></i><i></i><i></i></div>
		<h2>не тишина</h2>
		<p>У каждой мысли есть голос.</p>
	</article>
	<article class="creative-piece text-piece">
		<span>записки на полях</span>
		<h2>А что,<br />если?</h2>
		<p>Всё интересное<br />начинается<br />с этого вопроса.</p>
		<span class="text-piece-bottom"
			>просто продолжай <svg viewBox="0 0 24 24" aria-hidden="true"
				><path d="M4 12h16m-6-6 6 6-6 6" /></svg
			></span
		>
	</article>
	<span class="collage-caption">Эскизы проектов. Вдохновение для будущего.</span>
</div>
