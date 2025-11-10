<script lang="ts">
	import { goto } from '$app/navigation';
	import type { PageData } from './$types';
	import { formatRecord } from '$lib/utils/records';
	import { ensureCommissionerAccess } from '$lib/commissionerGate';
	import pugly from '$lib/assets/pugly.ico';
	import miller from '$lib/assets/miller.ico';

	const props = $props();
	const data = $derived(props.data as PageData);

	const seasons = $derived(data.seasons ?? []);
	const season = $derived(data.season ?? null);
	const weeks = $derived(data.weeks ?? []);
	const activeWeek = $derived(data.activeWeek ?? null);
	const members = $derived(data.members ?? []);
	const games = $derived(data.games ?? []);
	const gamesView = $derived(sortGamesByKickoff(games.map((game) => enrichGame(game))));

	const STORAGE_SEASON_KEY = 'bdp:selectedSeason';
	const STORAGE_WEEK_KEY = 'bdp:selectedWeek';

	const initialSeasonId = data.selectedSeasonId ?? seasons[0]?.id ?? '';
	const initialWeekNumber = data.selectedWeekNumber ?? activeWeek?.number ?? weeks[0]?.number ?? 1;

	let selectedSeasonId = $state(initialSeasonId);
	let selectedWeekValue = $state(String(initialWeekNumber));
	let restoredFromStorage = $state(false);

	const memberDisplayOrder = ['brad', 'dad', 'mom', 'danielle', 'lauren', 'dallin'];
	function displayIndex(member: typeof members[number]) {
		const index = memberDisplayOrder.findIndex((name) =>
			name === member.name.toLowerCase().trim()
		);
		return index === -1 ? memberDisplayOrder.length : index;
	}

	const sortedMembers = $derived(
		[...members].sort((a, b) => {
			if (b.weeksWon !== a.weeksWon) {
				return b.weeksWon - a.weeksWon;
			}
			const winDiff = b.seasonRecord.wins - a.seasonRecord.wins;
			if (winDiff !== 0) {
				return winDiff;
			}
			return a.seasonRecord.losses - b.seasonRecord.losses;
		})
	);

	const gridMembers = $derived(
		[...members].sort((a, b) => {
			const orderDiff = displayIndex(a) - displayIndex(b);
			if (orderDiff !== 0) return orderDiff;
			return a.name.localeCompare(b.name);
		})
	);

	type GameType = (typeof games)[number];
	const topMember = $derived(sortedMembers[0]);
	const homeGridMemberCount = $derived(Math.max(members.length, 1));

	const totalWins = $derived(members.reduce((acc, member) => acc + member.seasonRecord.wins, 0));
	const totalLosses = $derived(
		members.reduce((acc, member) => acc + member.seasonRecord.losses, 0)
	);
	const finalGameCount = $derived(gamesView.filter((game) => gameIsFinal(game)).length);
	const allPicks = $derived(gamesView.flatMap((game) => game.picks));
	const pendingPickCount = $derived(allPicks.filter((pick) => pick.status === 'pending').length);
	const correctPickCount = $derived(allPicks.filter((pick) => pick.status === 'correct').length);
	const incorrectPickCount = $derived(
		allPicks.filter((pick) => pick.status === 'incorrect').length
	);
	const allPicksSubmitted = $derived(
		gamesView.length > 0 &&
		members.length > 0 &&
		gamesView.every((game) =>
			members.every((member) => game.picks.some((pick) => pick.memberId === member.id))
		)
	);

	function memberWeekRecord(memberId: string) {
		let wins = 0;
		let losses = 0;
		for (const game of gamesView) {
			const pick = game.picks?.find((entry) => entry.memberId === memberId) ?? null;
			const outcome = pickOutcome(game, pick);
			if (outcome === 'win') {
				wins += 1;
			} else if (outcome === 'loss') {
				losses += 1;
			}
		}
		return { wins, losses };
	}

	function evaluateContention() {
		const membersToCheck = gridMembers;
		if (membersToCheck.length === 0) {
			return {};
		}
		const currentWins: Record<string, number> = {};
		for (const member of membersToCheck) {
			currentWins[member.id] = memberWeekRecord(member.id).wins;
		}
		const remainingGames = gamesView.filter((game) => !gameIsFinal(game));
		const canWinMap: Record<string, boolean> = {};
		for (const member of membersToCheck) {
			canWinMap[member.id] = false;
		}

		function dfs(index: number, wins: Record<string, number>) {
			if (index >= remainingGames.length) {
				let topScore = -Infinity;
				for (const member of membersToCheck) {
					const score = wins[member.id] ?? 0;
					if (score > topScore) {
						topScore = score;
					}
				}
				const leaders = membersToCheck.filter((member) => (wins[member.id] ?? 0) === topScore);
				for (const leader of leaders) {
					canWinMap[leader.id] = true;
				}
				return;
			}

			const game = remainingGames[index];
			const outcomes: Array<'home' | 'away'> = ['home', 'away'];
			for (const winnerSide of outcomes) {
				const nextWins: Record<string, number> = { ...wins };
				for (const pick of game.picks ?? []) {
					if (pick.chosenSide === winnerSide) {
						nextWins[pick.memberId] = (nextWins[pick.memberId] ?? 0) + 1;
					}
				}
				dfs(index + 1, nextWins);
				if (membersToCheck.every((member) => canWinMap[member.id])) {
					return;
				}
			}
		}

		dfs(0, { ...currentWins });
		return canWinMap;
	}

	const contentionMap = $derived(evaluateContention());

	const commissionerName = 'Brad';

	let weekStatus = $state('Upcoming');
	let selectedWeekNumber = $state(initialWeekNumber);
	let picksPageHref = $state('/picks');

	$effect(() => {
		if (gamesView.some((game) => gameInProgress(game))) {
			weekStatus = 'In Progress';
			return;
		}
		if (gamesView.some((game) => gameIsFinal(game))) {
			weekStatus = 'Final';
			return;
		}
		weekStatus = 'Upcoming';
	});

	$effect(() => {
		const parsed = Number.parseInt(selectedWeekValue, 10);
		selectedWeekNumber =
			!Number.isNaN(parsed) && parsed > 0 ? parsed : (activeWeek?.number ?? weeks[0]?.number ?? 1);
	});

	$effect(() => {
		if (!selectedSeasonId) {
			picksPageHref = '/picks';
			return;
		}
		const params = new URLSearchParams();
		params.set('season', selectedSeasonId);
		params.set('week', String(selectedWeekNumber));
		picksPageHref = `/picks?${params.toString()}`;
	});

	$effect(() => {
		if (typeof window === 'undefined' || restoredFromStorage) return;
		const url = new URL(window.location.href);
		const hasSeasonParam = url.searchParams.has('season');
		const hasWeekParam = url.searchParams.has('week');

		const storedSeason = window.localStorage.getItem(STORAGE_SEASON_KEY);
		const storedWeek = window.localStorage.getItem(STORAGE_WEEK_KEY);

		let seasonToUse = selectedSeasonId;
		let weekToUse = selectedWeekValue;
		let shouldNavigate = false;

		if (!hasSeasonParam && storedSeason && seasons.some((season) => season.id === storedSeason)) {
			seasonToUse = storedSeason;
			if (selectedSeasonId !== storedSeason) {
				selectedSeasonId = storedSeason;
			}
			shouldNavigate = true;
		}

		if (!hasWeekParam && storedWeek) {
			weekToUse = storedWeek;
			if (selectedWeekValue !== storedWeek) {
				selectedWeekValue = storedWeek;
			}
			shouldNavigate = true;
		}

		restoredFromStorage = true;

		if (shouldNavigate) {
			const parsed = Number.parseInt(weekToUse, 10);
			navigate(seasonToUse, Number.isNaN(parsed) ? undefined : parsed);
		}
	});

	$effect(() => {
		if (typeof window === 'undefined' || !restoredFromStorage) return;
		if (selectedSeasonId) {
			window.localStorage.setItem(STORAGE_SEASON_KEY, selectedSeasonId);
		}
		if (selectedWeekValue) {
			window.localStorage.setItem(STORAGE_WEEK_KEY, selectedWeekValue);
		}
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

	function normalizeStatus(status?: string | null) {
		return (status ?? '').toString().toLowerCase();
	}

	function normalizeGameStatus(status?: string | null): 'scheduled' | 'in-progress' | 'final' {
		const normalized = normalizeStatus(status);
		if (normalized === 'in-progress') return 'in-progress';
		if (normalized === 'final') return 'final';
		return 'scheduled';
	}

	function normalizePickStatus(status?: string | null): 'pending' | 'correct' | 'incorrect' | null {
		if (!status) return null;
		const value = status.toString().trim().toLowerCase();
		if (value === 'pending' || value === 'correct' || value === 'incorrect') {
			return value;
		}
		return null;
	}

	function normalizeSide(side?: string | null): 'home' | 'away' | null {
		const normalized = (side ?? '').toString().toLowerCase();
		if (normalized === 'home' || normalized === 'away') {
			return normalized;
		}
		return null;
	}

	function deriveWinner(game: GameType): 'home' | 'away' | null {
		const direct = normalizeSide(game.winner);
		if (direct) {
			return direct;
		}
		const homeScore = game.homeScore;
		const awayScore = game.awayScore;
		if (homeScore == null || awayScore == null) {
			return null;
		}
		if (homeScore === awayScore) {
			return null;
		}
		return homeScore > awayScore ? 'home' : 'away';
	}

	function enrichGame(game: GameType): GameType {
		const status = normalizeGameStatus(game.status);
		const winner = deriveWinner(game);
		const base = {
			...game,
			status,
			winner
		};
		return {
			...base,
			picks: (game.picks ?? []).map((pick) => {
				const side = normalizeSide(pick.chosenSide) ?? 'home';
				const normalized = normalizePickStatus(pick.status);
				return {
					...pick,
					chosenSide: side,
					status: normalized ?? determinePickStatus(base, side)
				};
			})
		};
	}

	function gameIsFinal(game: GameType) {
		return game.status === 'final';
	}

	function gameInProgress(game: GameType) {
		return game.status === 'in-progress';
	}

	function teamLabel(game: GameType, side: 'home' | 'away') {
		const team = side === 'home' ? game.homeTeam : game.awayTeam;
		if (!team) {
			return side === 'home' ? 'Home Team' : 'Away Team';
		}
		const parts = [team.location, team.name].filter((value) => !!value && value.trim().length > 0);
		if (parts.length > 0) {
			return parts.join(' ').trim();
		}
		if (team.code && team.code.trim().length > 0) {
			return team.code.toUpperCase();
		}
		return team.name?.trim() ?? (side === 'home' ? 'Home Team' : 'Away Team');
	}

	function winnerLabel(game: GameType) {
		const winner = deriveWinner(game);
		if (!winner) return null;
		const team = winner === 'home' ? game.homeTeam : game.awayTeam;
		if (!team) {
			return winner === 'home' ? 'Home Team' : 'Away Team';
		}
		return team.name ?? (winner === 'home' ? 'Home Team' : 'Away Team');
	}

	function formatTeamScore(value?: number | null) {
		if (typeof value === 'number' && Number.isFinite(value)) {
			return String(value);
		}
		return '—';
	}

	function teamScore(game: GameType, side: 'home' | 'away') {
		return formatTeamScore(side === 'home' ? game.homeScore : game.awayScore);
	}

	function pickOutcome(
		game: GameType,
		pick: GameType['picks'][number] | null
	): 'win' | 'loss' | 'pending' | 'none' {
		if (!pick) return 'none';
		const winner = deriveWinner(game);
		if (!winner || !gameIsFinal(game)) {
			return 'pending';
		}
		return pick.chosenSide === winner ? 'win' : 'loss';
	}

	function cellClasses(outcome: 'win' | 'loss' | 'pending' | 'none', hasPick: boolean) {
		const classes = [
			'family-grid__cell',
			'rounded-md',
			'border',
			'text-white',
			'text-center',
			'font-medium',
			'transition-colors',
			'duration-150'
		];

		if (outcome === 'win') {
			classes.push('bg-emerald-500/25', 'border-emerald-400/70', 'font-semibold');
		} else if (outcome === 'loss') {
			classes.push('bg-rose-500/25', 'border-rose-400/70', 'font-semibold');
		} else {
			classes.push('bg-slate-900/60', 'border-slate-700/70');
			if (!hasPick) {
				classes.push('italic', 'text-white/70');
			}
		}

		return classes.join(' ');
	}

	function teamNameOnly(game: GameType, side: 'home' | 'away') {
		const team = side === 'home' ? game.homeTeam : game.awayTeam;
		if (team?.name && team.name.trim().length > 0) {
			return team.name.trim();
		}
		if (team?.code && team.code.trim().length > 0) {
			return team.code.toUpperCase();
		}
		return teamLabel(game, side);
	}

	function determinePickStatus(
		game: GameType,
		side: 'home' | 'away'
	): 'pending' | 'correct' | 'incorrect' {
		const winner = deriveWinner(game);
		if (gameIsFinal(game) && winner) {
			return winner === side ? 'correct' : 'incorrect';
		}
		return 'pending';
	}

	function kickoffTimeValue(game: GameType): number | null {
		if (!game.kickoff) {
			return null;
		}
		const timestamp = Date.parse(game.kickoff);
		return Number.isNaN(timestamp) ? null : timestamp;
	}

	function sortGamesByKickoff(list: GameType[]) {
		return list
			.map((game, index) => ({ game, index }))
			.sort((a, b) => {
				const aValue = kickoffTimeValue(a.game);
				const bValue = kickoffTimeValue(b.game);
				if (aValue != null && bValue != null) {
					return aValue - bValue;
				}
				if (aValue != null) {
					return -1;
				}
				if (bValue != null) {
					return 1;
				}
				return a.index - b.index;
			})
			.map((entry) => entry.game);
	}

	function goToPicks() {
		if (!ensureCommissionerAccess()) {
			return;
		}
		goto(picksPageHref, { keepFocus: true, noscroll: false });
	}
</script>

{#if !season || !activeWeek}
	<section
		class="rounded-3xl border border-slate-700 bg-slate-900/80 p-8 text-center text-slate-200 shadow-xl shadow-black/40"
	>
		<h1 class="text-3xl font-semibold text-white">Big Dawg Pool</h1>
		<p class="mt-3 text-sm">Add a season and weeks in Supabase to start tracking picks.</p>
	</section>
{:else}
	<section
		class="space-y-6 rounded-3xl border border-emerald-400/40 bg-slate-900/85 p-6 shadow-xl shadow-emerald-950/40"
	>
		<div class="flex flex-wrap items-start justify-between gap-6">
			<div class="max-w-2xl space-y-3">
				<p class="text-xs tracking-[0.45em] text-emerald-300/90 uppercase">The Big Dawg Pool</p>
				<h1 class="text-4xl font-semibold text-white sm:text-5xl">
					Big Dawg Pool Grid · {season.year}
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
							<option value={String(weekOption.number)}>
								Week {weekOption.number}
								{#if weekOption.label && weekOption.label !== `Week ${weekOption.number}`}
									· {weekOption.label}
								{/if}
							</option>
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
			<button
				type="button"
				onclick={goToPicks}
				class="inline-flex items-center rounded-full bg-emerald-500 px-6 py-2 text-sm font-semibold text-emerald-950 shadow-lg shadow-emerald-900/50 transition hover:bg-emerald-400 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-400"
			>
				Set Week {activeWeek.number} Picks
			</button>
		</div>
		{#if allPicksSubmitted}
			<div
				class="w-full overflow-x-auto rounded-2xl border border-slate-800 bg-slate-950/80 p-2 shadow-inner shadow-black/40"
			>
				<div
					class="family-grid inline-grid min-w-full gap-1"
					style={`--family-grid-members: ${homeGridMemberCount};`}
				>
				<div class="family-grid__header rounded-md bg-slate-800/90 text-slate-100 uppercase">
					Game
				</div>
				{#each gridMembers as member (member.id)}
					<div
						class="family-grid__header rounded-md bg-slate-800/90 text-center text-slate-100 uppercase"
					>
						{member.name}
					</div>
				{/each}

				{#each gamesView as game (game.id)}
					<div
						class="family-grid__game rounded-md border border-slate-700 bg-slate-900 text-slate-100"
					>
						{teamNameOnly(game, 'home')} vs {teamNameOnly(game, 'away')}
					</div>
					{#each gridMembers as member (member.id)}
						{@const memberPick = game.picks.find((entry) => entry.memberId === member.id) ?? null}
						{@const outcome = pickOutcome(game, memberPick ?? null)}
						<div class={cellClasses(outcome, !!memberPick)}>
							{#if memberPick}
								{teamLabel(game, memberPick.chosenSide)}
							{:else}
								No pick yet
							{/if}
						</div>
					{/each}
				{/each}
				<div class="family-grid__game rounded-md border border-slate-700 bg-slate-900 text-slate-100 font-semibold">
					Points
				</div>
				{#each gridMembers as member (member.id)}
					<div class="family-grid__cell rounded-md border border-slate-600 bg-slate-900/70 text-center text-slate-100">
						{member.tieBreakers?.[activeWeek.number] ?? '—'}
					</div>
				{/each}
				<div class="family-grid__game rounded-md border border-slate-700 bg-slate-900 text-slate-100 font-semibold">
					Score
				</div>
				{#each gridMembers as member (member.id)}
					{@const record = memberWeekRecord(member.id)}
					<div class="family-grid__cell rounded-md border border-emerald-500/30 bg-emerald-500/10 text-center font-semibold text-emerald-200">
						{record.wins}-{record.losses}
					</div>
				{/each}
				<div class="family-grid__game rounded-md border border-slate-700 bg-slate-900 text-slate-100 font-semibold">
					Status
				</div>
				{#each gridMembers as member (member.id)}
					{@const canWin = contentionMap[member.id] ?? false}
					<div
						class={`family-grid__cell rounded-md text-center font-semibold ${
							canWin
								? 'status-pill status-pill--in'
								: 'status-pill status-pill--out'
						}`}
					>
						{canWin ? 'In contention' : 'Out'}
					</div>
				{/each}
			</div>
		</div>
		{:else}
			<div
				class="rounded-2xl border border-dashed border-slate-700 bg-slate-950/60 p-8 text-center"
			>
				<p class="text-lg font-semibold text-white">Waiting on picks…</p>
				<p class="mt-2 text-sm text-slate-300">
					The Big Dawg grid unlocks after all picks are locked in for the week. Check back soon!
				</p>
			</div>
		{/if}
	</section>

	<section class="grid gap-6 lg:grid-cols-[minmax(0,3fr)_minmax(0,2fr)]">
		<article
			class="space-y-5 rounded-3xl border border-slate-700 bg-slate-900/80 p-6 shadow-lg shadow-black/40"
		>
			<h2 class="text-3xl font-semibold text-white">Welcome to the Big Dawg Pool</h2>
			<p class="text-base text-slate-100/90 sm:text-lg">
				The Big Dawg Pool is the family tradition. {commissionerName} keeps scores honest while everyone
				chases bragging rights, prime rib, and the weekly crown.
			</p>
			<ul class="space-y-3 text-sm text-slate-200">
				<li class="flex items-center gap-3">
					<span class="inline-flex h-2 w-2 rounded-full bg-emerald-400"></span>
					Six players this year: {members.map((member) => member.name).join(', ')}.
				</li>
				<li class="flex items-center gap-3">
					<span class="inline-flex h-2 w-2 rounded-full bg-emerald-400"></span>
					Tie-breakers lock each week; lowest differential wins the Big Dawg.
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
						<p class="text-xs tracking-wide text-slate-300 uppercase">Current top dawg</p>
						<p class="mt-2 text-xl font-semibold text-white">{topMember.name}</p>
						<p class="text-xs text-slate-400">{formatRecord(topMember.seasonRecord)} on the year</p>
					</div>
				{/if}
			</div>
		</article>
		<aside
			class="space-y-4 rounded-3xl border border-slate-700 bg-slate-900/80 p-6 shadow-lg shadow-black/40"
		>
			<h2 class="flex items-center gap-2 text-lg font-semibold text-white">
				<img src={miller} alt="" class="h-5 w-5 rounded-[50%]" />
				Week {activeWeek.number} Snapshot
			</h2>
			<ul class="space-y-3 text-sm text-slate-200">
				<li class="flex items-center justify-between">
					<span>Games on the slate</span>
					<strong class="text-white">{gamesView.length}</strong>
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
			<button
				type="button"
				onclick={goToPicks}
				class="inline-flex w-full justify-center rounded-full border border-emerald-500/40 bg-emerald-500 px-4 py-2 text-sm font-semibold text-emerald-950 shadow-lg shadow-emerald-900/50 transition hover:bg-emerald-400 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-400"
			>
				Jump to picks
			</button>
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
				<button
					type="button"
					onclick={goToPicks}
					class="inline-flex w-full justify-center rounded-full border border-emerald-500/40 bg-emerald-500 px-3 py-2 text-sm font-semibold text-emerald-950 shadow hover:bg-emerald-400 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-400"
				>
					Update picks
				</button>
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
			{#each gamesView as game (game.id)}
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
							<div>
								<span class="font-medium text-white">{teamLabel(game, 'home')}</span>
								<span class="block text-xs text-slate-400">Home</span>
							</div>
							<span class="text-lg font-semibold text-white">{teamScore(game, 'home')}</span>
						</div>
						<div class="flex items-center justify-between rounded-xl bg-slate-900 px-3 py-2">
							<div>
								<span class="font-medium text-white">{teamLabel(game, 'away')}</span>
								<span class="block text-xs text-slate-400">Away</span>
							</div>
							<span class="text-lg font-semibold text-white">{teamScore(game, 'away')}</span>
						</div>
						<p class="text-xs text-emerald-300">
							Winner: {gameIsFinal(game) && game.winner ? winnerLabel(game) : 'TBD'}
						</p>
					</div>
					<p class="text-xs text-slate-400">{game.location ?? 'Venue TBD'}</p>
				</div>
			{/each}
		</div>
	</section>
{/if}

<style>
	.family-grid {
		grid-template-columns:
			minmax(120px, 1.2fr)
			repeat(var(--family-grid-members, 1), minmax(88px, 0.8fr));
		font-size: 0.7rem;
	}

	.family-grid__header,
	.family-grid__game,
	.family-grid__cell {
		padding: 0.35rem 0.45rem;
		line-height: 1.1;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.family-grid__header {
		font-size: 0.55rem;
		letter-spacing: 0.12em;
	}

	.family-grid__game {
		font-size: 0.7rem;
		font-weight: 600;
	}

	.family-grid__cell {
		font-size: 0.68rem;
	}

	@media (max-width: 1024px) {
		.family-grid {
			grid-template-columns:
				minmax(110px, 1.1fr)
				repeat(var(--family-grid-members, 1), minmax(80px, 0.75fr));
			font-size: 0.66rem;
		}

		.family-grid__header,
		.family-grid__game,
		.family-grid__cell {
			padding: 0.3rem 0.4rem;
		}
	}

	@media (max-width: 768px) {
		.family-grid {
			grid-template-columns:
				minmax(100px, 1.05fr)
				repeat(var(--family-grid-members, 1), minmax(72px, 0.7fr));
			font-size: 0.62rem;
		}

		.family-grid__game {
			font-size: 0.66rem;
		}

		.family-grid__cell {
			font-size: 0.62rem;
		}
	}

	@media (max-width: 640px) {
		.family-grid {
			grid-template-columns:
				minmax(94px, 1fr)
				repeat(var(--family-grid-members, 1), minmax(65px, 0.66fr));
			font-size: 0.58rem;
		}

		.family-grid__header,
		.family-grid__game,
		.family-grid__cell {
			padding: 0.28rem 0.35rem;
		}
	}

	.status-pill {
		border: 1px solid transparent;
		border-radius: 0.75rem;
		padding: 0.2rem 0.4rem;
	}

	.status-pill--in {
		background-color: rgba(16, 185, 129, 0.15);
		border-color: rgba(16, 185, 129, 0.35);
		color: rgb(209, 250, 229);
	}

	.status-pill--out {
		background-color: rgba(248, 113, 113, 0.15);
		border-color: rgba(248, 113, 113, 0.35);
		color: rgb(254, 226, 226);
	}
</style>
