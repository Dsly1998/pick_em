<script lang="ts">
import { goto } from '$app/navigation';
	import type { PageData } from './$types';
	import type { GamePick } from '$lib/types';
	import { formatRecord } from '$lib/utils/records';

	const props = $props();
	const data = $derived(props.data as PageData);

	const seasons = $derived(data.seasons ?? []);
	const season = $derived(data.season ?? null);
	const weeks = $derived(data.weeks ?? []);
	const activeWeek = $derived(data.activeWeek ?? null);
	const members = $derived(data.members ?? []);
	const games = $derived(data.games ?? []);
	let selectedSeasonId = $state('');
	let selectedWeekValue = $state('');

	$effect(() => {
		selectedSeasonId = data.selectedSeasonId ?? seasons[0]?.id ?? '';
		const weekNumber = data.selectedWeekNumber ?? activeWeek?.number ?? weeks[0]?.number ?? 1;
		selectedWeekValue = String(weekNumber);
	});

	const sortedMembers = $derived(
		[...members].sort(
			(a, b) =>
				b.seasonRecord.wins - a.seasonRecord.wins || a.seasonRecord.losses - b.seasonRecord.losses
		)
	);

	type GameType = (typeof games)[number];
	const topMember = $derived(sortedMembers[0]);

	const totalWins = $derived(members.reduce((acc, member) => acc + member.seasonRecord.wins, 0));
	const totalLosses = $derived(
		members.reduce((acc, member) => acc + member.seasonRecord.losses, 0)
	);
	const finalGameCount = $derived(games.filter((game) => game.status === 'final').length);
	const allPicks = $derived(games.flatMap((game) => game.picks));
	const pendingPickCount = $derived(allPicks.filter((pick) => pick.status === 'pending').length);
	const correctPickCount = $derived(allPicks.filter((pick) => pick.status === 'correct').length);
	const incorrectPickCount = $derived(
		allPicks.filter((pick) => pick.status === 'incorrect').length
	);

const commissionerName = 'Brad';

const weekStatus = $derived(() => {
	if (games.some((game) => game.status === 'in-progress')) {
		return 'In Progress';
		}
		if (games.some((game) => game.status === 'final')) {
			return 'Final';
		}
	return 'Upcoming';
});

const selectedWeekNumber = $derived(
	(() => {
		const parsed = Number.parseInt(selectedWeekValue, 10);
		if (!Number.isNaN(parsed) && parsed > 0) {
			return parsed;
		}
		return activeWeek?.number ?? weeks[0]?.number ?? 1;
	})()
);

const picksPageHref = $derived(() => {
	if (!selectedSeasonId) {
		return '/picks';
	}
	const params = new URLSearchParams();
	params.set('season', selectedSeasonId);
	params.set('week', String(selectedWeekNumber));
	return `/picks?${params.toString()}`;
});

function navigate(seasonId: string, weekNumber?: number) {
	const params = new URLSearchParams();
	params.set('season', seasonId);
	if (weekNumber && Number.isFinite(weekNumber)) {
		params.set('week', String(weekNumber));
	}
	goto(`?${params.toString()}`, { keepfocus: true, noscroll: true });
}

	function handleSeasonChange(id: string) {
		navigate(id);
	}

	function handleWeekChange(week: number) {
		if (!selectedSeasonId) return;
		navigate(selectedSeasonId, week);
	}

	function formatKickoff(kickoff?: string | null) {
		if (!kickoff) return 'TBD';
		return new Date(kickoff).toLocaleString(undefined, {
			weekday: 'short',
			month: 'short',
			day: 'numeric',
			hour: 'numeric',
			minute: '2-digit'
		});
	}

	function teamLabel(game: GameType, side: 'home' | 'away') {
	const team = side === 'home' ? game.homeTeam : game.awayTeam;
	if (!team) {
		return side === 'home' ? 'Home Team' : 'Away Team';
	}
	return `${team.location ?? ''} ${team.name ?? ''}`.trim() || (side === 'home' ? 'Home Team' : 'Away Team');
}

