import type { PageLoad } from './$types';
import { fetchCurrentWeek, fetchPageData, fetchSeasons, fetchWeeks } from '$lib/api/client';

export const load = (async ({ fetch, url }) => {
	const seasons = await fetchSeasons(fetch);
	if (seasons.length === 0) {
		return {
			seasons: [],
			season: null,
			weeks: [],
			activeWeek: null,
			members: [],
			games: [],
			weekResult: null,
			selectedSeasonId: null,
			selectedWeekNumber: null
		};
	}

	const seasonParam = url.searchParams.get('season');
	const selectedSeasonId = seasons.find((season) => season.id === seasonParam)?.id ?? seasons[0].id;

	const weeks = await fetchWeeks(fetch, selectedSeasonId);
	const weekParam = url.searchParams.get('week');
	const initialWeekNumber = weeks[0]?.number ?? 1;
	let selectedWeekNumber = initialWeekNumber;
	if (weekParam) {
		const parsed = Number.parseInt(weekParam, 10);
		if (!Number.isNaN(parsed) && parsed > 0) {
			selectedWeekNumber = parsed;
		}
	} else {
		try {
			const currentWeek = await fetchCurrentWeek(fetch, selectedSeasonId);
			if (
				currentWeek != null &&
				weeks.some((week) => week.number === currentWeek) &&
				currentWeek !== selectedWeekNumber
			) {
				selectedWeekNumber = currentWeek;
			}
		} catch (error) {
			console.error('Unable to load current week, falling back to default week.', error);
		}
	}

	const pageData = await fetchPageData(fetch, selectedSeasonId, selectedWeekNumber);

	return {
		seasons,
		selectedSeasonId,
		selectedWeekNumber: pageData.activeWeek.number,
		season: pageData.season,
		weeks: pageData.weeks,
		activeWeek: pageData.activeWeek,
		members: pageData.members,
		games: pageData.games,
		weekResult: pageData.weekResult ?? null
	};
}) satisfies PageLoad;
