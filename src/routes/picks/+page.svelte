<script lang="ts">
import { goto } from '$app/navigation';
	import { declareWeekWinner, syncWeek, upsertPick, upsertTieBreaker } from '$lib/api/client';
	import type { PageData } from './$types';
	import type { Game } from '$lib/types';

	const props = $props();
	const data = $derived(props.data as PageData);

	const seasons = $derived(data.seasons ?? []);
	const season = $derived(data.season ?? null);
	const weeks = $derived(data.weeks ?? []);
	const activeWeek = $derived(data.activeWeek ?? null);
	const members = $derived(data.members ?? []);
	const games = $derived(data.games ?? []);
	const weekResult = $derived(data.weekResult ?? null);

	const commissionerName = 'Brad';
	const commissioner = $derived(members.find((member) => member.isCommissioner) ?? null);

	let selectedSeasonId = $state('');
	let selectedWeekValue = $state('');
	let selectedMemberId = $state('');
	let selections = $state({} as Record<string, Record<string, 'home' | 'away'>>);
	let tieBreakerInputs = $state({} as Record<string, Record<number, string>>);
	let tieBreakerSaving = $state(false);
	let declareSaving = $state(false);
	let syncSaving = $state(false);
	let winnerMemberId = $state('');
	let winnerNotes = $state('');

	$effect(() => {
		selectedSeasonId = data.selectedSeasonId ?? seasons[0]?.id ?? '';
		const weekNumber = data.selectedWeekNumber ?? activeWeek?.number ?? weeks[0]?.number ?? 1;
		selectedWeekValue = String(weekNumber);
		selectedMemberId = members[0]?.id ?? '';
		selections = buildInitialSelections();
		tieBreakerInputs = buildInitialTieBreakers();
		winnerMemberId = weekResult?.winnerMemberId ?? '';
		winnerNotes = weekResult?.notes ?? '';
	});

	const selectedMember = $derived(
		members.find((member) => member.id === selectedMemberId) ?? members[0] ?? null
	);

let selectedWeekNumber = $state(1);
let currentTieBreaker = $state('');

$effect(() => {
	const parsed = Number.parseInt(selectedWeekValue, 10);
	selectedWeekNumber = !Number.isNaN(parsed) && parsed > 0
		? parsed
		: activeWeek?.number ?? weeks[0]?.number ?? 1;
});

$effect(() => {
	if (!selectedMember || !activeWeek) {
		currentTieBreaker = '';
		return;
	}
	currentTieBreaker = tieBreakerInputs[selectedMember.id]?.[activeWeek.number] ?? '';
});

	const selectedMemberSummary = $derived(
		selectedMember
			? {
					seasonRecord: selectedMember.seasonRecord,
					lastWeekRecord: selectedMember.lastWeekRecord,
					weeksWon: selectedMember.weeksWon
				}
			: null
	);

	function buildInitialSelections() {
		const base: Record<string, Record<string, 'home' | 'away'>> = {};
		for (const member of members) {
			base[member.id] = {};
		}
		for (const game of games) {
			for (const pick of game.picks) {
				base[pick.memberId] ??= {};
				base[pick.memberId][game.gameKey] = pick.chosenSide as 'home' | 'away';
			}
		}
		return base;
	}

	function buildInitialTieBreakers() {
		const base: Record<string, Record<number, string>> = {};
		for (const member of members) {
			base[member.id] = {};
			for (const week of weeks) {
				const stored = member.tieBreakers[week.number];
				base[member.id][week.number] = stored != null ? String(stored) : '';
			}
		}
		return base;
	}

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
		selectedWeekValue = String(week);
		navigate(selectedSeasonId, week);
	}

	function handleTieBreakerInput(value: string) {
		if (!selectedMemberId || !activeWeek) return;
		tieBreakerInputs = {
			...tieBreakerInputs,
			[selectedMemberId]: {
				...tieBreakerInputs[selectedMemberId],
				[activeWeek.number]: value
			}
		};
	}

	function getMemberPick(game: Game, memberId: string) {
		return selections[memberId]?.[game.gameKey] ?? null;
	}

	function pickIsCorrect(game: Game, side: 'home' | 'away') {
		return game.status === 'final' && game.winner === side;
	}

	function formattedDate(kickoff?: string | null) {
		if (!kickoff) return 'TBD';
	return new Date(kickoff).toLocaleString(undefined, {
		weekday: 'short',
		month: 'short',
		day: 'numeric',
		hour: 'numeric',
		minute: '2-digit'
	});
}

