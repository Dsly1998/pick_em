<script lang="ts">
	import { goto } from '$app/navigation';
	import {
		clearPick,
		declareWeekWinner,
		syncWeek,
		upsertPick,
		upsertTieBreaker
	} from '$lib/api/client';
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
	let gamesView = $state<Game[]>([]);
	const weekResult = $derived(data.weekResult ?? null);

	const commissionerName = 'Brad';
	const commissioner = $derived(members.find((member) => member.isCommissioner) ?? null);

	const STORAGE_SEASON_KEY = 'bdp:selectedSeason';
	const STORAGE_WEEK_KEY = 'bdp:selectedWeek';
	const STORAGE_MEMBER_KEY = 'bdp:selectedMember';

	const initialSeasonId = data.selectedSeasonId ?? seasons[0]?.id ?? '';
	const initialWeekNumber = data.selectedWeekNumber ?? activeWeek?.number ?? weeks[0]?.number ?? 1;
	const initialMemberId = members[0]?.id ?? '';

	let selectedSeasonId = $state(initialSeasonId);
	let selectedWeekValue = $state(String(initialWeekNumber));
	let selectedMemberId = $state(initialMemberId);
	let selections = $state({} as Record<string, Record<string, 'home' | 'away'>>);
	let tieBreakerInputs = $state({} as Record<string, Record<number, string>>);
	let tieBreakerSaving = $state(false);
	let declareSaving = $state(false);
	let syncSaving = $state(false);
	let winnerMemberId = $state('');
	let winnerNotes = $state('');
	let restoredFromStorage = $state(false);

	$effect(() => {
		const nextSeason = data.selectedSeasonId ?? seasons[0]?.id ?? '';
		const currentSeasonParam =
			typeof window !== 'undefined'
				? (new URL(window.location.href).searchParams.get('season') ?? '')
				: '';
		if (currentSeasonParam && nextSeason && currentSeasonParam !== nextSeason) {
			return;
		}
		if (nextSeason && nextSeason !== selectedSeasonId) {
			selectedSeasonId = nextSeason;
		}
		const weekNumber = data.selectedWeekNumber ?? activeWeek?.number ?? weeks[0]?.number ?? 1;
		const nextWeekValue = String(weekNumber);
		const currentWeekParam =
			typeof window !== 'undefined'
				? (new URL(window.location.href).searchParams.get('week') ?? '')
				: '';
		if (currentWeekParam && nextWeekValue && currentWeekParam !== nextWeekValue) {
			return;
		}
		if (nextWeekValue !== selectedWeekValue) {
			selectedWeekValue = nextWeekValue;
		}
		if (!members.some((member) => member.id === selectedMemberId)) {
			selectedMemberId = members[0]?.id ?? '';
		}
		selections = buildInitialSelections();
		tieBreakerInputs = buildInitialTieBreakers();
		winnerMemberId = weekResult?.winnerMemberId ?? '';
		winnerNotes = weekResult?.notes ?? '';
	});

		$effect(() => {
			gamesView = games.map((game) => prepareGame(game));
		});

	const selectedMember = $derived(
		members.find((member) => member.id === selectedMemberId) ?? members[0] ?? null
	);

	const familyGridMemberCount = $derived(Math.max(members.length, 1));

	let selectedWeekNumber = $state(1);
	let currentTieBreaker = $state('');

	$effect(() => {
		const parsed = Number.parseInt(selectedWeekValue, 10);
		selectedWeekNumber =
			!Number.isNaN(parsed) && parsed > 0 ? parsed : (activeWeek?.number ?? weeks[0]?.number ?? 1);
	});

	$effect(() => {
		if (!selectedMember || !activeWeek) {
			currentTieBreaker = '';
			return;
		}
		currentTieBreaker = tieBreakerInputs[selectedMember.id]?.[activeWeek.number] ?? '';
	});

	$effect(() => {
		if (typeof window === 'undefined' || !restoredFromStorage) return;
		if (selectedSeasonId) {
			window.localStorage.setItem(STORAGE_SEASON_KEY, selectedSeasonId);
		}
		if (selectedWeekValue) {
			window.localStorage.setItem(STORAGE_WEEK_KEY, selectedWeekValue);
		}
		if (selectedMemberId) {
			window.localStorage.setItem(STORAGE_MEMBER_KEY, selectedMemberId);
		}
	});

	$effect(() => {
		if (typeof window === 'undefined' || restoredFromStorage) return;
		const url = new URL(window.location.href);
		const hasSeasonParam = url.searchParams.has('season');
		const hasWeekParam = url.searchParams.has('week');

		const storedSeason = window.localStorage.getItem(STORAGE_SEASON_KEY);
		const storedWeek = window.localStorage.getItem(STORAGE_WEEK_KEY);
		const storedMember = window.localStorage.getItem(STORAGE_MEMBER_KEY);

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

		if (storedMember && members.some((member) => member.id === storedMember)) {
			selectedMemberId = storedMember;
		}

		restoredFromStorage = true;

		if (shouldNavigate) {
			const parsed = Number.parseInt(weekToUse, 10);
			navigate(seasonToUse, Number.isNaN(parsed) ? undefined : parsed);
		}
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
		return gameIsFinal(game) && game.winner === side;
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

	function normalizeStatus(status?: string | null) {
		return (status ?? '').toString().toLowerCase();
	}

	function normalizeGameStatus(status?: string | null): 'scheduled' | 'in-progress' | 'final' {
		const normalized = normalizeStatus(status);
		if (normalized === 'in-progress') return 'in-progress';
		if (normalized === 'final') return 'final';
		return 'scheduled';
	}

	function normalizePickStatus(
		status?: string | null
	): 'pending' | 'correct' | 'incorrect' | null {
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

	function deriveWinner(game: Game): 'home' | 'away' | null {
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

	function enrichGame(game: Game): Game {
		const status = normalizeGameStatus(game.status);
		const winner = deriveWinner(game);
		return {
			...game,
			status,
			winner
		};
	}

	function prepareGame(game: Game): Game {
		const enriched = enrichGame(game);
		return {
			...enriched,
			picks: (game.picks ?? []).map((pick) => {
				const side = normalizeSide(pick.chosenSide) ?? 'home';
				const normalized = normalizePickStatus(pick.status);
				return {
					...pick,
					chosenSide: side,
					status: normalized ?? determinePickStatus(enriched, side)
				};
			})
		};
	}

	function gameIsFinal(game: Game) {
		return game.status === 'final';
	}

	function gameInProgress(game: Game) {
		return game.status === 'in-progress';
	}

	function teamLabel(game: Game, side: 'home' | 'away') {
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

	function teamNameOnly(game: Game, side: 'home' | 'away') {
		const team = side === 'home' ? game.homeTeam : game.awayTeam;
		if (team?.name && team.name.trim().length > 0) {
			return team.name.trim();
		}
		if (team?.code && team.code.trim().length > 0) {
			return team.code.toUpperCase();
		}
		return teamLabel(game, side);
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
		const previous = memberSelections[gameKey] ?? null;

		// Clicking the same side again removes the pick.
		if (previous === side) {
			delete memberSelections[gameKey];
			selections = { ...selections, [selectedMemberId]: memberSelections };
			try {
				await clearPick(fetch, {
					seasonId: season.id,
					weekNumber: activeWeek.number,
					memberId: selectedMemberId,
					gameKey
				});
				applyLocalPick(gameKey, selectedMemberId, null);
			} catch (error) {
				if (previous) {
					memberSelections[gameKey] = previous;
				}
				selections = { ...selections, [selectedMemberId]: memberSelections };
				console.error(error);
				alert('Unable to clear pick. Please try again.');
			}
			return;
		}

		memberSelections[gameKey] = side;
		selections = { ...selections, [selectedMemberId]: memberSelections };
			try {
				const { pick } = await upsertPick(fetch, {
					seasonId: season.id,
					weekNumber: activeWeek.number,
					memberId: selectedMemberId,
					gameKey,
					side
				});
				if (pick) {
					let sourceGame: Game | null =
						gamesView.find((entry) => entry.gameKey === gameKey) ?? null;
					if (!sourceGame) {
						const raw = games.find((entry) => entry.gameKey === gameKey) ?? null;
						sourceGame = raw ? prepareGame(raw) : null;
					}
					const chosenSide = (pick.chosenSide as 'home' | 'away') ?? 'home';
					let normalizedStatus = normalizePickStatus(pick.status);
					if (!normalizedStatus) {
						if (sourceGame && gameIsFinal(sourceGame)) {
							normalizedStatus = sourceGame.winner === chosenSide ? 'correct' : 'incorrect';
						} else {
							normalizedStatus = 'pending';
						}
					}
					applyLocalPick(gameKey, selectedMemberId, {
						memberId: pick.memberId,
						chosenSide,
						status: normalizedStatus
					});
				} else {
				applyLocalPick(gameKey, selectedMemberId, {
					memberId: selectedMemberId,
					chosenSide: side
				});
			}
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
				'border-emerald-400',
				'bg-emerald-500/25',
				'text-emerald-50',
				'shadow-lg',
				'shadow-emerald-600/30',
				'ring-1',
				'ring-emerald-300/60'
			);
		}

		if (gameIsFinal(game)) {
			if (pickIsCorrect(game, side)) {
				classes.push('border-emerald-400', 'bg-emerald-500/20', 'text-emerald-100');
			} else if (game.winner) {
				classes.push('border-rose-400/60', 'bg-rose-500/20', 'text-rose-100/90');
			}
		}

		return classes.join(' ');
	}

	function pickOutcome(
		game: Game,
		pick: Game['picks'][number] | null
	): 'win' | 'loss' | 'pending' | 'none' {
		if (!pick) return 'none';
		const winner = deriveWinner(game);
		if (!winner || !gameIsFinal(game)) {
			return 'pending';
		}
		return pick.chosenSide === winner ? 'win' : 'loss';
	}

	function cellClasses(game: Game, memberId: string, pick: Game['picks'][number] | null) {
		const classes = [
			'family-grid__cell',
			'rounded-md',
			'border',
			'text-center',
			'text-white',
			'font-medium',
			'transition-colors',
			'duration-150'
		];

		const outcome = pickOutcome(game, pick);

		if (outcome === 'win') {
			classes.push('bg-emerald-500/25', 'border-emerald-400/70', 'font-semibold');
		} else if (outcome === 'loss') {
			classes.push('bg-rose-500/25', 'border-rose-400/70', 'font-semibold');
		} else {
			classes.push('bg-slate-900/60', 'border-slate-700/70');
			if (!pick) {
				classes.push('italic', 'text-white/70');
			}
		}

		if (memberId === selectedMemberId) {
			classes.push('ring-1', 'ring-emerald-300/60');
		}

		return classes.join(' ');
	}

	function saveLabel(game: Game, side: 'home' | 'away') {
		return teamLabel(game, side);
	}

	function determinePickStatus(
		game: Game,
		side: 'home' | 'away'
	): 'pending' | 'correct' | 'incorrect' {
		const winner = deriveWinner(game);
		if (gameIsFinal(game) && winner) {
			return winner === side ? 'correct' : 'incorrect';
		}
		return 'pending';
	}

	function applyLocalPick(
		gameKey: string,
		memberId: string,
		pick: {
			memberId: string;
			chosenSide: 'home' | 'away';
			status?: 'pending' | 'correct' | 'incorrect';
		} | null
	) {
		gamesView = gamesView.map((game) => {
			if (game.gameKey !== gameKey) {
				return game;
			}
			const next = { ...game };
			const picks = next.picks.filter((entry) => entry.memberId !== memberId);
			if (pick) {
				const status =
					normalizePickStatus(pick.status) ?? determinePickStatus(next, pick.chosenSide);
				picks.push({
					memberId,
					chosenSide: pick.chosenSide,
					status
				});
			}
			next.picks = picks;
			return next;
		});
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
				<div class="text-xs font-semibold tracking-wide text-slate-200 uppercase">
					Season
					<div
						class="mt-1 w-full rounded-xl border border-slate-700 bg-slate-900 px-4 py-2 text-sm text-white"
					>
						{season.label} · {season.year}
					</div>
				</div>
				<div class="text-xs font-semibold tracking-wide text-slate-200 uppercase">
					Week
					<div
						class="mt-1 w-full rounded-xl border border-slate-700 bg-slate-900 px-4 py-2 text-sm text-white"
					>
						Week {selectedWeekNumber}
					</div>
				</div>
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
					{#each gamesView as game (game.id)}
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
								{#if gameIsFinal(game) && game.winner}
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
									{#if gameIsFinal(game) && game.winner}
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
							<span class="text-slate-200">Viewing week</span>
							<span class="font-semibold text-white">Week {selectedWeekNumber}</span>
						</li>
						<li class="flex items-center justify-between">
							<span class="text-slate-200">Games this week</span>
							<span class="font-semibold text-white">{gamesView.length}</span>
						</li>
						<li class="flex items-center justify-between">
							<span class="text-slate-200">Final games</span>
							<span class="font-semibold text-emerald-300">
								{gamesView.filter((game) => gameIsFinal(game)).length}
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
								{gamesView.filter((game) => !selections[selectedMemberId]?.[game.gameKey]).length}
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
				<h2 class="text-lg font-semibold text-white">
					Family Picks Grid · Week {selectedWeekNumber}
				</h2>
				<span class="text-xs tracking-wide text-slate-400 uppercase">{season.label}</span>
			</div>
			<div
				class="w-full overflow-x-auto rounded-2xl border border-slate-700 bg-slate-950/80 p-2 shadow-inner shadow-black/40"
			>
				<div
					class="family-grid inline-grid min-w-full gap-1"
					style={`--family-grid-members: ${familyGridMemberCount};`}
				>
					<div class="family-grid__header rounded-md bg-slate-800/90 text-slate-100 uppercase">
						Game
					</div>
					{#each members as member (member.id)}
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
						{#each members as member (member.id)}
					{@const memberPick = game.picks.find((entry) => entry.memberId === member.id) ?? null}
							{@const outcome = pickOutcome(game, memberPick)}
							<div class={cellClasses(game, member.id, memberPick)}>
								{#if memberPick}
									{teamLabel(game, memberPick.chosenSide as 'home' | 'away')}
								{:else}
									No pick yet
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
</style>
