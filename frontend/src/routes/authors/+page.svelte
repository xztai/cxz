<script lang="ts">
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import * as Field from '#lib/components/ui/field/index.js';
	import * as Empty from '#lib/components/ui/empty/index.js';
	import type { PageProps } from './$types';
	let { data }: PageProps = $props();
	function pageLink(offset: number) {
		return `/authors?${new URLSearchParams({ q: data.q, offset: String(offset) })}`;
	}
</script>

<svelte:head
	><title>Каталог авторов — etheyl</title><meta
		name="description"
		content="Найдите автора по имени, адресу или тому, что он создаёт."
	/></svelte:head
>
<section class="creator-workspace catalog-page page-inset">
	<div class="workspace-heading">
		<div>
			<h1>Найдите своего автора</h1>
			<p>Люди, за чьим творчеством хочется следить.</p>
		</div>
		{#if data.user}<Button href="/account/profile" variant="outline">Мой профиль автора</Button
			>{/if}
	</div>
	<form method="GET" action="/authors" class="catalog-search">
		<Field.FieldGroup
			><Field.Field
				><Field.FieldLabel for="author-search">Поиск авторов</Field.FieldLabel><Input
					id="author-search"
					name="q"
					value={data.q}
					maxlength={100}
					placeholder="Имя, никнейм или тема"
				/></Field.Field
			></Field.FieldGroup
		><Button type="submit">Найти</Button>{#if data.q}<Button href="/authors" variant="ghost"
				>Сбросить</Button
			>{/if}
	</form>
	{#if !data.authors.length}<Empty.Root
			><Empty.Header
				><Empty.Title>{data.q ? 'Авторов не нашлось' : 'Здесь появятся авторы'}</Empty.Title
				><Empty.Description
					>{data.q
						? 'Попробуйте другое имя, никнейм или более короткий запрос.'
						: 'Оформите профиль и станьте одним из первых авторов etheyl.'}</Empty.Description
				></Empty.Header
			><Empty.Content
				>{#if data.q}<Button href="/authors" variant="outline">Все авторы</Button>{:else}<Button
						href={data.user ? '/account/profile' : '/auth/register'}>Стать автором</Button
					>{/if}</Empty.Content
			></Empty.Root
		>
	{:else}<div class="author-directory">
			{#each data.authors as author (author.id)}<a
					class="directory-entry"
					href={`/authors/${author.nickname}`}
				>
					<div class="directory-cover">
						{#if author.cover}<img src={author.cover.url} alt="" loading="lazy" />{:else}<span
								class="cover-wordmark"
								aria-hidden="true">etheyl</span
							>{/if}
					</div>
					<div class="directory-body">
						<div class="directory-identity">
							{#if author.avatar}<img
									class="directory-avatar"
									src={author.avatar.url}
									alt=""
									loading="lazy"
								/>{:else}<span class="directory-avatar avatar-initial" aria-hidden="true"
									>{author.displayName.slice(0, 1).toUpperCase()}</span
								>{/if}
							<div>
								<h2>{author.displayName}</h2>
								<p>@{author.nickname}</p>
							</div>
						</div>
						<p class="directory-description">
							{author.description || 'Автор ещё не добавил описание.'}
						</p>
						<span class="directory-link"
							>Познакомиться с автором <span aria-hidden="true">↗</span></span
						>
					</div>
				</a>{/each}
		</div>{/if}
	{#if data.offset > 0 || data.hasMore}<nav
			class="catalog-pagination"
			aria-label="Страницы каталога"
		>
			{#if data.offset > 0}<Button href={pageLink(Math.max(0, data.offset - 24))} variant="outline"
					>Предыдущие</Button
				>{/if}{#if data.hasMore}<Button href={pageLink(data.offset + 24)} variant="outline"
					>Следующие авторы</Button
				>{/if}
		</nav>{/if}
</section>