function winnerLabel(game: GameType) {
	if (!game.winner) return null;
	const side = game.winner === 'home' ? 'home' : 'away';
	const team = side === 'home' ? game.homeTeam : game.awayTeam;
	if (!team) {
		return side === 'home' ? 'Home Team' : 'Away Team';
	}
	return team.name ?? (side === 'home' ? 'Home Team' : 'Away Team');
}

	function cellClasses(game: GameType, pick?: GamePick | null) {
		const classes = [
			'rounded-xl',
			'border',
			'border-slate-700/60',
			'bg-slate-900/80',
			'p-3',
			'text-sm',
			'text-slate-100',
			'shadow-sm'
		];

		if (pick) {
			classes.push('border-emerald-400/50', 'bg-emerald-600/15');
		}

		if (pick && game.status === 'final' && game.winner) {
			if (game.winner === pick.chosenSide) {
				classes.push('border-emerald-400', 'bg-emerald-500/20', 'text-emerald-50');
			} else {
				classes.push('border-rose-400/70', 'bg-rose-500/20', 'text-rose-100');
			}
		}

		return classes.join(' ');
	}
</script>

{#if !season || !activeWeek}
	<section
		class="rounded-3xl border border-slate-700 bg-slate-900/80 p-8 text-center text-slate-200 shadow-xl shadow-black/40"
	>
		<h1 class="text-3xl font-semibold text-white">Big Dog Pool</h1>
		<p class="mt-3 text-sm">Add a season and weeks in Supabase to start tracking picks.</p>
	</section>
{:else}
	<section
		class="space-y-6 rounded-3xl border border-emerald-400/40 bg-slate-900/85 p-6 shadow-xl shadow-emerald-950/40"
	>
		<div class="flex flex-wrap items-start justify-between gap-6">
			<div class="max-w-2xl space-y-3">
				<p class="text-xs tracking-[0.45em] text-emerald-300/90 uppercase">The Big Dog Pool</p>
				<h1 class="text-4xl font-semibold text-white sm:text-5xl">
					Big Dog Pool Grid · {season.year}
				</h1>
				<p class="text-base text-slate-100/90 sm:text-lg">
					{commissionerName} keeps the official ledger, but everyone can track the action right here.
					Check the grid as picks lock in and every final goes green.
				</p>
			</div>
			<div class="flex flex-wrap items-center gap-3 text-sm text-slate-100">
				<label class="flex flex-col gap-1">
					<span class="text-xs tracking-wide text-slate-300 uppercase">Season</span>
					<select
						class="min-w-[12rem] rounded-xl border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-white focus:border-emerald-400 focus:ring-2 focus:ring-emerald-400 focus:outline-none"
						bind:value={selectedSeasonId}
						onchange={(event) => handleSeasonChange(event.currentTarget.value)}
					>
						{#each seasons as seasonOption (seasonOption.id)}
							<option value={seasonOption.id}>{seasonOption.label} · {seasonOption.year}</option>
						{/each}
					</select>
				</label>
				<label class="flex flex-col gap-1">
					<span class="text-xs tracking-wide text-slate-300 uppercase">Week</span>
					<select
						class="min-w-[8rem] rounded-xl border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-white focus:border-emerald-400 focus:ring-2 focus:ring-emerald-400 focus:outline-none"
						bind:value={selectedWeekValue}
						onchange={(event) => handleWeekChange(Number(event.currentTarget.value))}
					>
						{#each weeks as weekOption (weekOption.number)}
							<option value={weekOption.number}>Week {weekOption.number}</option>
						{/each}
					</select>
				</label>
			</div>
			<div class="flex flex-wrap gap-4 text-sm text-slate-100">
				<div
					class="rounded-2xl border border-slate-700 bg-slate-950/70 px-4 py-3 text-center shadow-sm"
				>
					<p class="text-xs tracking-wide text-emerald-300/80 uppercase">Current Week</p>
					<p class="mt-2 text-xl font-semibold text-white">Week {activeWeek.number}</p>
					<p class="text-xs text-slate-400">{weekStatus}</p>
				</div>
				<div
					class="rounded-2xl border border-slate-700 bg-slate-950/70 px-4 py-3 text-center shadow-sm"
				>
					<p class="text-xs tracking-wide text-emerald-300/80 uppercase">Season Record</p>
					<p class="mt-2 text-xl font-semibold text-white">{totalWins}-{totalLosses}</p>
					<p class="text-xs text-slate-400">Combined family wins/losses</p>
				</div>
			</div>
		</div>
		<div class="flex flex-wrap items-center justify-between gap-3 text-sm text-slate-100/80">
			<div class="flex flex-wrap gap-3">
				<span
					class="inline-flex items-center gap-2 rounded-full border border-slate-700 bg-slate-950/70 px-3 py-1.5"
				>
					<span class="h-2 w-2 rounded-full bg-emerald-400 shadow-[0_0_8px_rgba(16,185,129,0.8)]"
					></span>
					{season.label}
				</span>
				<span
					class="inline-flex items-center gap-2 rounded-full border border-slate-700 bg-slate-950/70 px-3 py-1.5"
				>
					<strong class="text-white">{finalGameCount}</strong>
					finals recorded
				</span>
				<span
					class="inline-flex items-center gap-2 rounded-full border border-slate-700 bg-slate-950/70 px-3 py-1.5"
				>
					<strong class="text-white">{correctPickCount}</strong>
					correct · {incorrectPickCount} incorrect
				</span>
				<span
					class="inline-flex items-center gap-2 rounded-full border border-slate-700 bg-slate-950/70 px-3 py-1.5"
				>
					<strong class="text-white">{pendingPickCount}</strong>
					picks still open
				</span>
			</div>
		<a
			href={picksPageHref}
				class="inline-flex items-center rounded-full bg-emerald-500 px-6 py-2 text-sm font-semibold text-emerald-950 shadow-lg shadow-emerald-900/50 transition hover:bg-emerald-400 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-400"
			>
				Set Week {activeWeek.number} Picks
			</a>
		</div>
		<div
			class="overflow-x-auto rounded-2xl border border-slate-800 bg-slate-950/80 p-2 shadow-inner shadow-black/40"
		>
			<div
				class="grid min-w-[720px] gap-2"
				style={`grid-template-columns: minmax(240px, 1.5fr) repeat(${members.length}, minmax(140px, 1fr));`}
			>
				<div class="px-3 py-2 text-xs font-semibold tracking-wide text-slate-300 uppercase">
					Game
				</div>
				{#each members as member (member.id)}
					<div class="px-3 py-2 text-xs font-semibold tracking-wide text-slate-300 uppercase">
						{member.name}
					</div>
				{/each}

				{#each games as game (game.id)}
					<div
						class="space-y-1 rounded-xl border border-slate-700 bg-slate-900/85 p-3 text-sm text-slate-100"
					>
						<p class="font-semibold text-white">
							{teamLabel(game, 'home')} <span class="text-emerald-300">vs</span>
							{teamLabel(game, 'away')}
						</p>
						<p class="text-xs text-slate-300">
							{formatKickoff(game.kickoff)} · {game.location ?? 'Venue TBD'}
						</p>
						{#if game.status === 'final' && game.homeScore != null && game.awayScore != null}
							<p class="text-xs text-emerald-300">
								Final: {game.homeScore} - {game.awayScore}
								{#if winnerLabel(game)}
									• {winnerLabel(game)}
								{/if}
							</p>
						{:else}
							<p class="text-xs tracking-wide text-slate-400 uppercase">{game.status}</p>
						{/if}
					</div>
					{#each members as member (member.id)}
						{@const memberPick = game.picks.find((entry) => entry.memberId === member.id)}
						<div class={cellClasses(game, memberPick)}>
							<p class="text-xs tracking-wide text-slate-200 uppercase">
								{memberPick ? (memberPick.chosenSide === 'home' ? 'Home' : 'Away') : 'Pending'}
							</p>
							<p class="mt-1 text-sm font-semibold text-white">
								{memberPick ? teamLabel(game, memberPick.chosenSide) : 'No pick yet'}
							</p>
							{#if game.status === 'final' && memberPick}
								<p class="mt-1 text-xs text-slate-100">
									{game.winner === memberPick.chosenSide ? 'Correct pick' : 'Missed pick'}
								</p>
							{:else if !memberPick}
								<p class="mt-1 text-xs text-slate-300">Locks when submitted</p>
							{:else}
								<p class="mt-1 text-xs text-slate-200">Locked in</p>
							{/if}
						</div>
					{/each}
				{/each}
			</div>
		</div>
	</section>

	<section class="grid gap-6 lg:grid-cols-[minmax(0,3fr)_minmax(0,2fr)]">
		<article
			class="space-y-5 rounded-3xl border border-slate-700 bg-slate-900/80 p-6 shadow-lg shadow-black/40"
		>
			<h2 class="text-3xl font-semibold text-white">Welcome to the Big Dog Pool</h2>
			<p class="text-base text-slate-100/90 sm:text-lg">
				The Big Dog Pool is the family tradition. {commissionerName} keeps scores honest while everyone
				chases bragging rights, prime rib, and the weekly crown.
			</p>
			<ul class="space-y-3 text-sm text-slate-200">
				<li class="flex items-center gap-3">
					<span class="inline-flex h-2 w-2 rounded-full bg-emerald-400"></span>
					Six players this year: {members.map((member) => member.name).join(', ')}.
				</li>
				<li class="flex items-center gap-3">
					<span class="inline-flex h-2 w-2 rounded-full bg-emerald-400"></span>
					Tie-breakers lock each week; lowest differential wins the Big Dog bone.
				</li>
				<li class="flex items-center gap-3">
					<span class="inline-flex h-2 w-2 rounded-full bg-emerald-400"></span>
					Everything syncs with SportsData.io, so finals light up automatically.
				</li>
			</ul>
			<div class="grid gap-3 sm:grid-cols-2">
				<div
					class="rounded-2xl border border-emerald-400/40 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-200"
				>
					<p class="text-xs tracking-wide text-emerald-200/80 uppercase">Commissioner</p>
					<p class="mt-2 text-xl font-semibold text-white">{commissionerName}</p>
					<p class="text-xs text-emerald-100/80">Call the commish if a pick needs adjusting.</p>
				</div>
				{#if topMember}
					<div
						class="rounded-2xl border border-slate-700 bg-slate-950/60 px-4 py-3 text-sm text-slate-100"
					>
						<p class="text-xs tracking-wide text-slate-300 uppercase">Current top dog</p>
						<p class="mt-2 text-xl font-semibold text-white">{topMember.name}</p>
						<p class="text-xs text-slate-400">{formatRecord(topMember.seasonRecord)} on the year</p>
					</div>
				{/if}
			</div>
		</article>
		<aside
			class="space-y-4 rounded-3xl border border-slate-700 bg-slate-900/80 p-6 shadow-lg shadow-black/40"
		>
			<h2 class="text-lg font-semibold text-white">Week {activeWeek.number} Snapshot</h2>
			<ul class="space-y-3 text-sm text-slate-200">
				<li class="flex items-center justify-between">
					<span>Games on the slate</span>
					<strong class="text-white">{games.length}</strong>
				</li>
				<li class="flex items-center justify-between">
					<span>Finals recorded</span>
					<strong class="text-emerald-300">{finalGameCount}</strong>
				</li>
				<li class="flex items-center justify-between">
					<span>Correct picks</span>
					<strong class="text-emerald-300">{correctPickCount}</strong>
				</li>
				<li class="flex items-center justify-between">
					<span>Incorrect picks</span>
					<strong class="text-rose-300">{incorrectPickCount}</strong>
				</li>
				<li class="flex items-center justify-between">
					<span>Picks still open</span>
					<strong class="text-white">{pendingPickCount}</strong>
				</li>
			</ul>
			<a
			href={picksPageHref}
				class="inline-flex w-full justify-center rounded-full border border-emerald-500/40 bg-emerald-500 px-4 py-2 text-sm font-semibold text-emerald-950 shadow-lg shadow-emerald-900/50 transition hover:bg-emerald-400 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-400"
			>
				Jump to picks
			</a>
		</aside>
	</section>

	<section class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
		{#each sortedMembers as member (member.id)}
			<div
				class="space-y-4 rounded-2xl border border-slate-700 bg-slate-900/85 p-5 shadow-md shadow-black/40"
			>
				<div class="flex items-center justify-between">
					<p class="text-base font-semibold text-white">{member.name}</p>
					<span class="text-xs tracking-wide text-emerald-300 uppercase"
						>Weeks won: {member.weeksWon}</span
					>
				</div>
				<div class="space-y-2 text-sm text-slate-200">
					<div class="flex items-center justify-between">
						<span class="text-xs tracking-wide text-slate-300 uppercase">Season</span>
						<span class="font-semibold text-emerald-300">{formatRecord(member.seasonRecord)}</span>
					</div>
					<div class="flex items-center justify-between">
						<span class="text-xs tracking-wide text-slate-300 uppercase">Last week</span>
						<span>{formatRecord(member.lastWeekRecord)}</span>
					</div>
					<div class="flex items-center justify-between">
						<span class="text-xs tracking-wide text-slate-300 uppercase">Tie breaker</span>
						<span>{member.tieBreakers[activeWeek.number] ?? '—'} pts</span>
					</div>
				</div>
				<a
				href={picksPageHref}
					class="inline-flex w-full justify-center rounded-full border border-emerald-500/40 bg-emerald-500 px-3 py-2 text-sm font-semibold text-emerald-950 shadow hover:bg-emerald-400 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-400"
				>
					Update picks
				</a>
			</div>
		{/each}
	</section>

	<section
		class="rounded-3xl border border-slate-700 bg-slate-900/85 p-6 shadow-lg shadow-black/40"
	>
		<div class="flex flex-wrap items-center justify-between gap-3">
			<h2 class="text-lg font-semibold text-white">Season Leaderboard</h2>
			<span class="text-xs tracking-wide text-slate-300 uppercase">Wins · Losses · Weeks won</span>
		</div>
		<table class="mt-4 w-full text-left text-sm">
			<thead class="text-xs tracking-wide text-slate-400 uppercase">
				<tr class="border-b border-slate-700 text-slate-300">
					<th class="py-2 font-medium">Player</th>
					<th class="py-2 text-right font-medium">Season</th>
					<th class="py-2 text-right font-medium">Last Week</th>
					<th class="py-2 text-right font-medium">Weeks Won</th>
				</tr>
			</thead>
			<tbody class="divide-y divide-slate-800/60 text-slate-100">
				{#each sortedMembers as member, index (member.id)}
					<tr class="transition hover:bg-slate-800/60">
						<td class="py-3">
							<div class="flex items-center gap-2">
								<span class="text-xs font-semibold tracking-wide text-emerald-300 uppercase">
									#{index + 1}
								</span>
								<span class="font-semibold text-white">{member.name}</span>
							</div>
						</td>
						<td class="py-3 text-right font-semibold text-emerald-300">
							{formatRecord(member.seasonRecord)}
						</td>
						<td class="py-3 text-right text-slate-200">
							{formatRecord(member.lastWeekRecord)}
						</td>
						<td class="py-3 text-right text-slate-300">{member.weeksWon}</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</section>

	<section
		class="rounded-3xl border border-slate-700 bg-slate-900/85 p-6 shadow-lg shadow-black/40"
	>
		<h2 class="text-lg font-semibold text-white">Week {activeWeek.number} Matchups</h2>
		<div class="mt-4 grid gap-4 md:grid-cols-2 xl:grid-cols-3">
			{#each games as game (game.id)}
				<div
					class="space-y-3 rounded-2xl border border-slate-700 bg-slate-950/70 p-4 shadow-sm shadow-black/30"
				>
					<div class="flex flex-wrap items-center justify-between gap-2 text-xs text-slate-300">
						<span
							class="rounded-full border border-emerald-400/40 px-2 py-1 tracking-wide uppercase"
						>
							{game.status}
						</span>
						<span>{formatKickoff(game.kickoff)}</span>
					</div>
					<div class="space-y-2 text-sm text-slate-100">
						<div class="flex items-center justify-between rounded-xl bg-slate-900 px-3 py-2">
							<span class="font-medium text-white">{teamLabel(game, 'home')}</span>
							<span class="text-xs text-slate-300">Home</span>
						</div>
						<div class="flex items-center justify-between rounded-xl bg-slate-900 px-3 py-2">
							<span class="font-medium text-white">{teamLabel(game, 'away')}</span>
							<span class="text-xs text-slate-300">Away</span>
						</div>
						{#if game.winner}
							<p class="text-xs text-emerald-300">
								Winner: {winnerLabel(game)}
							</p>
						{/if}
					</div>
							<p class="text-xs text-slate-400">{game.location ?? 'Venue TBD'}</p>
				</div>
			{/each}
		</div>
	</section>
{/if}
