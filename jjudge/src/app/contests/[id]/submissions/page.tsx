import Link from "next/link";

import { api } from "@/lib/api";

type ContestSubmission = {
	id: number;
	contest_id: number;
	problem_id: number;
	user_id?: number;
	username?: string;
	language?: string;
	verdict?: string;
	score?: number;
	cpu_time?: number;
	memory?: number;
	message?: string;
	tests_passed?: number;
	tests_total?: number;
	submitted_at?: string;
};

const verdictStyles: Record<string, string> = {
	PENDING: "border-border/70 bg-muted/50 text-muted-foreground",
	JUDGING: "border-blue-500/40 bg-blue-500/10 text-blue-700",
	AC: "border-emerald-500/40 bg-emerald-500/10 text-emerald-700",
	WA: "border-amber-500/50 bg-amber-500/10 text-amber-700",
	TLE: "border-sky-500/40 bg-sky-500/10 text-sky-700",
	MLE: "border-purple-500/40 bg-purple-500/10 text-purple-700",
	RE: "border-rose-500/40 bg-rose-500/10 text-rose-700",
	CE: "border-orange-500/40 bg-orange-500/10 text-orange-700",
	SE: "border-red-500/40 bg-red-500/10 text-red-700",
	IE: "border-red-500/40 bg-red-500/10 text-red-700",
};

const formatCpuTime = (value?: number) => {
	if (value === undefined || value === null) return "—";
	return `${value} ms`;
};

const formatMemory = (value?: number) => {
	if (value === undefined || value === null) return "—";
	const mb = value / (1024 * 1024);
	return `${mb.toFixed(2)} MB`;
};

const formatDate = (value?: string) => {
	if (!value) return "—";
	return new Intl.DateTimeFormat(undefined, {
		year: "numeric",
		month: "short",
		day: "2-digit",
		hour: "2-digit",
		minute: "2-digit",
	}).format(new Date(value));
};

export const dynamic = "force-dynamic";

async function fetchSubmissions(contestId: string): Promise<ContestSubmission[] | null> {
	try {
		return await api.get<ContestSubmission[]>(`/contests/${contestId}/submissions`, {
			cache: "no-store",
		});
	} catch {
		return null;
	}
}

export default async function ContestSubmissionsPage({
	params,
}: {
	params: Promise<{ id: string }>;
}) {
	const { id } = await params;
	const submissions = await fetchSubmissions(id);

	return (
		<div className="mx-auto flex w-full max-w-6xl flex-col gap-8 px-4 py-12 sm:px-6">
			<div className="space-y-2">
				<p className="text-xs font-semibold uppercase tracking-[0.25em] text-primary">
					Contest {id}
				</p>
				<h1 className="text-3xl font-bold leading-tight sm:text-4xl">Submissions</h1>
				<div className="flex flex-wrap gap-3">
					<Link
						href={`/contests/${id}`}
						className="text-sm text-muted-foreground underline hover:text-foreground"
					>
						← Back to contest
					</Link>
				</div>
			</div>

			<div className="overflow-hidden border border-border/70 bg-card/70">
				{submissions === null ? (
					<div className="px-6 py-10 text-center text-sm text-destructive">
						Failed to load submissions.
					</div>
				) : submissions.length === 0 ? (
					<div className="px-6 py-10 text-center text-sm text-muted-foreground">
						No submissions yet.
					</div>
				) : (
					<div className="overflow-x-auto">
						<table className="min-w-full divide-y divide-border/70 text-sm">
							<thead className="bg-muted/70 text-xs uppercase tracking-wide text-muted-foreground">
								<tr>
									<th className="px-4 py-3 text-left font-semibold">ID</th>
									<th className="px-4 py-3 text-left font-semibold">User</th>
									<th className="px-4 py-3 text-left font-semibold">Problem</th>
									<th className="px-4 py-3 text-left font-semibold">Verdict</th>
									<th className="px-4 py-3 text-left font-semibold">Score</th>
									<th className="px-4 py-3 text-left font-semibold">Time</th>
									<th className="px-4 py-3 text-left font-semibold">Memory</th>
									<th className="px-4 py-3 text-left font-semibold">Language</th>
									<th className="px-4 py-3 text-left font-semibold">Submitted at</th>
									<th className="px-4 py-3 text-left font-semibold">Details</th>
								</tr>
							</thead>
							<tbody className="divide-y divide-border/70">
								{submissions.map((submission) => {
									const verdict = submission.verdict?.toUpperCase?.() ?? "PENDING";
									const verdictClass =
										verdictStyles[verdict] ?? "border-border/70 bg-muted/50 text-foreground";

									return (
										<tr key={submission.id} className="hover:bg-muted/40">
											<td className="px-4 py-3 font-semibold text-muted-foreground">
												#{submission.id}
											</td>
											<td className="px-4 py-3">
												{submission.username ??
													(submission.user_id ? `User #${submission.user_id}` : "—")}
											</td>
											<td className="px-4 py-3 text-muted-foreground">
												<Link
													href={`/contests/${id}/problems/${submission.problem_id}`}
													className="border border-border/70 px-2 py-1 text-xs font-semibold transition hover:border-primary/60 hover:bg-muted/60"
												>
													Problem {submission.problem_id}
												</Link>
											</td>
											<td className="px-4 py-3">
												<span
													className={`inline-flex border px-3 py-1 text-xs font-semibold uppercase ${verdictClass}`}
												>
													{verdict}
												</span>
												{submission.message && (
													<p className="mt-1 text-[11px] text-muted-foreground">
														{submission.message}
													</p>
												)}
											</td>
											<td className="px-4 py-3 text-muted-foreground">
												{submission.score ?? "—"}
											</td>
											<td className="px-4 py-3 text-muted-foreground">
												{formatCpuTime(submission.cpu_time)}
											</td>
											<td className="px-4 py-3 text-muted-foreground">
												{formatMemory(submission.memory)}
											</td>
											<td className="px-4 py-3 text-muted-foreground">
												{submission.language?.toUpperCase?.() ?? "—"}
											</td>
											<td className="px-4 py-3 text-muted-foreground">
												{formatDate(submission.submitted_at)}
											</td>
											<td className="px-4 py-3">
												<Link
													href={`/contests/${id}/submissions/${submission.id}`}
													className="border border-border/70 px-3 py-1 text-xs font-semibold transition hover:border-primary/60 hover:bg-muted/60"
												>
													View
												</Link>
											</td>
										</tr>
									);
								})}
							</tbody>
						</table>
					</div>
				)}
			</div>
		</div>
	);
}
