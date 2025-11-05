import type { Game, HomePageData, PicksPageData, RecordSummary, TeamInfo, Week } from '$lib/types';

const familyMembers = [
	{
		id: 'member-dallin',
		name: 'Dallin',
		isCommissioner: false,
		seasonRecord: { wins: 44, losses: 18 },
		lastWeekRecord: { wins: 10, losses: 4 },
		weeksWon: 4,
		tieBreakers: {
			5: 46
		}
	},
	{
		id: 'member-danielle',
		name: 'Danielle',
		isCommissioner: false,
		seasonRecord: { wins: 41, losses: 21 },
		lastWeekRecord: { wins: 9, losses: 5 },
		weeksWon: 3,
		tieBreakers: {
			5: 43
		}
	},
	{
		id: 'member-lauren',
		name: 'Lauren',
		isCommissioner: false,
		seasonRecord: { wins: 38, losses: 24 },
		lastWeekRecord: { wins: 8, losses: 6 },
		weeksWon: 2,
		tieBreakers: {
			5: 48
		}
	},
	{
		id: 'member-brad',
		name: 'Brad',
		isCommissioner: true,
		seasonRecord: { wins: 36, losses: 26 },
		lastWeekRecord: { wins: 7, losses: 7 },
		weeksWon: 2,
		tieBreakers: {
			5: 50
		}
	},
	{
		id: 'member-dad',
		name: 'Dad',
		isCommissioner: false,
		seasonRecord: { wins: 34, losses: 28 },
		lastWeekRecord: { wins: 7, losses: 7 },
		weeksWon: 1,
		tieBreakers: {
			5: 41
		}
	},
	{
		id: 'member-mom',
		name: 'Mom',
		isCommissioner: false,
		seasonRecord: { wins: 32, losses: 30 },
		lastWeekRecord: { wins: 6, losses: 8 },
		weeksWon: 1,
		tieBreakers: {
			5: 39
		}
	}
];

const season = {
	id: '2025REG',
	year: 2025,
	label: '2025 Regular Season'
};

const weeks: Week[] = [
	{ number: 4, label: 'Week 4' },
	{ number: 5, label: 'Week 5' },
	{ number: 6, label: 'Week 6' }
];

const activeWeekNumber = 5;

const games: Game[] = [
	makeGame({
		gameKey: '202510501',
		location: 'Tottenham Hotspur Stadium (London, UK)',
		status: 'final',
		kickoff: '2025-10-05T13:30:00Z',
		channel: 'NFLN',
		home: team('CLE', 'Cleveland Browns', 'Cleveland'),
		away: team('MIN', 'Minnesota Vikings', 'Minnesota'),
		winner: 'away',
		homeScore: 20,
		awayScore: 24
	}),
	makeGame({
		gameKey: '202510502',
		location: 'SoFi Stadium (Inglewood, CA)',
		status: 'scheduled',
		kickoff: '2025-10-05T17:00:00Z',
		channel: 'FOX',
		home: team('LAR', 'Los Angeles Rams', 'Los Angeles'),
		away: team('SEA', 'Seattle Seahawks', 'Seattle')
	}),
	makeGame({
		gameKey: '202510503',
		location: 'Arrowhead Stadium (Kansas City, MO)',
		status: 'scheduled',
		kickoff: '2025-10-05T20:25:00Z',
		channel: 'CBS',
		home: team('KC', 'Kansas City Chiefs', 'Kansas City'),
		away: team('BUF', 'Buffalo Bills', 'Buffalo')
	}),
	makeGame({
		gameKey: '202510504',
		location: 'Lumen Field (Seattle, WA)',
		status: 'scheduled',
		kickoff: '2025-10-06T00:20:00Z',
		channel: 'NBC',
		home: team('SEA', 'Seattle Seahawks', 'Seattle'),
		away: team('SF', 'San Francisco 49ers', 'San Francisco')
	}),
	makeGame({
		gameKey: '202510505',
		location: 'Lambeau Field (Green Bay, WI)',
		status: 'scheduled',
		kickoff: '2025-10-06T23:15:00Z',
		channel: 'ESPN',
		home: team('GB', 'Green Bay Packers', 'Green Bay'),
		away: team('CHI', 'Chicago Bears', 'Chicago')
	})
];

const picksByGame = new Map<string, Game['picks']>([
	[
		'202510501',
		[
			pick('member-dallin', 'away', 'correct'),
			pick('member-danielle', 'home', 'incorrect'),
			pick('member-lauren', 'away', 'correct'),
			pick('member-brad', 'home', 'incorrect'),
			pick('member-dad', 'away', 'correct'),
			pick('member-mom', 'home', 'incorrect')
		]
	],
	[
		'202510502',
		[
			pick('member-dallin', 'home'),
			pick('member-danielle', 'home'),
			pick('member-lauren', 'away'),
			pick('member-brad', 'home'),
			pick('member-dad', 'away'),
			pick('member-mom', 'home')
		]
	],
	[
		'202510503',
		[
			pick('member-dallin', 'home'),
			pick('member-danielle', 'away'),
			pick('member-lauren', 'away'),
			pick('member-brad', 'home'),
			pick('member-dad', 'home'),
			pick('member-mom', 'away')
		]
	],
	[
		'202510504',
		[
			pick('member-dallin', 'away'),
			pick('member-danielle', 'home'),
			pick('member-lauren', 'home'),
			pick('member-brad', 'away'),
			pick('member-dad', 'home'),
			pick('member-mom', 'home')
		]
	],
	[
		'202510505',
		[
			pick('member-dallin', 'home'),
			pick('member-danielle', 'away'),
			pick('member-lauren', 'home'),
			pick('member-brad', 'home'),
			pick('member-dad', 'away'),
			pick('member-mom', 'home')
		]
	]
]);

for (const game of games) {
	game.picks = picksByGame.get(game.gameKey) ?? [];
}

export const mockPicksPageData: PicksPageData = {
	season,
	weeks,
	activeWeek: weeks.find((week) => week.number === activeWeekNumber) ?? weeks[0],
	members: familyMembers,
	games
};

export const mockHomePageData: HomePageData = {
	season,
	upcomingWeek: weeks.find((week) => week.number === activeWeekNumber) ?? weeks[0],
	members: familyMembers,
	games
};

function team(code: string, name: string, location: string): TeamInfo {
	return { code, name, location };
}

function pick(
	memberId: string,
	chosenSide: 'home' | 'away',
	status: 'pending' | 'correct' | 'incorrect' = 'pending'
) {
	return { memberId, chosenSide, status };
}

function makeGame({
	gameKey,
	location,
	status,
	kickoff,
	channel,
	home,
	away,
	winner,
	homeScore,
	awayScore
}: {
	gameKey: string;
	location: string;
	status: Game['status'];
	kickoff: string;
	channel?: string;
	home: TeamInfo;
	away: TeamInfo;
	winner?: Game['winner'];
	homeScore?: number;
	awayScore?: number;
}): Game {
	return {
		gameKey,
		location,
		status,
		kickoff,
		channel,
		home,
		away,
		homeScore,
		awayScore,
		winner,
		picks: []
	};
}

export function formatRecord({ wins, losses }: RecordSummary): string {
	return `${wins}-${losses}`;
}
