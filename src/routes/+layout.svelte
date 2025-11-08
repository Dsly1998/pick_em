<script lang="ts">
	import '../app.css';
	import favicon from '$lib/assets/favicon.svg';
	import { page } from '$app/stores';

	const props = $props();
	const { children } = props;

	const links = [
		{ href: '/', label: 'Home' },
		{ href: '/picks', label: 'Picks' }
	];

	const activePath = $derived($page.url.pathname);
	const currentYear = new Date().getFullYear();
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

<div class="min-h-dvh bg-slate-950 text-slate-100">
	<header class="border-b border-slate-800 bg-slate-900">
		<nav class="mx-auto flex max-w-6xl items-center justify-between px-4 py-4">
			<a href="/" class="flex items-center gap-2 text-lg font-semibold tracking-tight text-white">
				<span
					class="inline-block h-2 w-2 rounded-full bg-emerald-400 shadow-[0_0_12px_rgba(74,222,128,0.8)]"
				></span>
				Big Dawg Pool
			</a>
			<div class="flex items-center gap-2 text-sm font-medium">
				{#each links as link}
					{@const isActive = activePath === link.href}
					<a
						href={link.href}
						class={`rounded-full px-4 py-2 font-semibold transition focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-400 ${isActive ? 'bg-emerald-500 text-emerald-950 shadow-lg shadow-emerald-900/40' : 'text-slate-200 hover:bg-slate-800/80 hover:text-emerald-300'}`}
						aria-current={isActive ? 'page' : undefined}
					>
						{link.label}
					</a>
				{/each}
			</div>
		</nav>
	</header>

	<main class="mx-auto flex max-w-6xl flex-1 flex-col gap-6 px-4 pt-10 pb-16">
		{@render children?.()}
	</main>

	<footer class="border-t border-slate-800 bg-slate-900 py-6 text-xs text-slate-300">
		<div class="mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-3 px-4">
			<span>Big Dawg Pool · Est. 2024 &mdash; {currentYear}</span>
			<span>Powered by SvelteKit · Supabase · SportsData.io</span>
		</div>
	</footer>
</div>
