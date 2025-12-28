import Link from "next/link";

import { api } from "@/lib/api";

type Submission = {
	id: number;
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
	created_at?: string;
};

const verdictStyles: Record<string, string> = {
	AC: "border-emerald-500/40 bg-emerald-500/10 text-emerald-700",
	WA: "border-amber-500/50 bg-amber-500/10 text-amber-700",
	TLE: "border-sky-500/40 bg-sky-500/10 text-sky-700",
	MLE: "border-purple-500/40 bg-purple-500/10 text-purple-700",
	RTE: "border-rose-500/40 bg-rose-500/10 text-rose-700",
};

const formatCpuTime = (value?: number) => {
	if (value === undefined || value === null) return "—";
	return `${(value / 1000).toFixed(1)} ms`;
};

const formatMemory = (value?: number) => {
	if (value === undefined || value === null) return "—";
	const mb = value / (1024 * 1024);
	return `${mb.toFixed(2)} MB`;
};

const formatTests = (passed?: number, total?: number) => {
	if (passed === undefined || total === undefined) return "—";
	return `${passed}/${total}`;
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

async function fetchSubmissions() {
	try {
		return await api.get<Submission[]>("/submissions", { cache: "no-store" });
	} catch {
		return null;
	}
}

export default async function SubmissionsPage() {
	const submissions = await fetchSubmissions();

	const sortedSubmissions =
		submissions?.slice().sort((a, b) => {
			const timeA = a.created_at ? new Date(a.created_at).getTime() : 0;
			const timeB = b.created_at ? new Date(b.created_at).getTime() : 0;
			return timeB - timeA;
		}) ?? [];

	return (
		<div className="mx-auto flex w-full max-w-6xl flex-col gap-8 px-4 py-12 sm:px-6">
			<div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
				<div className="space-y-2">
					<p className="text-xs font-semibold uppercase tracking-[0.25em] text-primary">Submissions</p>
					<h1 className="text-3xl font-bold leading-tight sm:text-4xl">All submissions</h1>
					<p className="text-sm text-muted-foreground">Latest submissions across all problems.</p>
				</div>
			</div>

			<div className="overflow-hidden border border-border/70 bg-card/70">
				{submissions ? (
					sortedSubmissions.length > 0 ? (
						<div className="overflow-x-auto">
							<table className="min-w-full divide-y divide-border/70 text-sm">
								<thead className="bg-muted/70 text-xs uppercase tracking-wide text-muted-foreground">
									<tr>
										<th className="px-4 py-3 text-left font-semibold">ID</th>
										<th className="px-4 py-3 text-left font-semibold">User</th>
										<th className="px-4 py-3 text-left font-semibold">Problem</th>
										<th className="px-4 py-3 text-left font-semibold">Verdict</th>
										<th className="px-4 py-3 text-left font-semibold">Score</th>
										<th className="px-4 py-3 text-left font-semibold">Tests</th>
										<th className="px-4 py-3 text-left font-semibold">Time</th>
										<th className="px-4 py-3 text-left font-semibold">Memory</th>
										<th className="px-4 py-3 text-left font-semibold">Language</th>
										<th className="px-4 py-3 text-left font-semibold">Submitted at</th>
										<th className="px-4 py-3 text-left font-semibold">Details</th>
									</tr>
								</thead>
								<tbody className="divide-y divide-border/70">
									{sortedSubmissions.map((submission) => {
										const verdict = submission.verdict?.toUpperCase?.() ?? "PENDING";
										const verdictClass =
											verdictStyles[verdict] ?? "border-border/70 bg-muted/50 text-foreground";

										return (
											<tr key={submission.id} className="hover:bg-muted/40">
												<td className="px-4 py-3 font-semibold text-muted-foreground">
													#{submission.id}
												</td>
												<td className="px-4 py-3 text-foreground">
													{submission.username ?? (submission.user_id ? `User #${submission.user_id}` : "—")}
												</td>
												<td className="px-4 py-3 text-muted-foreground">
													<Link
														href={`/problems/${submission.problem_id}`}
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
													{formatTests(submission.tests_passed, submission.tests_total)}
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
													{formatDate(submission.created_at)}
												</td>
												<td className="px-4 py-3">
													<Link
														href={`/submissions/${submission.id}`}
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
					) : (
						<div className="px-6 py-10 text-center text-sm text-muted-foreground">
							No submissions yet.
						</div>
					)
				) : (
					<div className="px-6 py-10 text-center text-sm text-destructive">
						Failed to load submissions.
					</div>
				)}
			</div>
		</div>
	);
}
