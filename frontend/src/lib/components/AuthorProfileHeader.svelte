<script lang="ts">
	import type { Profile } from '#lib/api/client.js';

	let {
		profile
	}: {
		profile: Pick<Profile, 'cover' | 'avatar' | 'displayName' | 'nickname' | 'description'>;
	} = $props();
</script>

{#if profile.cover}
	<img class="cover" src={profile.cover.url} alt="Обложка страницы автора" />
{/if}
<div class="profile-heading">
	{#if profile.avatar}
		<img class="avatar" src={profile.avatar.url} alt="" width="88" height="88" />
	{/if}
	<div>
		<h1>{profile.displayName}</h1>
		{#if profile.nickname}<p class="nickname">@{profile.nickname}</p>{/if}
	</div>
</div>
{#if profile.description}<p class="description">{profile.description}</p>{/if}

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
</style>
