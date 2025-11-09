<script lang="ts">
	import '../app.css';
	import favicon from '$lib/assets/favicon.svg';
	import gunner from '$lib/assets/gunner.ico';
	import pugly from '$lib/assets/pugly.ico';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { ensureCommissionerAccess } from '$lib/commissionerGate';

	const props = $props();
	const { children } = props;

	const links = [
		{ href: '/', label: 'Home', requiresCommissioner: false },
		{ href: '/picks', label: 'Picks', requiresCommissioner: true }
	];

	const activePath = $derived($page.url.pathname);
	const currentYear = new Date().getFullYear();

	function navClass(isActive: boolean) {
		return `rounded-full px-4 py-2 font-semibold transition focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-400 ${
			isActive
				? 'bg-emerald-500 text-emerald-950 shadow-lg shadow-emerald-900/40'
				: 'text-slate-200 hover:bg-slate-800/80 hover:text-emerald-300'
		}`;
	}

	function handleNav(link: (typeof links)[number]) {
		if (link.requiresCommissioner && !ensureCommissionerAccess()) {
			return;
		}
		goto(link.href);
	}
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

<div class="min-h-dvh bg-slate-950 text-slate-100">
	<header class="border-b border-slate-800 bg-slate-900">
		<nav class="mx-auto flex max-w-6xl items-center justify-between px-4 py-4">
			<a href="/" class="flex items-center gap-2 text-lg font-semibold tracking-tight text-white">
				<img src={gunner} alt="" class="h-6 w-6 -translate-y-0.3 rounded-[50%]" />
				Big Dawg Pool
			</a>
			<div class="flex items-center gap-2 text-sm font-medium">
				{#each links as link}
					{@const isActive = activePath === link.href}
					{#if link.requiresCommissioner}
						<button
							type="button"
							class={navClass(isActive)}
							onclick={() => handleNav(link)}
							aria-current={isActive ? 'page' : undefined}
						>
							{link.label}
						</button>
					{:else}
						<a
							href={link.href}
							class={navClass(isActive)}
							aria-current={isActive ? 'page' : undefined}
						>
							{link.label}
						</a>
					{/if}
				{/each}
			</div>
		</nav>
	</header>

	<main class="mx-auto flex max-w-6xl flex-1 flex-col gap-6 px-4 pt-10 pb-16">
		{@render children?.()}
	</main>

	<footer class="border-t border-slate-800 bg-slate-900 py-6 text-xs text-slate-300">
		<div class="mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-3 px-4">
			<span class="flex items-center gap-2">
				<img src={pugly} alt="" class="h-6 w-6 rounded-[50%]" />
				Big Dawg Pool · Est. 2024 &mdash; {currentYear}
			</span>
			<span>Powered by SvelteKit · Supabase · SportsData.io</span>
		</div>
	</footer>
</div>
