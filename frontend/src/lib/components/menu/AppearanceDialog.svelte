<script lang="ts">
	import { onMount } from 'svelte';
	import Modal from '$components/Modal.svelte';
	import { api } from '$lib/services/api';
	import { t, APPEARANCE } from '$lib/services/i18n.svelte';
	import { toast } from '$lib/stores/toast';
	import { appearance, chooseTheme } from '$lib/stores/theme';
	import type { Theme } from '$lib/theme/apply';

	interface Props {
		onclose: () => void;
	}

	let { onclose }: Props = $props();

	let themes = $state<Theme[]>([]);
	let busy = $state(false);

	const lightThemes = $derived(themes.filter((th) => th.mode === 'light'));
	const darkThemes = $derived(themes.filter((th) => th.mode === 'dark'));
	const settings = $derived($appearance?.settings);

	onMount(async () => {
		try {
			themes = await api.getThemes();
		} catch (err: any) {
			toast.error(err?.message ?? String(err));
		}
	});

	function inUse(th: Theme): boolean {
		if (!settings) return false;
		return th.mode === 'light' ? settings.light_theme === th.code : settings.dark_theme === th.code;
	}

	async function pick(th: Theme) {
		if (busy) return;
		busy = true;
		try {
			await chooseTheme(th);
		} catch (err: any) {
			toast.error(err?.message ?? String(err));
		} finally {
			busy = false;
		}
	}

	/** Preview colors of a theme card: the theme's own tokens, never the app's variables */
	function preview(th: Theme) {
		const v = th.tokens.vars;
		const g = th.tokens.gray;
		const a = th.tokens.accent;
		const dark = th.mode === 'dark';
		return {
			page: v['page-bg'],
			panel: dark ? g[8] : v['surface'],
			border: dark ? g[7] : g[2],
			appbar: v['appbar'],
			appbarFg: v['appbar-fg'],
			text: v['fg'],
			secondary: dark ? g[4] : g[6],
			accent: dark ? a[4] : a[6],
			selected: v['row-selected']
		};
	}
</script>

{#snippet themeCard(th: Theme)}
	{@const c = preview(th)}
	<button
		type="button"
		class="theme-card"
		class:in-use={inUse(th)}
		aria-pressed={inUse(th)}
		disabled={busy}
		onclick={() => pick(th)}
		data-theme-code={th.code}
	>
		<div class="theme-preview" style:background-color={c.page} aria-hidden="true">
			<div class="pv-appbar" style:background-color={c.appbar}>
				<span style:background-color={c.appbarFg}></span>
			</div>
			<div class="pv-panel" style:background-color={c.panel} style:border-color={c.border}>
				<div class="pv-title" style:background-color={c.accent}></div>
				<div class="pv-row" style:background-color={c.text}></div>
				<div class="pv-row pv-selected" style:background-color={c.selected}>
					<span style:background-color={c.text}></span>
				</div>
				<div class="pv-row short" style:background-color={c.secondary}></div>
			</div>
		</div>
		<div class="theme-name">
			<span>{th.name}</span>
			{#if inUse(th)}
				<span class="in-use-badge">✓ {t(APPEARANCE.IN_USE)}</span>
			{/if}
		</div>
	</button>
{/snippet}

<Modal open={true} {onclose} size="expanded" title={t(APPEARANCE.TITLE)}>
	<div class="appearance-dialog">
		<div class="dialog-header">
			<h2 id="modal-title" class="dialog-title">{t(APPEARANCE.TITLE)}</h2>
			<button type="button" class="close-btn" onclick={onclose} aria-label={t(APPEARANCE.CLOSE)}>×</button>
		</div>
		<p class="hint">{t(APPEARANCE.HINT)}</p>

		{#each [{ list: lightThemes, mode: 'light', title: APPEARANCE.LIGHT_THEMES }, { list: darkThemes, mode: 'dark', title: APPEARANCE.DARK_THEMES }] as section (section.mode)}
			<section class="theme-section" data-mode={section.mode}>
				<h3 class="section-title">
					{t(section.title)}
					{#if settings?.mode === section.mode}
						<span class="current-mode">({t(APPEARANCE.CURRENT_MODE)})</span>
					{/if}
				</h3>
				<div class="theme-grid">
					{#each section.list as th (th.code)}
						{@render themeCard(th)}
					{/each}
				</div>
			</section>
		{/each}
	</div>
</Modal>

<style>
	.appearance-dialog {
		@apply p-6 max-h-[85vh] overflow-y-auto;
	}

	.dialog-header {
		@apply flex items-center justify-between mb-2;
	}

	.dialog-title {
		@apply text-xl font-bold text-heading dark:text-blue-400;
	}

	.close-btn {
		@apply text-2xl leading-none px-2 text-gray-500 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-50;
	}

	.hint {
		@apply text-sm text-gray-600 dark:text-gray-400 mb-4;
	}

	.section-title {
		@apply text-base font-semibold mb-2 text-fg;
	}

	.current-mode {
		@apply text-sm font-normal text-gray-500 dark:text-gray-400;
	}

	.theme-section {
		@apply mb-6;
	}

	.theme-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
		gap: 0.75rem;
	}

	.theme-card {
		@apply rounded-lg border-2 border-gray-200 dark:border-gray-700 p-2 text-left transition-colors;
		@apply hover:border-primary-500 focus-visible:border-primary-500;
	}

	.theme-card.in-use {
		@apply border-primary-600 dark:border-primary-400;
	}

	.theme-card:disabled {
		@apply cursor-wait;
	}

	.theme-preview {
		@apply rounded overflow-hidden h-24 flex flex-col;
	}

	.pv-appbar {
		@apply h-4 flex items-center px-2;
	}

	.pv-appbar span {
		@apply block h-1 w-8 rounded-sm opacity-80;
	}

	.pv-panel {
		@apply m-2 flex-1 rounded border p-2 flex flex-col gap-1.5;
	}

	.pv-title {
		@apply h-1.5 w-12 rounded-sm;
	}

	.pv-row {
		@apply h-1 w-full rounded-sm opacity-70;
	}

	.pv-row.short {
		@apply w-2/3;
	}

	.pv-selected {
		@apply h-2.5 opacity-100 flex items-center px-1;
	}

	.pv-selected span {
		@apply block h-1 w-3/4 rounded-sm opacity-70;
	}

	.theme-name {
		@apply mt-2 flex items-center justify-between text-sm text-fg;
	}

	.in-use-badge {
		@apply text-xs font-medium text-primary-600 dark:text-primary-400;
	}
</style>
