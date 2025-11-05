export type Season = {
	id: string;
	year: number;
	label: string;
};

export type Week = {
	number: number;
	label: string;
};

export type RecordSummary = {
	wins: number;
	losses: number;
};

export type Member = {
	id: string;
	name: string;
	isCommissioner: boolean;
	seasonRecord: RecordSummary;
	lastWeekRecord: RecordSummary;
	weeksWon: number;
	tieBreakers: {
		[weekNumber: number]: number;
	};
};

export type TeamInfo = {
	code: string;
	name: string;
	location: string;
};

export type PickStatus = 'pending' | 'correct' | 'incorrect';

export type GamePick = {
	memberId: string;
	chosenSide: 'home' | 'away';
	status: PickStatus;
};

export type Game = {
	id: string;
	gameKey: string;
	kickoff?: string | null;
	location: string;
	status: 'scheduled' | 'in-progress' | 'final';
	channel?: string | null;
	homeTeam: TeamInfo;
	awayTeam: TeamInfo;
	homeScore?: number | null;
	awayScore?: number | null;
	winner?: 'home' | 'away' | null;
	picks: GamePick[];
};

export type PicksPageData = {
	season: Season;
	weeks: Week[];
	activeWeek: Week;
	members: Member[];
	games: Game[];
};

export type HomePageData = {
	upcomingWeek: Week;
	season: Season;
	members: Member[];
	games: Game[];
};
