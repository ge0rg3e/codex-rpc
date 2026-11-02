<script lang="ts">
	import { onMount } from 'svelte';

	// Keep labels, asset names, formatting and frame conditions aligned with
	// apps/client/activity.go and discordActivityAt in apps/client/discord.go.
	const labels = {
		idle: 'Idle', thinking: 'Thinking', coding: 'Writing a response',
		editing: 'Editing files', running: 'Using tools', waiting: 'Waiting for your input',
		success: 'Task complete', error: 'Encountered an error', stopped: 'Session paused'
	} as const;
	type Step = { state: keyof typeof labels; tokens: number; prompts: number; tools: number; tool?: string; seconds: number };
	// Illustrative log snapshots, not a connection to the visitor's local client.
	// Terminal states fall back to idle after a minute; a new event can change them sooner.
	const steps: Step[] = [
		{ state: 'thinking', tokens: 1200, prompts: 1, tools: 0, seconds: 20 },
		{ state: 'editing', tokens: 5600, prompts: 1, tools: 1, tool: 'apply_patch', seconds: 20 },
		{ state: 'running', tokens: 7200, prompts: 1, tools: 2, tool: 'exec_command', seconds: 20 },
		{ state: 'error', tokens: 7400, prompts: 1, tools: 2, seconds: 20 },
		{ state: 'thinking', tokens: 8600, prompts: 1, tools: 2, seconds: 20 },
		{ state: 'editing', tokens: 11200, prompts: 1, tools: 3, tool: 'apply_patch', seconds: 20 },
		{ state: 'waiting', tokens: 11200, prompts: 1, tools: 4, tool: 'request_user_input', seconds: 20 },
		{ state: 'coding', tokens: 12800, prompts: 2, tools: 4, seconds: 20 },
		{ state: 'success', tokens: 13200, prompts: 2, tools: 4, seconds: 61 },
		{ state: 'idle', tokens: 13200, prompts: 2, tools: 4, seconds: 20 },
		{ state: 'thinking', tokens: 13800, prompts: 3, tools: 4, seconds: 20 },
		{ state: 'stopped', tokens: 13800, prompts: 3, tools: 4, seconds: 61 },
		{ state: 'idle', tokens: 13800, prompts: 3, tools: 4, seconds: 20 }
	];
	let index = $state(0);
	let elapsed = $state(0);
	let stepElapsed = $state(0);
	let now = $state(0);
	let reducedMotion = $state(false);
	const step = $derived(steps[index]);
	const pose = $derived(step.state);
	const label = $derived(labels[pose]);
	const tokens = $derived(step.tokens);
	const format = (value: number) => value >= 1e9 ? `${(value / 1e9).toFixed(1)}B` : value >= 1e6 ? `${(value / 1e6).toFixed(1)}M` : value >= 1e3 ? `${(value / 1e3).toFixed(1)}k` : String(value);
	const input = $derived(Math.round(tokens * 0.8));
	const output = $derived(tokens - input);
	const reasoning = $derived(Math.round(tokens * 0.1));
	const frames = $derived([
		`gpt-6.1-sol · high reasoning · ${format(tokens)} tokens`,
		`Codex desktop · ${format(tokens)} session tokens`,
		...(input > 0 || output > 0 ? [`Tokens · ${format(input)} input · ${format(output)} output · ${format(Math.round(tokens * 0.4))} cached`] : []),
		...(reasoning > 0 ? [`Reasoning · ${format(reasoning)} tokens`] : []),
		`Context · ${Math.round(62000 * 100 / 258000)}% · ${format(62000)} / ${format(258000)} tokens`,
		...(step.tools > 0 || step.prompts > 0 ? [`Session · ${step.prompts} prompts · ${step.tools} tool calls`] : []),
		...(step.tool && ['running', 'editing', 'waiting'].includes(pose) ? [`Tool · ${step.tool}`] : []),
		`Lifetime · ${format(12800000)} tokens · 87 sessions`,
		'Lifetime · 264 prompts · 1280 tool calls'
	]);
	const frame = $derived(Math.floor(now / 20) % frames.length);
	const clock = $derived(`${String(Math.floor(elapsed / 3600)).padStart(2, '0')}:${String(Math.floor(elapsed / 60) % 60).padStart(2, '0')}:${String(elapsed % 60).padStart(2, '0')}`);
	onMount(() => {
		const preference = window.matchMedia('(prefers-reduced-motion: reduce)');
		const updatePreference = () => { reducedMotion = preference.matches; };
		updatePreference();
		if (!reducedMotion) now = Math.floor(Date.now() / 1000);
		preference.addEventListener('change', updatePreference);
		const timer = setInterval(() => {
			if (document.hidden || reducedMotion) return;
			now = Math.floor(Date.now() / 1000);
			elapsed += 1;
			stepElapsed += 1;
			if (stepElapsed >= step.seconds) {
				index = (index + 1) % steps.length;
				stepElapsed = 0;
				if (index === 0) elapsed = 0;
			}
		}, 1000);
		return () => { clearInterval(timer); preference.removeEventListener('change', updatePreference); };
	});
