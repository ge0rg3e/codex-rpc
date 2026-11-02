<script lang="ts">
  import { onMount } from "svelte";
  import { Call } from "@wailsio/runtime";
  type ResetRecord = { reset_type: string; announced_at: string };
  type ResetStatus = {
    latest_reset: ResetRecord | null;
    scheduled_reset: {
      reset_type: string;
      scheduled_for: string | null;
    } | null;
    active_watch: {
      level: string;
      reset_chance_percent: number | null;
      forecast_window: string;
      text: string;
    } | null;
    stats: { last_reset_at: string | null; days_since_last: number | null };
  };
  type Snapshot = {
    enabled: boolean;
    tokens: number;
    sessions: number;
    active: boolean;
    source: string;
    login: string;
    rank: number | null;
    public: boolean;
    presence: string;
    sync: string;
    scanError: string;
    pairCode: string;
    resetStatus: ResetStatus | null;
    resetError: string;
  };
  let snapshot = $state<Snapshot>({
    enabled: false,
    tokens: 0,
    sessions: 0,
    active: false,
    source: "",
    login: "",
    rank: null,
    public: false,
    presence: "Connecting to the client…",
    sync: "",
    scanError: "",
    pairCode: "",
    resetStatus: null,
    resetError: "",
  });
  let error = $state(""),
    busy = $state(false);
  const call = <T,>(method: string, ...args: unknown[]) =>
    Call.ByName(`main.PresenceService.${method}`, ...args) as Promise<T>;
  const format = (n: number) =>
    new Intl.NumberFormat("en", {
      notation: "compact",
      maximumFractionDigits: 2,
    }).format(n);
  const resetType = (value: string) =>
    value === "banked" ? "Banked" : "Regular";
  const formatResetTime = (value: string | null) =>
    value
      ? new Intl.DateTimeFormat(undefined, {
          month: "short",
          day: "numeric",
          year: "numeric",
          hour: "numeric",
          minute: "2-digit",
        }).format(new Date(value))
      : "Time unavailable";
  const formatElapsed = (days: number | null) => {
    if (days === null || !Number.isFinite(days)) return "";

    const seconds = Math.max(0, days * 24 * 60 * 60);
    const units: [Intl.RelativeTimeFormatUnit, number][] = [
      ["second", 60],
      ["minute", 60],
      ["hour", 24],
      ["day", 7],
      ["week", 4.345],
      ["month", 12],
      ["year", Infinity],
    ];
    let amount = seconds;
    let unit: Intl.RelativeTimeFormatUnit = "second";

    for (const [nextUnit, limit] of units) {
      unit = nextUnit;
      if (amount < limit) break;
      amount /= limit;
    }

    return new Intl.RelativeTimeFormat(undefined, { numeric: "always" }).format(
      -Math.round(amount),
      unit,
    );
  };
  async function refresh() {
    try {
      snapshot = await call<Snapshot>("Snapshot");
    } catch {
      error =
        "Open this interface through the Wails app to read your local usage.";
    }
  }
  async function act(method: string, ...args: unknown[]) {
    busy = true;
    error = "";
    try {
      await call(method, ...args);
      await refresh();
      return true;
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
      return false;
    } finally {
      busy = false;
    }
  }
  onMount(() => {
    void refresh();
    const timer = setInterval(refresh, 2500);
    return () => clearInterval(timer);
  });
</script>