function teamLabel(game: Game, side: 'home' | 'away') {
	const team = side === 'home' ? game.homeTeam : game.awayTeam;
	if (!team) {
		return side === 'home' ? 'Home Team' : 'Away Team';
	}
	return `${team.location ?? ''} ${team.name ?? ''}`.trim() || (side === 'home' ? 'Home Team' : 'Away Team');
}

function winnerTeamLabel(game: Game): string {
	if (!game.winner) {
		return '';
	}
	const side = game.winner === 'home' ? 'home' : 'away';
	return teamLabel(game, side);
}

async function handlePick(gameKey: string, side: 'home' | 'away') {
		if (!selectedMemberId || !season || !activeWeek) return;
		const memberSelections = { ...(selections[selectedMemberId] ?? {}) };
		const previous = memberSelections[gameKey];
		memberSelections[gameKey] = side;
		selections = { ...selections, [selectedMemberId]: memberSelections };
		try {
			await upsertPick(fetch, {
				seasonId: season.id,
				weekNumber: activeWeek.number,
				memberId: selectedMemberId,
				gameKey,
				side
			});
		} catch (error) {
			if (previous) {
				memberSelections[gameKey] = previous;
			} else {
				delete memberSelections[gameKey];
			}
			selections = { ...selections, [selectedMemberId]: memberSelections };
			console.error(error);
			alert('Unable to save pick. Please try again.');
		}
	}

	async function saveTieBreaker() {
		if (!selectedMember || !season || !activeWeek) return;
		const rawValue = tieBreakerInputs[selectedMember.id]?.[activeWeek.number] ?? '';
		const points = Number.parseInt(rawValue, 10);
		if (Number.isNaN(points)) {
			alert('Enter a valid numeric tie-breaker.');
			return;
		}
		tieBreakerSaving = true;
		try {
			await upsertTieBreaker(fetch, {
				seasonId: season.id,
				weekNumber: activeWeek.number,
				memberId: selectedMember.id,
				points
			});
		} catch (error) {
			console.error(error);
			alert('Unable to save tie breaker.');
		} finally {
			tieBreakerSaving = false;
		}
	}

	async function submitWinner() {
		if (!season || !activeWeek) return;
		const declaredBy = commissioner?.id ?? selectedMemberId ?? members[0]?.id;
		if (!declaredBy) {
			alert('No commissioner available to declare a winner.');
			return;
		}
		declareSaving = true;
		try {
			const { weekResult: updated } = await declareWeekWinner(fetch, {
				seasonId: season.id,
				weekNumber: activeWeek.number,
				winnerMemberId: winnerMemberId ? winnerMemberId : null,
				declaredByMemberId: declaredBy,
				notes: winnerNotes
			});
			if (updated) {
				winnerMemberId = updated?.winnerMemberId ?? '';
				winnerNotes = updated?.notes ?? '';
			}
		} catch (error) {
			console.error(error);
			alert('Unable to declare winner.');
		} finally {
			declareSaving = false;
		}
	}

	async function syncCurrentWeek() {
		if (!season || !activeWeek) return;
		syncSaving = true;
		try {
			await syncWeek(fetch, { seasonId: season.id, weekNumber: activeWeek.number });
			alert('Games synced. Refresh to see updates.');
		} catch (error) {
			console.error(error);
			alert('Unable to sync games.');
		} finally {
			syncSaving = false;
		}
	}

	function pickButtonClasses({
		game,
		side,
		isActive
	}: {
		game: Game;
		side: 'home' | 'away';
		isActive: boolean;
	}) {
		const classes = [
			'flex',
			'h-full',
			'flex-col',
			'items-start',
			'justify-center',
			'rounded-2xl',
			'border',
			'border-slate-700',
			'bg-slate-900',
			'p-4',
			'text-left',
			'text-slate-100',
			'transition',
			'duration-150',
			'hover:border-emerald-400/60',
			'hover:bg-slate-800'
		];

		if (isActive) {
			classes.push(
				'border-emerald-400/70',
				'bg-emerald-500/20',
				'text-emerald-100',
				'shadow',
				'shadow-emerald-500/20'
			);
		}

		if (game.status === 'final') {
			if (pickIsCorrect(game, side)) {
				classes.push('border-emerald-400', 'bg-emerald-500/20', 'text-emerald-100');
			} else if (game.winner) {
				classes.push('border-rose-400/60', 'bg-rose-500/20', 'text-rose-100/90');
			}
		}

		return classes.join(' ');
	}

	function cellClasses(game: Game, memberId: string, chosen: 'home' | 'away' | null) {
		const classes = [
			'rounded-xl',
			'border',
			'border-slate-700',
			'bg-slate-900',
			'p-3',
			'text-sm',
			'text-slate-200',
			'transition',
			'duration-200'
		];

		if (chosen) {
			classes.push('bg-emerald-500/15', 'border-emerald-400/50', 'text-emerald-50');
		}

		if (game.status === 'final' && game.winner) {
			if (chosen === game.winner) {
				classes.push('bg-emerald-500/20', 'border-emerald-400/70', 'text-emerald-100');
			} else if (chosen) {
				classes.push('bg-rose-500/20', 'border-rose-400/60', 'text-rose-100/90');
			}
		}

		if (memberId === selectedMemberId) {
			classes.push('ring-1', 'ring-emerald-400/40');
		}

		return classes.join(' ');
	}

	function saveLabel(game: Game, side: 'home' | 'away') {
		const team = side === 'home' ? game.homeTeam : game.awayTeam;
		return `${team.location} ${team.name}`;
	}