</script>

<figure aria-label="Example Discord presence using the desktop app's activity states" class="w-full max-w-[80%] mx-auto">
	<div class="rounded-xl border border-[#39363e] bg-[#252329] p-[18px] text-[#ececec]">
		<div class="mb-[19px] flex items-center justify-between text-[11px] text-[#b7b7b7] [&>svg]:size-4 [&>svg]:text-[#b7b7b7]">
			<span>Playing a game</span><svg aria-label="Discord" viewBox="0 0 24 24" fill="currentColor"
				><path
					d="M19.7 5a18 18 0 0 0-4.3-1.4l-.5 1a16 16 0 0 0-5.8 0l-.5-1A18 18 0 0 0 4.3 5C1.6 9 1 13 1.3 17a17 17 0 0 0 5.3 2.7l1.1-1.8-1.7-.8.4-.3a12 12 0 0 0 11.2 0l.4.3-1.7.8 1.1 1.8a17 17 0 0 0 5.3-2.7C23 12.4 21.9 8.4 19.7 5ZM8.5 14.5c-1 0-1.7-.9-1.7-1.9s.7-1.9 1.7-1.9 1.7.9 1.7 1.9-.7 1.9-1.7 1.9Zm7 0c-1 0-1.7-.9-1.7-1.9s.7-1.9 1.7-1.9 1.7.9 1.7 1.9-.7 1.9-1.7 1.9Z"
				/></svg
			>
		</div>
		<div class="flex items-center gap-[14px] [&_strong]:text-[15px] [&_strong]:font-[550] [&_p]:my-1 [&_p]:text-[11px] [&_p]:leading-normal [&_p]:text-[#ececec] max-[800px]:gap-[11px]">
			<div class="relative size-16 shrink-0 [&>img:first-child]:size-full [&>img:first-child]:rounded-xl max-[800px]:size-[52px] max-[600px]:size-[60px]">
				<img src={`/sprites/codex/codex-${pose}.png`} alt={`Codex: ${label}`} title={`Codex · ${label} · gpt-6.1-sol · 12.8M lifetime tokens`} width="64" height="64" />
			</div>
			<div class="min-w-0 min-h-[100px] flex-1 content-center">
				<strong>codex-rpc</strong>
				<p>{label} in codex-rpc</p>
				<p class="min-h-[28px] !text-[#b7b7b7]" data-testid="presence-demo-stats">{frames[frame]}</p>
				<span class="mt-[5px] block text-[11px] text-[#b7b7b7] tabular-nums">{clock} elapsed</span>
			</div>
		</div>
		<a
			href="https://github.com/ge0rg3e/codex-rpc"
			target="_blank"
			rel="noreferrer"
			class="mt-[18px] block rounded-md bg-[#3b3842] p-[10px] text-center text-xs font-medium text-[#f1f0f5] hover:bg-[#48434f]"
		>
			View on GitHub
		</a>
	</div>
</figure>
