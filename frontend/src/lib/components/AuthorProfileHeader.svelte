<script lang="ts">
	import type { Profile } from '#lib/api/client.js';

	let {
		profile,
		publicPage = false
	}: {
		profile: Pick<Profile, 'cover' | 'avatar' | 'displayName' | 'nickname' | 'description'>;
		publicPage?: boolean;
	} = $props();
	let initial = $derived(Array.from(profile.displayName.trim())[0]?.toLocaleUpperCase('ru-RU'));
</script>

<div class={{ 'public-profile': publicPage }}>
	{#if profile.cover}
		<img class="cover" src={profile.cover.url} alt="Обложка страницы автора" />
	{:else if publicPage}
		<div class="cover cover-placeholder" aria-hidden="true"></div>
	{/if}
	<div class="profile-heading">
		{#if profile.avatar}
			<img class="avatar" src={profile.avatar.url} alt="" width="88" height="88" />
		{:else if publicPage}
			<span class="avatar avatar-placeholder" aria-hidden="true">{initial}</span>
		{/if}
		<div class="identity">
			<h1>{profile.displayName}</h1>
			{#if profile.nickname}<p class="nickname">@{profile.nickname}</p>{/if}
		</div>
	</div>
	{#if profile.description}<p class="description">{profile.description}</p>{/if}
</div>

<style>
	.cover {
		width: 100%;
		max-height: 320px;
		object-fit: cover;
		border-radius: 16px;
		margin-bottom: 30px;
	}
	.profile-heading {
		display: flex;
		align-items: center;
		gap: 24px;
		overflow-wrap: anywhere;
	}
	.profile-heading h1 {
		margin-bottom: 10px;
	}
	.avatar {
		width: 88px;
		height: 88px;
		object-fit: cover;
		border-radius: 16px;
		flex: 0 0 auto;
	}
	.nickname {
		color: var(--muted);
		font-size: 14px;
	}
	.description {
		white-space: pre-wrap;
		overflow-wrap: anywhere;
		max-width: 720px;
		font-size: 16px;
		line-height: 1.8;
		margin-top: 32px;
	}
	.public-profile {
		--avatar-size: 96px;
		--avatar-overlap: 28px;
		--profile-gap: 20px;
	}
	.public-profile .cover {
		aspect-ratio: 4 / 1;
		min-height: 104px;
		margin: 0;
		background: var(--surface);
	}
	.public-profile .cover-placeholder {
		aspect-ratio: auto;
		height: clamp(104px, 13vw, 152px);
		background: var(--blue-soft);
	}
	.public-profile .profile-heading {
		position: relative;
		align-items: start;
		gap: var(--profile-gap);
		margin-top: calc(-1 * var(--avatar-overlap));
	}
	.public-profile .avatar {
		width: var(--avatar-size);
		height: var(--avatar-size);
		border: 4px solid var(--white);
	}
	.avatar-placeholder {
		display: grid;
		place-items: center;
		background: var(--blue-soft);
		color: var(--blue-hover);
		font-size: 34px;
		font-weight: 800;
	}
	.public-profile .identity {
		padding-top: calc(var(--avatar-overlap) + 12px);
		min-width: 0;
	}
	.public-profile h1 {
		margin-bottom: 2px;
	}
	.public-profile .description {
		max-width: 72ch;
		margin: 14px 0 0 calc(var(--avatar-size) + var(--profile-gap));
	}
	@media (max-width: 640px) {
		.public-profile {
			--avatar-size: 72px;
			--avatar-overlap: 18px;
			--profile-gap: 12px;
		}
		.public-profile .identity {
			padding-top: calc(var(--avatar-overlap) + 9px);
		}
		.public-profile .description {
			margin-left: 0;
			font-size: 15px;
		}
		.avatar-placeholder {
			font-size: 27px;
		}
	}
</style>
