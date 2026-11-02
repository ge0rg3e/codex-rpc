<script lang="ts">
	import { page } from '$app/state';
	import { invalidateAll } from '$app/navigation';
	import { ArrowDown, ArrowUpRight, Check, ChevronDown, Copy, Download, GitFork, LogOut, Terminal, Trophy, Radio, Code2 } from '@lucide/svelte';
	import PresenceDemo from '#lib/PresenceDemo.svelte';
	import DevicePairing from '#lib/DevicePairing.svelte';
	import GitHubIcon from '#lib/GitHubIcon.svelte';
	import PlatformIcon from '#lib/PlatformIcon.svelte';
	import type { PageProps } from './$types';
	let { data }: PageProps = $props();
	const device = $derived(page.url.searchParams.get('device'));
	const repository = 'https://github.com/ge0rg3e/codex-rpc';
	const downloads = [
		{ platform: 'Windows', format: 'Installer · x64', file: 'codex-rpc-windows-amd64-installer.exe' },
		{ platform: 'macOS', format: 'Intel & Apple silicon', file: 'codex-rpc-macos-universal.dmg' },
		{ platform: 'Linux', format: 'Automatic setup · x64', file: 'codex-rpc-linux-amd64.AppImage' }
	];
	const metrics = [
		{ key: 'tokens', label: 'Tokens' },
		{ key: 'hours', label: 'Hours' },
		{ key: 'sessions', label: 'Sessions' },
		{ key: 'streak', label: 'Streak' }
	];
	const metric = $derived(metrics.find((item) => item.key === page.url.searchParams.get('rank'))?.key ?? 'tokens');
	const compact = (value: number) => new Intl.NumberFormat('en', { notation: 'compact', maximumFractionDigits: 1 }).format(value);
	let isPublic = $derived(data.user?.public ?? false);
	let savingVisibility = $state(false);
	let visibilityMessage = $state('');
	let signingOut = $state(false);
	let signOutError = $state('');
	let downloadOpen = $state(false);
	let downloadDropdown: HTMLDivElement;
	let platform = $derived(data.platform);
	const installCommand = $derived(`curl -fsSL ${page.url.origin}/install.sh | sh`);
	let copiedCommand = $state(false);
	let copyError = $state('');
	async function copyInstallCommand() {
		try { await navigator.clipboard.writeText(installCommand); copiedCommand = true; copyError = ''; }
		catch { copyError = 'Select and copy the command manually.'; }
	}

	async function signOut() {
		if (signingOut) return;
		signingOut = true;
		signOutError = '';
		try {
			const response = await fetch('/api/logout', { method: 'POST' });
			if (!response.ok) throw new Error('Sign out failed');
			window.location.reload();
		} catch {
			signOutError = 'Could not sign out. Please try again.';
			signingOut = false;
		}
	}

	async function toggleVisibility() {
		if (!data.user || savingVisibility) return;
		const next = !isPublic;
		savingVisibility = true;
		visibilityMessage = '';
		try {
			const response = await fetch('/api/profile', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ public: next })
			});
			if (!response.ok) {
				visibilityMessage = response.status === 401 ? 'Sign in again to change your visibility.' : 'Visibility could not be saved. Try again.';
				return;
			}
			isPublic = next;
			visibilityMessage = next ? 'Your profile is now public.' : 'Your profile is now private.';
			try {
				await invalidateAll();
			} catch {
				visibilityMessage = 'Visibility saved. Refresh to update the leaderboard.';
			}
		} catch {
			visibilityMessage = 'Visibility could not be saved. Check your connection and try again.';
		} finally {
			savingVisibility = false;
		}
	}
</script>

<svelte:head>
	<title>codex-rpc — Your coding session, on Discord</title>
	<meta name="description" content="Bring your Codex session to Discord. Share your activity, follow your stats, and find your place on the codex-rpc leaderboard." />
	<meta name="theme-color" content="#171717" />
</svelte:head>

