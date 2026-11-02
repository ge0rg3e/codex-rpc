<script lang="ts">
	let { platform, dark = false }: { platform: string; dark?: boolean } = $props();
	const icons: Record<string, string> = { Windows: 'windows', macOS: 'apple', Linux: 'linux', 'Arch Linux': 'archlinux' };
</script>

<img src={`/platforms/${icons[platform]}.svg`} alt="" width="20" height="20" class={`size-5 shrink-0 object-contain brightness-0 ${dark ? '' : 'invert'}`} />
