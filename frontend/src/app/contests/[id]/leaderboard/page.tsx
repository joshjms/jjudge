import { api } from "@/lib/api";

type ContestProblem = {
	contest_id: number;
	problem_id: number;
	ordinal: number;
	max_points: number;
	problem?: { id: number; title?: string };
};

type Contest = {
	id: number;
	title: string;
	scoring_type: "icpc" | "ioi";
	problems?: ContestProblem[];
};

type ContestProblemResult = {
	problem_id: number;
	score: number;
	accepted: boolean;
	attempts: number;
	penalty_seconds: number;
};

type LeaderboardEntry = {
	rank: number;
	user_id: number;
	username: string;
	total_score: number;
	penalty_seconds: number;
	problem_results: Record<string, ContestProblemResult>;
};

type LeaderboardResponse = {
	entries: LeaderboardEntry[];
};

export const dynamic = "force-dynamic";

async function fetchContest(id: string): Promise<Contest | null> {
	try {
		return await api.get<Contest>(`/contests/${id}`, { cache: "no-store" });
	} catch {
		return null;
	}
}

async function fetchLeaderboard(id: string): Promise<LeaderboardResponse | null> {
	try {
		return await api.get<LeaderboardResponse>(`/contests/${id}/leaderboard`, {
			cache: "no-store",
		});
	} catch {
		return null;
	}
}

function ordinalLabel(ordinal: number): string {
	if (ordinal < 26) return String.fromCharCode(65 + ordinal);
	return (
		String.fromCharCode(65 + Math.floor(ordinal / 26) - 1) +
		String.fromCharCode(65 + (ordinal % 26))
	);
}

function formatPenalty(seconds: number): string {
	const h = Math.floor(seconds / 3600);
	const m = Math.floor((seconds % 3600) / 60);
	const s = seconds % 60;
	if (h > 0) return `${h}h ${m}m`;
	if (m > 0) return `${m}m ${s}s`;
	return `${s}s`;
}

export async function generateMetadata({ params }: { params: Promise<{ id: string }> }) {
	const { id } = await params;
	const contest = await fetchContest(id);
	if (!contest) return { title: "Leaderboard" };
	return { title: `Leaderboard · ${contest.title}` };
}

export default async function LeaderboardPage({ params }: { params: Promise<{ id: string }> }) {
	const { id } = await params;
	const [contest, leaderboard] = await Promise.all([fetchContest(id), fetchLeaderboard(id)]);

	const sortedProblems = (contest?.problems ?? []).slice().sort((a, b) => a.ordinal - b.ordinal);
	const entries = leaderboard?.entries ?? [];

	return (
		<div className="mx-auto flex w-full max-w-6xl flex-col gap-8 px-4 py-12 sm:px-6">
			<div className="space-y-2">
				<p className="text-xs font-semibold uppercase tracking-[0.25em] text-primary">Contest</p>
				<h1 className="text-3xl font-bold leading-tight sm:text-4xl">
					{contest?.title ?? `Contest ${id}`} — Leaderboard
				</h1>
				{contest && (
					<p className="text-sm text-muted-foreground">
						Scoring: {contest.scoring_type === "icpc" ? "ICPC (accepted problems + penalty)" : "IOI (partial score sum)"}
					</p>
				)}
			</div>

			<div className="overflow-hidden border border-border/70">
				{leaderboard === null ? (
					<div className="px-6 py-10 text-center text-sm text-destructive">
						Failed to load leaderboard.
					</div>
				) : entries.length === 0 ? (
					<div className="px-6 py-10 text-center text-sm text-muted-foreground">
						No submissions yet.
					</div>
				) : (
					<div className="overflow-x-auto">
						<table className="min-w-full divide-y divide-border/70 text-sm">
							<thead className="bg-muted/70 text-xs uppercase tracking-wide text-muted-foreground">
								<tr>
									<th className="px-4 py-3 text-left font-semibold">Rank</th>
									<th className="px-4 py-3 text-left font-semibold">User</th>
									<th className="px-4 py-3 text-left font-semibold">
										{contest?.scoring_type === "icpc" ? "Solved" : "Score"}
									</th>
									{contest?.scoring_type === "icpc" && (
										<th className="px-4 py-3 text-left font-semibold">Penalty</th>
									)}
									{sortedProblems.map((cp) => (
										<th key={cp.problem_id} className="px-4 py-3 text-center font-semibold">
											{ordinalLabel(cp.ordinal)}
										</th>
									))}
								</tr>
							</thead>
							<tbody className="divide-y divide-border/70">
								{entries.map((entry) => (
									<tr key={entry.user_id} className="hover:bg-muted/40">
										<td className="px-4 py-3 font-bold text-muted-foreground">
											#{entry.rank}
										</td>
										<td className="px-4 py-3 font-semibold">{entry.username}</td>
										<td className="px-4 py-3 font-semibold text-foreground">
											{entry.total_score}
										</td>
										{contest?.scoring_type === "icpc" && (
											<td className="px-4 py-3 text-muted-foreground">
												{entry.penalty_seconds > 0
													? formatPenalty(entry.penalty_seconds)
													: "—"}
											</td>
										)}
										{sortedProblems.map((cp) => {
											const pr = entry.problem_results?.[String(cp.problem_id)];
											if (!pr)
												return (
													<td key={cp.problem_id} className="px-4 py-3 text-center text-muted-foreground">
														—
													</td>
												);
											if (contest?.scoring_type === "icpc") {
												return (
													<td key={cp.problem_id} className="px-4 py-3 text-center">
														{pr.accepted ? (
															<span className="text-emerald-700 font-semibold">
																+{pr.attempts}
															</span>
														) : (
															<span className="text-rose-700">
																-{pr.attempts}
															</span>
														)}
													</td>
												);
											}
											return (
												<td key={cp.problem_id} className="px-4 py-3 text-center">
													<span className={pr.score > 0 ? "text-emerald-700 font-semibold" : "text-muted-foreground"}>
														{pr.score}
													</span>
												</td>
											);
										})}
									</tr>
								))}
							</tbody>
						</table>
					</div>
				)}
			</div>
		</div>
	);
}
