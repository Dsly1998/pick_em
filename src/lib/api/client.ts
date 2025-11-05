import { PUBLIC_API_BASE_URL } from '$env/static/public';

const API_BASE = (PUBLIC_API_BASE_URL ?? '').replace(/\/$/, '');

function resolvePath(path: string) {
	return API_BASE ? `${API_BASE}${path}` : path;
}

async function apiFetch<T>(fetchFn: typeof fetch, path: string, init?: RequestInit): Promise<T> {
	const response = await fetchFn(resolvePath(path), {
		credentials: 'include',
		headers: {
			'Content-Type': 'application/json',
			...(init?.headers ?? {})
		},
		...init
	});

	if (!response.ok) {
		let message = `${response.status} ${response.statusText}`;
		try {
			const data = await response.json();
			if (typeof data?.error === 'string') {
				message = data.error;
			}
		} catch {
			// ignore JSON parsing failures, keep default message
		}
		throw new Error(message);
	}

	if (response.status === 204) {
		return undefined as T;
	}

	return (await response.json()) as T;
}

export type SeasonsResponse = {
	seasons: Array<{
		id: string;
		label: string;
		year: number;
		sportsDataSeasonKey: string;
	}>;
};

export type WeeksResponse = {
	weeks: Array<{
		id: string;
		number: number;
		label: string;
		startsAt?: string | null;
		endsAt?: string | null;
	}>;
};

export type PageDataResponse = {
	season: {
		id: string;
		label: string;
		year: number;
		sportsDataSeasonKey: string;
	};
	weeks: WeeksResponse['weeks'];
	activeWeek: WeeksResponse['weeks'][number];
	members: Array<{
		id: string;
		name: string;
		isCommissioner: boolean;
		seasonRecord: { wins: number; losses: number };
		lastWeekRecord: { wins: number; losses: number };
		weeksWon: number;
		tieBreakers: Record<number, number>;
	}>;
	games: Array<{
		id: string;
		gameKey: string;
		kickoff?: string | null;
		location: string;
		status: string;
		channel?: string;
		homeTeam: { code: string; name: string; location: string };
		awayTeam: { code: string; name: string; location: string };
		homeScore?: number | null;
		awayScore?: number | null;
		winner?: string | null;
		picks: Array<{
			memberId: string;
			chosenSide: string;
			status: string;
		}>;
	}>;
	weekResult?: {
		seasonWeekId: string;
		winnerMemberId?: string | null;
		declaredByMemberId?: string | null;
		notes?: string | null;
		declaredAt?: string | null;
	} | null;
};

export async function fetchSeasons(fetchFn: typeof fetch): Promise<SeasonsResponse['seasons']> {
	const { seasons } = await apiFetch<SeasonsResponse>(fetchFn, '/api/seasons');
	return seasons;
}

export async function fetchWeeks(
	fetchFn: typeof fetch,
	seasonId: string
): Promise<WeeksResponse['weeks']> {
	const { weeks } = await apiFetch<WeeksResponse>(fetchFn, `/api/seasons/${seasonId}/weeks`);
	return weeks;
}

export async function fetchPageData(
	fetchFn: typeof fetch,
	seasonId: string,
	weekNumber: number
): Promise<PageDataResponse> {
	return apiFetch<PageDataResponse>(fetchFn, `/api/seasons/${seasonId}/weeks/${weekNumber}`);
}

export async function upsertPick(
	fetchFn: typeof fetch,
	params: {
		seasonId: string;
		weekNumber: number;
		memberId: string;
		gameKey: string;
		side: 'home' | 'away';
	}
) {
	return apiFetch<{ pick: { memberId: string; chosenSide: string; status: string } }>(
		fetchFn,
		`/api/seasons/${params.seasonId}/weeks/${params.weekNumber}/picks`,
		{
			method: 'POST',
			body: JSON.stringify({
				memberId: params.memberId,
				gameKey: params.gameKey,
				side: params.side
			})
		}
	);
}

export async function upsertTieBreaker(
	fetchFn: typeof fetch,
	params: { seasonId: string; weekNumber: number; memberId: string; points: number }
) {
	return apiFetch<{ tieBreaker: { memberId: string; weekNumber: number; points: number } }>(
		fetchFn,
		`/api/seasons/${params.seasonId}/weeks/${params.weekNumber}/tie-breaker`,
		{
			method: 'POST',
			body: JSON.stringify({
				memberId: params.memberId,
				points: params.points
			})
		}
	);
}

export async function declareWeekWinner(
	fetchFn: typeof fetch,
	params: {
		seasonId: string;
		weekNumber: number;
		winnerMemberId: string | null;
		declaredByMemberId: string;
		notes?: string;
	}
) {
	return apiFetch<{ weekResult: PageDataResponse['weekResult'] }>(
		fetchFn,
		`/api/seasons/${params.seasonId}/weeks/${params.weekNumber}/winner`,
		{
			method: 'POST',
			body: JSON.stringify({
				winnerMemberId: params.winnerMemberId,
				declaredByMemberId: params.declaredByMemberId,
				notes: params.notes ?? ''
			})
		}
	);
}

export async function syncWeek(
	fetchFn: typeof fetch,
	params: { seasonId: string; weekNumber: number }
) {
	return apiFetch<{ syncedGames: number }>(
		fetchFn,
		`/api/seasons/${params.seasonId}/weeks/${params.weekNumber}/sync`,
		{ method: 'POST' }
	);
}
