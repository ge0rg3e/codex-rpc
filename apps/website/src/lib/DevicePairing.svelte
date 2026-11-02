<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import GitHubIcon from '#lib/GitHubIcon.svelte';

	let { code, login }: { code: string; login?: string } = $props();
	let dialog: HTMLDialogElement;
	let busy = $state(false);
	let approved = $state(false);
	let message = $state('');

	$effect(() => { dialog.showModal(); });

	async function close() {
		if (busy) return;
		const url = new URL(page.url.href);
		url.searchParams.delete('device');
		await goto(url, { replaceState: true });
	}

	async function approve() {
		if (busy || approved || !login) return;
		busy = true;
		message = '';
		try {
			const response = await fetch('/api/device/approve', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ code })
			});
			if (response.ok) approved = true;
			else if (response.status === 410) message = 'This request expired or was already used. Start a new connection in the desktop app.';
			else if (response.status === 401) message = 'Your session expired. Sign in again to continue.';
			else message = 'Could not approve the connection. Try again.';
		} catch {
			message = 'Could not reach the server. Try again.';
		} finally {
			busy = false;
		}
	}
</script>

<dialog bind:this={dialog} oncancel={(event) => { event.preventDefault(); void close(); }} aria-labelledby="pairing-title" class="m-auto w-[calc(100%-3rem)] max-w-md rounded-2xl border border-line bg-panel p-6 text-ink backdrop:bg-black/65">
	<h2 id="pairing-title" class="text-xl font-semibold">{approved ? 'Connection approved' : 'Connect codex-rpc?'}</h2>
	{#if approved}
		<p class="mt-3 text-sm leading-6 text-muted">Return to the desktop app to finish connecting.</p>
	{:else}
		<p class="mt-3 text-sm leading-6 text-muted">Only approve if you started this connection in the desktop app and its code matches.</p>
		<p class="my-4 rounded-lg bg-panel-raised px-4 py-3 font-mono tracking-widest">{code}</p>
		<p class="text-sm text-muted">{login ? `Connect using @${login}.` : 'Sign in with GitHub to review and approve this request.'}</p>
	{/if}
	{#if message}<p role="alert" class="mt-4 text-sm text-muted">{message}</p>{/if}
	<div class="mt-6 flex flex-wrap justify-end gap-3">
		<button type="button" onclick={close} disabled={busy} class="rounded-lg border border-line px-4 py-2 text-sm hover:bg-panel-raised">{approved ? 'Done' : 'Decline'}</button>
		{#if !approved}
			{#if !login || message.startsWith('Your session expired')}
				<a href={`/auth/github?device=${code}`} class="inline-flex items-center gap-2 rounded-lg bg-accent px-4 py-2 text-sm font-medium text-[#211b32]"><GitHubIcon /> Sign in with GitHub</a>
			{:else}
				<button type="button" onclick={approve} disabled={busy} class="rounded-lg bg-accent px-4 py-2 text-sm font-medium text-[#211b32]">{busy ? 'Approving…' : 'Approve connection'}</button>
			{/if}
		{/if}
	</div>
</dialog>