{#if device && /^[A-F0-9]{8}$/.test(device)}
	{#key device}<DevicePairing code={device} login={data.user?.login} />{/key}
{/if}

<svelte:window onclick={(event) => {
	if (downloadOpen && event.target instanceof Node && !downloadDropdown?.contains(event.target)) downloadOpen = false;
}} onkeydown={(event) => {
	if (event.key === 'Escape' && downloadOpen) {
		downloadOpen = false;
		downloadDropdown?.querySelector('button')?.focus();
	}
}} />

<a href="#main" class="sr-only focus:not-sr-only focus:absolute focus:top-3 focus:left-3 focus:z-10 focus:rounded-lg focus:bg-accent focus:p-3 focus:text-canvas">Skip to content</a>
<div class="mx-auto max-w-[1200px] px-6 sm:px-10">
	<header class="flex min-h-24 flex-wrap items-center justify-between gap-x-6 gap-y-2 border-b border-line py-3">
		<a href="/" class="flex items-center text-lg font-medium tracking-tight">
			<img src="/logo.png" class="h-auto w-[10%] mr-1" />
			codex<span class="font-normal text-muted">-rpc</span>
		</a>
		<nav aria-label="Main navigation" class="flex items-center gap-5 text-sm sm:gap-8">
			<a href="#leaderboard" class="text-muted hover:text-ink">Leaderboard</a>
			<a href={repository} target="_blank" rel="noreferrer" class="hidden items-center gap-1.5 text-muted hover:text-ink sm:flex">GitHub <ArrowUpRight size={14} /></a>
			{#if data.user}
				<div class="flex items-center rounded-lg border border-line">
					<a href={`https://github.com/${data.user.login}`} class="max-w-24 truncate rounded-l-lg py-2 pr-1 pl-3 text-sm sm:max-w-36">@{data.user.login}</a>
					<button type="button" onclick={signOut} disabled={signingOut} aria-label={signingOut ? 'Signing out' : 'Sign out'} title={signingOut ? 'Signing out…' : 'Sign out'} class="flex min-h-10 min-w-10 items-center justify-center rounded-r-lg text-muted hover:bg-panel hover:text-ink disabled:cursor-wait disabled:opacity-60">
						<LogOut size={15} />
					</button>
				</div>
			{:else}<a href="/auth/github" aria-label="Sign in with GitHub" class="flex items-center gap-2 rounded-lg border border-line bg-panel px-3 py-2 hover:bg-panel-raised"
					><GitHubIcon /> Sign in with GitHub</a
				>{/if}
		</nav>
	</header>
	{#if signOutError}<p role="alert" class="mt-3 text-right text-sm text-[#f0aaa6]">{signOutError}</p>{/if}
	<main id="main">
		<section aria-labelledby="hero-title" class="grid items-center gap-14 py-20 md:grid-cols-[1.1fr_1fr] md:gap-10 md:py-28 lg:gap-20">
			<div>
				<h1 id="hero-title" class="max-w-[560px] text-[clamp(2.9rem,5.4vw,4.5rem)] leading-[1.06] font-medium tracking-[-0.055em]">Codex on Discord</h1>
				<p class="mt-6 max-w-[370px] text-base leading-7 text-muted">Give your Codex sessions a place on Discord. Your activity, your stats, and a little personality. All in your presence.</p>
				<div id="downloads" class="mt-9 scroll-mt-8">
					{#if platform === 'Linux'}
						<div class="max-w-full rounded-xl border border-line bg-panel p-4">
							<div class="mb-3 flex items-center justify-between gap-4 text-sm"><span class="flex items-center gap-2"><PlatformIcon platform="Linux" /> Install on Linux</span><button type="button" onclick={copyInstallCommand} aria-label="Copy Linux install command" class="flex min-h-9 items-center gap-2 rounded-md px-2 text-accent hover:bg-panel-raised">{#if copiedCommand}<Check size={15} /> Copied{:else}<Copy size={15} /> Copy{/if}</button></div>
							<code class="block text-xs leading-6 break-all select-all">{installCommand}</code>
						</div>
						<p class="mt-3 text-xs leading-5 text-muted">Detects your distribution and enables start at sign-in.</p>
						<p role="status" class="text-xs text-muted">{copyError}</p>
						<a href={`${repository}/blob/main/apps/client/scripts/install.sh`} target="_blank" rel="noreferrer" class="mt-2 inline-flex items-center gap-1 text-xs text-accent underline underline-offset-4">Read the source <ArrowUpRight size={13} /></a>
					{:else if platform === 'Windows' || platform === 'macOS'}
						{@const download = downloads.find((item) => item.platform === platform)!}
						<a href={`${repository}/releases/latest/download/${download.file}`} class="inline-flex items-center gap-3 rounded-xl bg-accent px-5 py-3.5 text-sm font-semibold text-[#211b32] hover:bg-[#d4c8ff]"><PlatformIcon {platform} dark /> Download for {platform} <Download size={16} /></a>
						<p class="mt-3 text-xs text-muted">{download.format}. Free & open source.</p>
					{:else}
						<p class="text-sm text-muted">Choose your desktop platform to get codex-rpc.</p>
					{/if}
					<div bind:this={downloadDropdown} class="relative w-fit max-w-full">
						<button type="button" aria-expanded={downloadOpen} aria-controls="platform-downloads" onclick={() => downloadOpen = !downloadOpen} class="mt-3 flex min-h-10 items-center gap-2 text-xs text-muted hover:text-ink">{platform === 'unknown' ? 'Choose platform' : 'Other platforms'} <ChevronDown size={14} class={`transition-transform duration-200 motion-reduce:transition-none ${downloadOpen ? 'rotate-180' : ''}`} /></button>
						<div id="platform-downloads" inert={!downloadOpen} class={`absolute top-full left-0 z-20 mt-2 w-72 max-w-[calc(100vw-3rem)] origin-top overflow-hidden rounded-xl border border-line bg-panel p-1.5 transition-[opacity,transform,visibility] duration-200 ease-out motion-reduce:transition-none ${downloadOpen ? 'visible translate-y-0 opacity-100' : 'invisible -translate-y-2 opacity-0'}`}>
						{#each downloads as download}
							<a href={download.platform === 'Linux' ? '#downloads' : `${repository}/releases/latest/download/${download.file}`} onclick={(event) => { downloadOpen = false; if (download.platform === 'Linux') { event.preventDefault(); platform = 'Linux'; } }} aria-label={`Download codex-rpc for ${download.platform}: ${download.format}`} class="flex items-center justify-between gap-4 rounded-lg px-3 py-3 text-sm hover:bg-panel-raised">
								<span class="flex items-center gap-3"><PlatformIcon platform={download.platform} /><span><span class="block font-medium">{download.platform}</span><span class="mt-1 block text-xs text-muted">{download.format}</span></span></span><Download size={16} class="shrink-0 text-accent" />
							</a>
						{/each}
						</div>
					</div>
				</div>
				<a href="#leaderboard" class="mt-5 inline-flex items-center gap-2 py-2 text-sm text-muted hover:text-ink">Meet the builders <ArrowDown size={15} /></a>
			</div>

			<PresenceDemo />
		</section>
		<div class="flex flex-wrap items-center justify-between gap-x-8 gap-y-5 border-y border-line py-5 text-sm text-muted">
			<span class="flex items-center gap-2.5"><Radio size={16} class="text-accent" /> Activity that follows your session</span><span class="flex items-center gap-2.5"
				><Check size={16} class="text-accent" /> Tokens, tools & time at a glance</span
			><span class="flex items-center gap-2.5"><GitFork size={16} class="text-accent" /> Built in the open</span>
		</div>
		<section id="leaderboard" aria-labelledby="leaderboard-title" class="scroll-mt-8 py-20 sm:py-24">
			<div class="flex flex-wrap items-end justify-between gap-6">
				<div>
					<h2 id="leaderboard-title" class="text-3xl font-medium tracking-[-0.04em] sm:text-4xl">Good company. Great builds.</h2>
					<p class="mt-3 text-sm leading-6 text-muted">The people putting their Codex sessions to work.</p>
				</div>
				<div class="flex flex-col items-start gap-2 sm:items-end">
					{#if data.user}
						<button type="button" role="switch" aria-checked={isPublic} aria-label="Public profile" aria-describedby="profile-visibility-help" disabled={savingVisibility} onclick={toggleVisibility} class="flex min-h-11 items-center gap-3 py-2 text-sm hover:text-accent disabled:cursor-wait disabled:opacity-60">
							<span>{savingVisibility ? 'Saving…' : isPublic ? 'Profile public' : 'Profile private'}</span>
							<span aria-hidden="true" class={`flex h-5 w-9 items-center rounded-full p-0.5 ${isPublic ? 'bg-accent' : 'bg-[#55505e]'}`}><span class={`size-4 rounded-full ${isPublic ? 'translate-x-4 bg-[#211b32]' : 'bg-ink'}`}></span></span>
						</button>
						<p id="profile-visibility-help" class="text-xs text-muted">Show your profile on the leaderboard.</p>
					{:else}
						<a href="/auth/github" class="rounded-xl border border-line bg-panel px-4 py-3 text-sm hover:bg-panel-raised">Sign in to manage visibility</a>
					{/if}
					<p role="status" class="max-w-xs text-xs text-muted">{visibilityMessage}</p>
				</div>
			</div>
			<div class="mt-8 overflow-hidden rounded-2xl border border-line">
				<div class="flex flex-wrap items-center justify-between gap-4 border-b border-line bg-panel px-4 py-4 sm:px-6">
					<nav aria-label="Rank leaderboard by" class="flex gap-1">
						{#each metrics as item}<a
								href={`?rank=${item.key}#leaderboard`}
								aria-current={metric === item.key ? 'page' : undefined}
								class={`rounded-lg px-3 py-2 text-xs font-medium sm:px-4 ${metric === item.key ? 'bg-[#383142] text-[#e0d6fa]' : 'text-muted hover:bg-panel-raised hover:text-ink'}`}
								>{item.label}</a
							>{/each}
					</nav>
					<span class="text-xs text-muted">{metric === 'streak' ? 'Current streak' : 'All time'}</span>
				</div>
				<div class="overflow-x-auto">
					<table class="w-full min-w-[610px] border-collapse text-left text-sm">
						<caption class="sr-only">Public codex-rpc members ranked by {metric}</caption>
						<thead class="border-b border-line text-xs text-muted"
							><tr
								><th scope="col" class="w-16 px-6 py-4 font-normal">Rank</th><th scope="col" class="px-4 py-4 font-normal">Builder</th>{#each metrics as item}<th
										scope="col"
										aria-sort={metric === item.key ? 'descending' : 'none'}
										class={`px-5 py-4 text-right font-normal ${metric === item.key ? 'text-accent' : ''}`}>{item.label}</th
									>{/each}</tr
							></thead
						>
						<tbody
							>{#each data.entries as entry, index}<tr class="border-b border-line last:border-0 hover:bg-panel"
									><td class="px-6 py-5 text-muted tabular-nums"
										>{#if index === 0}<Trophy size={17} class="text-accent" /><span class="sr-only">1</span>{:else}{String(index + 1).padStart(2, '0')}{/if}</td
									><td class="px-4 py-5"
										><a href={`https://github.com/${entry.login}`} target="_blank" rel="noreferrer" class="flex items-center gap-3 font-medium hover:text-accent"
											>{#if entry.avatar}<img src={entry.avatar} alt="" width="32" height="32" class="size-8 rounded-full bg-panel-raised" loading="lazy" />{:else}<span
													class="flex size-8 items-center justify-center rounded-full bg-panel-raised text-xs">{entry.login.slice(0, 2)}</span
												>{/if}{entry.login}</a
										></td
									><td class="px-5 py-5 text-right tabular-nums">{compact(entry.tokens)}</td><td class="px-5 py-5 text-right text-muted tabular-nums"
										>{entry.hours.toLocaleString('en', { maximumFractionDigits: 1 })}h</td
									><td class="px-5 py-5 text-right text-muted tabular-nums">{compact(entry.sessions)}</td><td class="px-5 py-5 text-right text-muted tabular-nums">{entry.streak}d</td
									></tr
								>{/each}</tbody
						>
					</table>
				</div>
				{#if data.entries.length === 0}<div class="flex flex-col items-center px-6 py-12 text-center">
						<span class="mb-4 flex size-11 items-center justify-center rounded-xl border border-line bg-panel"><Trophy size={20} class="text-accent" /></span>
						<h3 class="font-medium">{data.unavailable ? 'The leaderboard is taking a break.' : 'The next session could be yours.'}</h3>
						<p class="mt-2 max-w-sm text-sm leading-6 text-muted">
							{data.unavailable
								? 'Rankings could not load. Try refreshing in a moment.'
								: 'Connect your GitHub account in the desktop app and enable your public profile to join the leaderboard.'}
						</p>
						{#if data.unavailable}<a href={`?rank=${metric}#leaderboard`} data-sveltekit-reload class="mt-5 text-sm text-accent underline underline-offset-4">Try again</a>{/if}
					</div>{/if}
				<div class="flex flex-wrap items-center justify-between gap-3 border-t border-line bg-panel px-6 py-4 text-xs text-muted">
					<span>Make your sessions count. Join from the desktop app.</span><a href="#downloads" class="flex items-center gap-1.5 text-accent hover:underline"
						>Get codex-rpc <ArrowUpRight size={14} /></a
					>
				</div>
			</div>
		</section>
	</main>
	<footer class="flex flex-wrap items-center justify-between gap-6 border-t border-line py-8 text-xs text-muted">
		<div class="flex items-center gap-3"><img src="/logo.png" class="h-auto w-[6%] mr-1" /><span>codex-rpc <span class="mx-2 text-line">/</span> A presence for your process.</span></div>
		<div class="flex items-center gap-6">
			<a href={repository} target="_blank" rel="noreferrer" class="hover:text-ink">Source code</a><a href={`${repository}/issues`} target="_blank" rel="noreferrer" class="hover:text-ink"
				>Feedback</a
			><a href={`${repository}/blob/main/LICENSE`} target="_blank" rel="noreferrer" class="hover:text-ink">License</a>
		</div>
		<p class="w-full text-[11px] text-muted">An independent community project. Not affiliated with OpenAI or Discord.</p>
	</footer>
</div>