<main class="flex min-h-dvh flex-col p-5">
  <header class="flex items-center justify-between">
    <div
      class="flex items-center text-[15px] font-[550] tracking-[-.6px] [&>span]:font-normal [&>span]:text-muted"
    >
      <img
        class="mr-[9px] size-6 rounded-md object-contain"
        src="/logo.png"
        alt=""
        width="24"
        height="24"
      />codex<span>-rpc</span>
    </div>
  </header>
  <section
    class="mt-7 mb-[23px] flex items-center justify-between gap-[15px] [&_strong]:block [&_strong]:text-xs [&_strong]:font-medium [&_div>span]:mt-[5px] [&_div>span]:block [&_div>span]:max-w-[225px] [&_div>span]:text-[10px] [&_div>span]:leading-normal [&_div>span]:text-muted"
  >
    <div>
      <strong>Discord presence</strong><span
        >{snapshot.presence || "Connecting…"}</span
      >
    </div>
    <button
      class="flex h-[18px] w-[30px] shrink-0 items-center rounded-full border border-[#3a414e] bg-line p-0.5 [&>span]:size-3 [&>span]:rounded-full [&>span]:bg-muted [&>span]:transition-transform [&>span]:duration-[120ms] [&.on]:border-accent [&.on]:bg-accent [&.on>span]:translate-x-3 [&.on>span]:bg-[#202b44] motion-reduce:[&>span]:transition-none"
      class:on={snapshot.enabled}
      role="switch"
      aria-checked={snapshot.enabled}
      aria-label="Discord presence"
      disabled={busy}
      onclick={() => act("SetEnabled", !snapshot.enabled)}><span></span></button
    >
  </section>
  <section
    class="grid grid-cols-[1.25fr_1fr] gap-[25px] border-y border-line py-[21px] [&_div>span]:block [&_div>span]:text-[10px] [&_div>span]:text-muted [&_strong]:mt-[9px] [&_strong]:mb-1.5 [&_strong]:block [&_strong]:text-[28px] [&_strong]:leading-[1.15] [&_strong]:font-[450] [&_strong]:tracking-[-1px] [&_strong]:tabular-nums [&_small]:block [&_small]:text-[9px] [&_small]:text-muted"
  >
    <div>
      <span>Total tokens</span><strong>{format(snapshot.tokens)}</strong><small
        >{format(snapshot.sessions)} sessions</small
      >
    </div>
    <div>
      <span>Leaderboard</span><button
        class="mt-[9px] mb-1.5 flex items-center gap-[14px] border-0 bg-transparent p-0 text-[28px] leading-[1.15] font-[450] tracking-[-1px] text-[#d5def7] tabular-nums [&>span]:text-xs [&>span]:text-muted hover:[&>span]:text-accent"
        aria-label="Open leaderboard"
        onclick={() => act("OpenLeaderboard")}
        >{snapshot.rank ? `#${snapshot.rank}` : "—"}<span>↗</span></button
      ><small>{snapshot.rank ? "All time" : "Not ranked yet"}</small>
    </div>
  </section>
  <section class="pt-[21px] pb-[17px]">
    {#if snapshot.login}<div
        class="flex items-center justify-between gap-4 [&_strong]:block [&_strong]:text-xs [&_strong]:font-[450] [&_div>span]:mt-[5px] [&_div>span]:block [&_div>span]:text-[10px] [&_div>span]:text-muted"
      >
        <div>
          <strong>@{snapshot.login}</strong><span
            >{snapshot.public ? "Visible on leaderboard" : "Public profile hidden"}</span
          >
        </div>
        <button
          class="border-0 bg-transparent p-0 text-[10px] text-muted hover:text-ink"
          disabled={busy}
          onclick={() => act("Disconnect")}>Disconnect</button
        >
      </div>
      <div class="mt-3 flex items-center justify-between gap-3">
        <span class="text-[10px] text-muted">Leaderboard visibility</span>
        <button
          class="flex h-[18px] w-[30px] shrink-0 items-center rounded-full border border-[#3a414e] bg-line p-0.5 [&>span]:size-3 [&>span]:rounded-full [&>span]:bg-muted [&>span]:transition-transform [&.on]:border-accent [&.on]:bg-accent [&.on>span]:translate-x-3 [&.on>span]:bg-[#202b44]"
          class:on={snapshot.public}
          role="switch"
          aria-checked={snapshot.public}
          aria-label="Leaderboard visibility"
          disabled={busy}
          onclick={() => act("SetPublic", !snapshot.public)}><span></span></button
        >
      </div>
      <p class="mt-2.5 mb-0 text-center text-[9px] leading-[1.6] text-muted">
        {snapshot.sync}
      </p>{:else}<button
        class="flex w-full items-center justify-center gap-[9px] rounded-md border border-[#343b48] bg-[#1a1e27] px-3 py-2.5 text-[11px] font-[450] text-ink hover:border-[#4a5468] hover:bg-[#242a36] [&>svg]:size-[14px] [&>span]:ml-auto [&>span]:text-muted"
        disabled={busy || !!snapshot.pairCode}
        onclick={() => act("ConnectGitHub")}
        ><svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"
          ><path
            d="M12 2a10 10 0 0 0-3.16 19.49c.5.09.68-.22.68-.48v-1.86c-2.78.61-3.37-1.18-3.37-1.18-.45-1.16-1.11-1.47-1.11-1.47-.91-.62.07-.61.07-.61 1 .07 1.53 1.03 1.53 1.03.89 1.53 2.34 1.09 2.91.83.09-.65.35-1.09.63-1.34-2.22-.25-4.56-1.11-4.56-4.94 0-1.09.39-1.98 1.03-2.68-.1-.25-.45-1.27.1-2.65 0 0 .84-.27 2.75 1.02a9.6 9.6 0 0 1 5 0c1.91-1.29 2.75-1.02 2.75-1.02.55 1.38.2 2.4.1 2.65.64.7 1.03 1.59 1.03 2.68 0 3.84-2.34 4.69-4.57 4.94.36.31.68.92.68 1.85v2.75c0 .27.18.58.69.48A10 10 0 0 0 12 2Z"
          /></svg
        >{snapshot.pairCode ? "Finish in your browser" : "Connect GitHub"}<span
          >↗</span
        ></button
      >{#if snapshot.pairCode}<p
          class="mt-3 mb-0 flex items-center gap-1.5 text-[10px] text-muted [&>strong]:font-mono [&>strong]:tracking-[.5px] [&>strong]:text-ink [&>button]:ml-auto"
        >
          Confirm <strong>{snapshot.pairCode}</strong>
          <button
            class="border-0 bg-transparent p-0 text-[10px] text-muted hover:text-ink"
            onclick={() => act("Disconnect")}>Cancel</button
          >
        </p>{:else}<p
          class="mt-2.5 mb-0 text-center text-[9px] leading-[1.6] text-muted"
        >
          Sync your usage. Join the leaderboard.
        </p>{/if}{/if}
  </section>
  <section
    class="mt-auto rounded-md border border-line bg-[#1c1c1c] px-3 py-2.5"
  >
    <div class="flex items-center justify-between gap-2">
      <div class="flex items-center gap-2">
        <span class="size-1.5 rounded-full bg-accent"></span>
        <strong class="text-xs font-medium">Codex resets</strong>
      </div>
      <button
        class="border-0 bg-transparent p-0 text-[9px] text-muted hover:text-ink"
        onclick={() => act("OpenResetSource")}>Data source ↗</button
      >
    </div>

    {#if snapshot.resetStatus?.latest_reset}
      <div
        class="mt-2.5 flex items-center justify-between gap-2 border-t border-line pt-2"
      >
        <div>
          <span class="block text-[9px] text-muted">LAST RESET</span>
          <strong class="mt-0.5 block text-[11px] font-medium text-ink">
            {resetType(snapshot.resetStatus.latest_reset.reset_type)}
          </strong>
        </div>
        <div class="text-right">
          <time
            class="block text-[10px] text-ink"
            datetime={snapshot.resetStatus.stats.last_reset_at ||
              snapshot.resetStatus.latest_reset.announced_at}
            >{formatResetTime(
              snapshot.resetStatus.stats.last_reset_at ||
                snapshot.resetStatus.latest_reset.announced_at,
            )}</time
          >
          {#if snapshot.resetStatus.stats.days_since_last !== null}
            <span class="mt-0.5 block text-[9px] text-muted"
              >{formatElapsed(snapshot.resetStatus.stats.days_since_last)}</span
            >
          {/if}
        </div>
      </div>
    {:else if snapshot.resetStatus}
      <p class="mt-2 mb-0 text-[10px] text-muted">
        No reset history is available yet.
      </p>
    {:else if snapshot.resetError}
      <p class="mt-2 mb-0 text-[10px] text-muted">
        Reset status is temporarily unavailable.
      </p>
    {:else}
      <p class="mt-2 mb-0 text-[10px] text-muted">Checking reset status…</p>
    {/if}

    {#if snapshot.resetStatus?.scheduled_reset}
      <div
        class="mt-2 flex items-center justify-between gap-2 border-t border-line pt-2"
      >
        <div>
          <span class="block text-[9px] text-muted">SCHEDULED ANNOUNCEMENT</span
          >
          <strong class="mt-0.5 block text-[10px] font-medium text-ink">
            {resetType(snapshot.resetStatus.scheduled_reset.reset_type)} reset
          </strong>
        </div>
        {#if snapshot.resetStatus.scheduled_reset.scheduled_for}
          <time
            class="text-right text-[9px] text-muted"
            datetime={snapshot.resetStatus.scheduled_reset.scheduled_for}
          >
            {formatResetTime(
              snapshot.resetStatus.scheduled_reset.scheduled_for,
            )}
          </time>
        {:else}
          <span class="text-[9px] text-muted">Timing unknown</span>
        {/if}
      </div>
    {/if}

    {#if snapshot.resetStatus?.active_watch}
      <div
        class="mt-2 border-t border-line pt-2"
        title={snapshot.resetStatus.active_watch.text}
      >
        <p class="m-0 text-[10px] text-ink">
          Watch forecast: {snapshot.resetStatus.active_watch.level}
          {#if snapshot.resetStatus.active_watch.reset_chance_percent !== null}
            · {snapshot.resetStatus.active_watch.reset_chance_percent}%
          {/if}
          · {snapshot.resetStatus.active_watch.forecast_window}
        </p>
        <p class="mt-0.5 mb-0 text-[9px] text-muted">
          AI forecast, not an official commitment.
        </p>
      </div>
    {/if}
    {#if snapshot.resetError && snapshot.resetStatus}
      <p class="mt-1 mb-0 text-[9px] text-muted">
        Could not refresh; showing the last result.
      </p>
    {/if}
  </section>
  {#if error || snapshot.scanError}<p
      class="mb-[14px] text-[10px] leading-[1.6] wrap-anywhere text-[#e3aba6]"
      role="alert"
    >
      {error || snapshot.scanError}
    </p>{/if}
  <footer
    class="mt-auto flex items-center justify-between gap-2.5 pt-[15px] text-[9px] text-muted [&>button]:border-0 [&>button]:bg-transparent [&>button]:p-0 [&>button]:text-[10px] [&>button]:text-muted [&>button:hover]:text-ink"
  >
    <span class="inline-flex items-center gap-1.5"
      ><span
        class="size-1 rounded-full bg-[#737f92] [&.online]:bg-[#8eb8a4]"
        class:online={snapshot.active}
      ></span>{snapshot.active
        ? `${snapshot.source || "Codex"} active`
        : "Waiting for Codex"}</span
    ><button onclick={() => act("Quit")}>Quit</button>
  </footer>
</main>