</script>

{#if !season || !activeWeek}
	<section
		class="rounded-3xl border border-slate-700 bg-slate-900/80 p-8 text-center text-slate-200 shadow-xl shadow-black/40"
	>
		<h1 class="text-3xl font-semibold text-white">Big Dog Control</h1>
		<p class="mt-3 text-sm">Add a season and weeks in Supabase to start managing picks.</p>
	</section>
{:else}
	<div class="space-y-10">
		<header
			class="flex flex-col gap-6 rounded-3xl border border-emerald-500/40 bg-slate-900/85 p-6 shadow-xl shadow-emerald-900/40 sm:flex-row sm:items-center sm:justify-between sm:p-8"
		>
			<div class="space-y-3">
				<p class="text-xs tracking-[0.4em] text-emerald-300/80 uppercase">Big Dog Control</p>
				<h1 class="text-3xl font-semibold text-white sm:text-4xl">
					Set Week {selectedWeekNumber} Picks
				</h1>
				<p class="max-w-xl text-sm text-slate-100">
					This is the official control room for the Big Dog Pool. {commissionerName} can lock in adjustments,
					but everyone can cue up their winners and tie breaker right here.
				</p>
			</div>
			<div class="flex flex-col gap-3 sm:w-[22rem]">
				<label class="text-xs font-semibold tracking-wide text-slate-200 uppercase">
					Season
					<select
						bind:value={selectedSeasonId}
						class="mt-1 w-full rounded-xl border border-slate-700 bg-slate-900 px-4 py-2 text-sm text-white focus:border-emerald-400 focus:ring-2 focus:ring-emerald-400 focus:outline-none"
						onchange={(event) => handleSeasonChange(event.currentTarget.value)}
					>
						{#each seasons as seasonOption (seasonOption.id)}
							<option value={seasonOption.id}>{seasonOption.label} · {seasonOption.year}</option>
						{/each}
					</select>
				</label>
				<label class="text-xs font-semibold tracking-wide text-slate-200 uppercase">
					Week
					<select
						bind:value={selectedWeekValue}
						class="mt-1 w-full rounded-xl border border-slate-700 bg-slate-900 px-4 py-2 text-sm text-white focus:border-emerald-400 focus:ring-2 focus:ring-emerald-400 focus:outline-none"
						onchange={(event) => handleWeekChange(Number(event.currentTarget.value))}
					>
						{#each weeks as week (week.number)}
							<option value={week.number}>{week.label}</option>
						{/each}
					</select>
				</label>
				<label class="text-xs font-semibold tracking-wide text-slate-200 uppercase">
					Active family member
					<select
						bind:value={selectedMemberId}
						class="mt-1 w-full rounded-xl border border-slate-700 bg-slate-900 px-4 py-2 text-sm text-white focus:border-emerald-400 focus:ring-2 focus:ring-emerald-400 focus:outline-none"
					>
				{#each members as member (member.id)}
							<option value={member.id}>{member.name}</option>
						{/each}
					</select>
				</label>
			</div>
		</header>

		{#if selectedMemberSummary}
			<section class="grid gap-4 sm:grid-cols-3">
				<div class="rounded-2xl border border-slate-700 bg-slate-900 p-4 text-slate-100">
					<p class="text-xs tracking-wide text-slate-300 uppercase">Season record</p>
					<p class="mt-2 text-2xl font-semibold text-emerald-300">
						{selectedMemberSummary.seasonRecord.wins}-{selectedMemberSummary.seasonRecord.losses}
					</p>
				</div>
				<div class="rounded-2xl border border-slate-700 bg-slate-900 p-4 text-slate-100">
					<p class="text-xs tracking-wide text-slate-300 uppercase">Last week</p>
					<p class="mt-2 text-2xl font-semibold text-slate-100">
						{selectedMemberSummary.lastWeekRecord.wins}-{selectedMemberSummary.lastWeekRecord
							.losses}
					</p>
				</div>
				<div class="rounded-2xl border border-slate-700 bg-slate-900 p-4 text-slate-100">
					<p class="text-xs tracking-wide text-slate-300 uppercase">Weeks won</p>
					<p class="mt-2 text-2xl font-semibold text-emerald-200">
						{selectedMemberSummary.weeksWon}
					</p>
				</div>
			</section>
		{/if}

		<section class="grid gap-6 lg:grid-cols-[minmax(0,3fr)_minmax(0,2fr)]">
			<div class="space-y-4 rounded-3xl border border-slate-700 bg-slate-900 p-6">
				<div class="flex items-center justify-between gap-3">
					<h2 class="text-lg font-semibold text-white">Matchups</h2>
					<button
						class="inline-flex items-center gap-2 rounded-full border border-emerald-500/50 bg-emerald-500 px-4 py-2 text-xs font-semibold tracking-wide text-emerald-950 uppercase shadow hover:bg-emerald-400 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-400 disabled:cursor-not-allowed disabled:opacity-60"
						onclick={syncCurrentWeek}
						disabled={syncSaving}
					>
						{syncSaving ? 'Syncing…' : 'Sync Games'}
					</button>
				</div>
				<div class="space-y-5">
					{#each games as game (game.id)}
						<article class="space-y-4 rounded-2xl border border-slate-700 bg-slate-900 p-4">
							<header
								class="flex flex-wrap items-center justify-between gap-2 text-xs text-slate-200"
							>
								<div class="flex items-center gap-3">
									<span
										class="rounded-full border border-emerald-400/50 px-2 py-1 tracking-wide text-emerald-200 uppercase"
									>
										{game.status}
									</span>
									<span>{formattedDate(game.kickoff)}</span>
								</div>
								{#if game.channel}
									<span
										class="rounded-full border border-slate-600 px-2 py-1 tracking-wide text-slate-100 uppercase"
									>
										{game.channel}
									</span>
								{/if}
							</header>
							<div class="grid gap-3 sm:grid-cols-2">
								<button
									type="button"
									class={pickButtonClasses({
										game,
										side: 'home',
										isActive: getMemberPick(game, selectedMemberId) === 'home'
									})}
									onclick={() => handlePick(game.gameKey, 'home')}
								>
									<p class="text-xs tracking-wide text-slate-300 uppercase">Home</p>
									<p class="mt-1 text-lg font-semibold text-white">
										{teamLabel(game, 'home')}
									</p>
									{#if game.status === 'final' && game.winner}
										<p class="mt-2 text-xs text-emerald-200">
											Final: {game.winner === 'home' ? 'W' : 'L'}
										</p>
									{/if}
								</button>
								<button
									type="button"
									class={pickButtonClasses({
										game,
										side: 'away',
										isActive: getMemberPick(game, selectedMemberId) === 'away'
									})}
									onclick={() => handlePick(game.gameKey, 'away')}
								>
									<p class="text-xs tracking-wide text-slate-300 uppercase">Away</p>
									<p class="mt-1 text-lg font-semibold text-white">
										{teamLabel(game, 'away')}
									</p>
									{#if game.status === 'final' && game.winner}
										<p class="mt-2 text-xs text-emerald-200">
											Final: {game.winner === 'away' ? 'W' : 'L'}
										</p>
									{/if}
								</button>
							</div>
							<footer
								class="flex flex-wrap items-center justify-between gap-2 text-xs text-slate-400"
							>
								<span>{game.location ?? 'Venue TBD'}</span>
								<span class="text-slate-500">{game.gameKey}</span>
							</footer>
						</article>
					{/each}
				</div>
			</div>

			<div class="space-y-5">
				<section class="rounded-3xl border border-slate-700 bg-slate-900 p-6 text-slate-100">
					<h2 class="text-lg font-semibold text-white">Tie Breaker</h2>
					<p class="mt-2 text-sm text-slate-300">
						Enter your total points guess for Week {selectedWeekNumber}. We'll use this if there's a
						tie.
					</p>
					<input
						type="number"
						min="0"
						class="mt-4 w-full rounded-2xl border border-slate-700 bg-slate-900 px-4 py-2 text-sm text-white focus:border-emerald-400 focus:ring-2 focus:ring-emerald-400 focus:outline-none"
						value={currentTieBreaker}
						oninput={(event) => handleTieBreakerInput(event.currentTarget.value)}
					/>
					<button
						class="mt-4 inline-flex w-full justify-center rounded-full border border-emerald-500/50 bg-emerald-500 px-4 py-2 text-xs font-semibold tracking-wide text-emerald-950 uppercase shadow hover:bg-emerald-400 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-400 disabled:cursor-not-allowed disabled:opacity-60"
						onclick={saveTieBreaker}
						disabled={tieBreakerSaving}
					>
						{tieBreakerSaving ? 'Saving…' : 'Save Tie Breaker'}
					</button>
				</section>

				<section
					class="space-y-4 rounded-3xl border border-slate-700 bg-slate-900 p-6 text-slate-100"
				>
					<div class="flex items-center justify-between">
						<h2 class="text-lg font-semibold text-white">Game Insights</h2>
						<span class="text-xs tracking-wide text-slate-400 uppercase">Season {season.year}</span>
					</div>
					<ul class="space-y-3 text-sm text-slate-300">
						<li class="flex items-center justify-between">
							<span class="text-slate-200">Games this week</span>
							<span class="font-semibold text-white">{games.length}</span>
						</li>
						<li class="flex items-center justify-between">
							<span class="text-slate-200">Final games</span>
							<span class="font-semibold text-emerald-300">
								{games.filter((game) => game.status === 'final').length}
							</span>
						</li>
						<li class="flex items-center justify-between">
							<span class="text-slate-200">Your picks made</span>
							<span class="font-semibold text-emerald-200">
								{Object.keys(selections[selectedMemberId] ?? {}).length}
							</span>
						</li>
						<li class="flex items-center justify-between">
							<span class="text-slate-200">Unpicked games</span>
							<span class="font-semibold text-rose-300">
								{games.filter((game) => !selections[selectedMemberId]?.[game.gameKey]).length}
							</span>
						</li>
					</ul>

					{#if commissioner}
						<div class="space-y-3 rounded-2xl border border-slate-700 bg-slate-900 p-4">
							<h3 class="text-sm font-semibold text-white">Declare Winner</h3>
							<label class="text-xs tracking-wide text-slate-300 uppercase">
								Winner
								<select
									class="mt-1 w-full rounded-xl border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-white focus:border-emerald-400 focus:ring-2 focus:ring-emerald-400 focus:outline-none"
									bind:value={winnerMemberId}
								>
									<option value="">No winner yet</option>
									{#each members as member (member.id)}
										<option value={member.id}>{member.name}</option>
									{/each}
								</select>
							</label>
							<label class="text-xs tracking-wide text-slate-300 uppercase">
								Notes
								<textarea
									class="mt-1 w-full rounded-xl border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-white focus:border-emerald-400 focus:ring-2 focus:ring-emerald-400 focus:outline-none"
									rows="3"
									bind:value={winnerNotes}
								></textarea>
							</label>
							<button
								class="inline-flex w-full justify-center rounded-full border border-emerald-500/50 bg-emerald-500 px-4 py-2 text-xs font-semibold tracking-wide text-emerald-950 uppercase shadow hover:bg-emerald-400 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-400 disabled:cursor-not-allowed disabled:opacity-60"
								onclick={submitWinner}
								disabled={declareSaving}
							>
								{declareSaving ? 'Saving…' : 'Declare Winner'}
							</button>
						</div>
					{/if}
				</section>
			</div>
		</section>

		<section class="space-y-4 rounded-3xl border border-slate-700 bg-slate-900 p-6">
			<div class="flex flex-wrap items-center justify-between gap-3">
				<h2 class="text-lg font-semibold text-white">Family Picks Grid</h2>
				<span class="text-xs tracking-wide text-slate-400 uppercase">{season.label}</span>
			</div>
			<div
				class="overflow-x-auto rounded-2xl border border-slate-700 bg-slate-950/80 p-2 shadow-inner shadow-black/40"
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
							class="space-y-1 rounded-xl border border-slate-700 bg-slate-900 p-3 text-sm text-slate-100"
						>
							<p class="font-semibold text-white">
								{teamLabel(game, 'home')} <span class="text-emerald-300">vs</span>
								{teamLabel(game, 'away')}
							</p>
							<p class="text-xs text-slate-300">
								{formattedDate(game.kickoff)} · {game.location ?? 'Venue TBD'}
							</p>
							{#if game.status === 'final' && game.homeScore != null && game.awayScore != null}
								<p class="text-xs text-emerald-300">
									Final: {game.homeScore} - {game.awayScore}
									{#if game.winner}
										• {winnerTeamLabel(game)}
									{/if}
								</p>
							{:else}
								<p class="text-xs tracking-wide text-slate-400 uppercase">{game.status}</p>
							{/if}
						</div>
				{#each members as member (member.id)}
							{@const memberPick = game.picks.find((entry) => entry.memberId === member.id)}
							<div
								class={cellClasses(
									game,
									member.id,
									memberPick ? (memberPick.chosenSide as 'home' | 'away') : null
								)}
							>
								<p class="text-xs tracking-wide text-slate-200 uppercase">
									{memberPick ? (memberPick.chosenSide === 'home' ? 'Home' : 'Away') : 'Pending'}
								</p>
								<p class="mt-1 text-sm font-semibold text-white">
									{memberPick
										? saveLabel(game, memberPick.chosenSide as 'home' | 'away')
										: 'No pick yet'}
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
	</div>
{/if}
*** End of File
