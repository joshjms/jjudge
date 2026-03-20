import Link from "next/link";

import { api } from "@/lib/api";

type Submission = {
	id: number;
	problem_id: number;
	problem_title?: string;
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
	PENDING: "text-muted-foreground border-border/60",
	JUDGING: "text-blue-600 border-blue-500/40 bg-blue-500/5",
	AC: "text-emerald-600 border-emerald-500/40 bg-emerald-500/5",
	WA: "text-amber-600 border-amber-500/40 bg-amber-500/5",
	TLE: "text-sky-600 border-sky-500/40 bg-sky-500/5",
	MLE: "text-purple-600 border-purple-500/40 bg-purple-500/5",
	RE: "text-rose-600 border-rose-500/40 bg-rose-500/5",
	CE: "text-orange-600 border-orange-500/40 bg-orange-500/5",
	SE: "text-red-600 border-red-500/40 bg-red-500/5",
	IE: "text-red-600 border-red-500/40 bg-red-500/5",
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
		<section className="mx-auto max-w-6xl px-6 py-10">
			{/* Header */}
			<div className="mb-8 flex items-baseline gap-4 border-b border-border/60 pb-5">
				<h1 className="font-display text-5xl text-foreground">SUBMISSIONS</h1>
				{submissions && (
					<span className="text-xs font-mono text-muted-foreground tracking-widest">
						{sortedSubmissions.length} TOTAL
					</span>
				)}
			</div>

			{submissions === null ? (
				<p className="text-sm font-mono text-muted-foreground">
					Failed to load submissions. Check that the API server is running.
				</p>
			) : sortedSubmissions.length === 0 ? (
				<div className="flex flex-col items-center gap-3 py-20 text-center">
					<span className="font-display text-4xl text-muted-foreground/30">NO SUBMISSIONS</span>
					<p className="text-xs font-mono text-muted-foreground/50">No submissions yet.</p>
				</div>
			) : (
				<div className="overflow-x-auto">
					<table className="min-w-full text-sm">
						<thead>
							<tr className="border-b border-border/60">
								<th className="px-3 py-2.5 text-left text-[10px] font-mono font-semibold uppercase tracking-widest text-muted-foreground/60">ID</th>
								<th className="px-3 py-2.5 text-left text-[10px] font-mono font-semibold uppercase tracking-widest text-muted-foreground/60">User</th>
								<th className="px-3 py-2.5 text-left text-[10px] font-mono font-semibold uppercase tracking-widest text-muted-foreground/60">Problem</th>
								<th className="px-3 py-2.5 text-left text-[10px] font-mono font-semibold uppercase tracking-widest text-muted-foreground/60">Verdict</th>
								<th className="px-3 py-2.5 text-left text-[10px] font-mono font-semibold uppercase tracking-widest text-muted-foreground/60">Score</th>
								<th className="px-3 py-2.5 text-left text-[10px] font-mono font-semibold uppercase tracking-widest text-muted-foreground/60">Tests</th>
								<th className="px-3 py-2.5 text-left text-[10px] font-mono font-semibold uppercase tracking-widest text-muted-foreground/60">Time</th>
								<th className="px-3 py-2.5 text-left text-[10px] font-mono font-semibold uppercase tracking-widest text-muted-foreground/60">Memory</th>
								<th className="px-3 py-2.5 text-left text-[10px] font-mono font-semibold uppercase tracking-widest text-muted-foreground/60">Lang</th>
								<th className="px-3 py-2.5 text-left text-[10px] font-mono font-semibold uppercase tracking-widest text-muted-foreground/60">Submitted</th>
								<th className="px-3 py-2.5 text-left text-[10px] font-mono font-semibold uppercase tracking-widest text-muted-foreground/60"></th>
							</tr>
						</thead>
						<tbody className="divide-y divide-border/40">
							{sortedSubmissions.map((submission) => {
								const verdict = submission.verdict?.toUpperCase?.() ?? "PENDING";
								const verdictClass =
									verdictStyles[verdict] ?? "text-muted-foreground border-border/60";

								return (
									<tr key={submission.id} className="group hover:bg-muted/30 transition-colors">
										<td className="px-3 py-3 font-mono text-xs text-muted-foreground/60">
											#{submission.id}
										</td>
										<td className="px-3 py-3 text-sm text-foreground">
											{submission.username ?? (submission.user_id ? `User #${submission.user_id}` : "—")}
										</td>
										<td className="px-3 py-3">
											<Link
												href={`/problems/${submission.problem_id}`}
												className="text-sm text-foreground hover:text-primary transition-colors"
											>
												{submission.problem_title ?? `Problem ${submission.problem_id}`}
											</Link>
										</td>
										<td className="px-3 py-3">
											<span
												className={`inline-flex border px-2.5 py-0.5 text-[10px] font-mono tracking-widest ${verdictClass}`}
											>
												{verdict}
											</span>
										</td>
										<td className="px-3 py-3 font-mono text-xs text-muted-foreground">
											{submission.score ?? "—"}
										</td>
										<td className="px-3 py-3 font-mono text-xs text-muted-foreground">
											{formatTests(submission.tests_passed, submission.tests_total)}
										</td>
										<td className="px-3 py-3 font-mono text-xs text-muted-foreground">
											{formatCpuTime(submission.cpu_time)}
										</td>
										<td className="px-3 py-3 font-mono text-xs text-muted-foreground">
											{formatMemory(submission.memory)}
										</td>
										<td className="px-3 py-3 font-mono text-xs text-muted-foreground">
											{submission.language?.toUpperCase?.() ?? "—"}
										</td>
										<td className="px-3 py-3 font-mono text-xs text-muted-foreground">
											{formatDate(submission.created_at)}
										</td>
										<td className="px-3 py-3">
											<Link
												href={`/submissions/${submission.id}`}
												className="border border-border/60 px-2.5 py-1 text-[10px] font-mono tracking-widest text-muted-foreground transition hover:border-primary/60 hover:text-primary"
											>
												VIEW
											</Link>
										</td>
									</tr>
								);
							})}
						</tbody>
					</table>
				</div>
			)}
		</section>
	);
}
