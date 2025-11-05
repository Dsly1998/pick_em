export function formatRecord(record: { wins: number; losses: number }): string {
	return `${record.wins}-${record.losses}`;
}
