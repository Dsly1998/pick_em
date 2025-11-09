export type RemainingGame = {
	gameKey: string;
	picks: Array<{
		memberId: string;
		side: 'home' | 'away';
	}>;
};

export function computeContenders(
	members: Array<{ id: string; name?: string }>,
	currentWins: Record<string, number>,
	remainingGames: RemainingGame[],
	allowTies = true
): Record<string, boolean> {
	const memberIds = members.map((member) => member.id);
	const indexById = new Map(memberIds.map((id, index) => [id, index]));
	const possible = new Array(memberIds.length).fill(false);

	const initialWins = memberIds.map((id) => currentWins[id] ?? 0);

	const outcomes: Array<'home' | 'away'> = ['home', 'away'];

	function dfs(gameIndex: number, wins: number[]) {
		if (gameIndex >= remainingGames.length) {
			const topScore = Math.max(...wins);
			const leaderIndexes = wins
				.map((score, index) => ({ score, index }))
				.filter(({ score }) => score === topScore)
				.map(({ index }) => index);

			if (leaderIndexes.length === 0) return;

			if (allowTies) {
				for (const idx of leaderIndexes) {
					possible[idx] = true;
				}
			} else if (leaderIndexes.length === 1) {
				possible[leaderIndexes[0]] = true;
			}
			return;
		}

		const game = remainingGames[gameIndex];
		for (const winnerSide of outcomes) {
			const nextWins = wins.slice();
			for (const pick of game.picks) {
				if (pick.side !== winnerSide) continue;
				const memberIndex = indexById.get(pick.memberId);
				if (memberIndex == null) continue;
				nextWins[memberIndex] += 1;
			}
			dfs(gameIndex + 1, nextWins);
		}
	}

	dfs(0, initialWins);

	const result: Record<string, boolean> = {};
	memberIds.forEach((id, index) => {
		result[id] = possible[index];
	});
	return result;
}
