import Link from "next/link";
import { notFound } from "next/navigation";

import { api } from "@/lib/api";

type TestcaseResult = {
	submission_id?: number;
	testcase_id?: number;
	verdict?: string;
	cpu_time?: number;
	memory?: number;
	input?: string;
	expected_output?: string;
	actual_output?: string;
	error_message?: string;
};

type ContestSubmission = {
	id: number;
	contest_id: number;
	problem_id: number;
	user_id?: number;
	username?: string;
	code?: string;
	language?: string;
	verdict?: string;
	score?: number;
	cpu_time?: number;
	memory?: number;
	message?: string;
	tests_passed?: number;
	tests_total?: number;
	submitted_at?: string;
	updated_at?: string;
	testcase_results?: TestcaseResult[];
};

export const dynamic = "force-dynamic";

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

async function fetchSubmission(
	contestId: string,
	submissionId: string,
): Promise<ContestSubmission | null> {
	try {
		return await api.get<ContestSubmission>(
			`/contests/${contestId}/submissions/${submissionId}`,
			{ cache: "no-store" },
		);
	} catch {
		return null;
	}
}

export default async function ContestSubmissionPage({
	params,
}: {
	params: Promise<{ id: string; submissionId: string }>;
}) {
	const { id, submissionId } = await params;
	const submission = await fetchSubmission(id, submissionId);

	if (!submission) notFound();

	const verdict = submission.verdict?.toUpperCase?.() ?? "PENDING";
	const verdictClass =
		verdictStyles[verdict] ?? "border-border/70 bg-muted/50 text-foreground";

	return (
		<div className="mx-auto flex w-full max-w-5xl flex-col gap-8 px-4 py-12 sm:px-6">
			<div className="space-y-2">
				<p className="text-xs font-semibold uppercase tracking-[0.25em] text-primary">
					Contest {id} · Submission
				</p>
				<h1 className="text-3xl font-bold leading-tight sm:text-4xl">
					Submission #{submission.id}
				</h1>
				<div className="flex flex-wrap gap-3">
					<Link
						href={`/contests/${id}/submissions`}
						className="text-sm text-muted-foreground underline hover:text-foreground"
					>
						← All submissions
					</Link>
					<Link
						href={`/contests/${id}/problems/${submission.problem_id}`}
						className="text-sm text-muted-foreground underline hover:text-foreground"
					>
						Problem {submission.problem_id}
					</Link>
				</div>
			</div>

			{/* Summary */}
			<div className="grid grid-cols-2 gap-4 border border-border/70 p-5 sm:grid-cols-4">
				<div>
					<p className="text-xs uppercase tracking-wide text-muted-foreground">Verdict</p>
					<span className={`mt-1 inline-flex border px-3 py-1 text-sm font-semibold uppercase ${verdictClass}`}>
						{verdict}
					</span>
				</div>
				<div>
					<p className="text-xs uppercase tracking-wide text-muted-foreground">Score</p>
					<p className="mt-1 font-semibold">{submission.score ?? "—"}</p>
				</div>
				<div>
					<p className="text-xs uppercase tracking-wide text-muted-foreground">Time</p>
					<p className="mt-1 font-semibold">{submission.cpu_time != null ? `${submission.cpu_time} ms` : "—"}</p>
				</div>
				<div>
					<p className="text-xs uppercase tracking-wide text-muted-foreground">Memory</p>
					<p className="mt-1 font-semibold">
						{submission.memory != null
							? `${(submission.memory / (1024 * 1024)).toFixed(2)} MB`
							: "—"}
					</p>
				</div>
				<div>
					<p className="text-xs uppercase tracking-wide text-muted-foreground">Language</p>
					<p className="mt-1 font-semibold">{submission.language?.toUpperCase() ?? "—"}</p>
				</div>
				<div>
					<p className="text-xs uppercase tracking-wide text-muted-foreground">Tests</p>
					<p className="mt-1 font-semibold">
						{submission.tests_passed != null && submission.tests_total != null
							? `${submission.tests_passed}/${submission.tests_total}`
							: "—"}
					</p>
				</div>
				<div>
					<p className="text-xs uppercase tracking-wide text-muted-foreground">User</p>
					<p className="mt-1 font-semibold">
						{submission.username ?? (submission.user_id ? `#${submission.user_id}` : "—")}
					</p>
				</div>
			</div>

			{submission.message && (
				<div className="border border-destructive/50 bg-destructive/10 px-4 py-3 text-sm text-destructive">
					{submission.message}
				</div>
			)}

			{/* Source code */}
			{submission.code && (
				<div>
					<h2 className="mb-3 text-lg font-semibold">Source code</h2>
					<pre className="overflow-x-auto border border-border/70 bg-muted/30 p-4 text-xs leading-relaxed">
						{submission.code}
					</pre>
				</div>
			)}

			{/* Testcase results */}
			{submission.testcase_results && submission.testcase_results.length > 0 && (
				<div>
					<h2 className="mb-3 text-lg font-semibold">Test results</h2>
					<div className="overflow-hidden border border-border/70">
						<table className="min-w-full divide-y divide-border/70 text-sm">
							<thead className="bg-muted/70 text-xs uppercase tracking-wide text-muted-foreground">
								<tr>
									<th className="px-4 py-3 text-left font-semibold">#</th>
									<th className="px-4 py-3 text-left font-semibold">Verdict</th>
									<th className="px-4 py-3 text-left font-semibold">Time</th>
									<th className="px-4 py-3 text-left font-semibold">Memory</th>
								</tr>
							</thead>
							<tbody className="divide-y divide-border/70">
								{submission.testcase_results.map((tr, i) => {
									const tcVerdict = tr.verdict?.toUpperCase?.() ?? "PENDING";
									const tcClass =
										verdictStyles[tcVerdict] ?? "border-border/70 bg-muted/50 text-foreground";
									return (
										<tr key={i} className="hover:bg-muted/40">
											<td className="px-4 py-3 text-muted-foreground">{i + 1}</td>
											<td className="px-4 py-3">
												<span className={`inline-flex border px-2 py-0.5 text-xs font-semibold uppercase ${tcClass}`}>
													{tcVerdict}
												</span>
											</td>
											<td className="px-4 py-3 text-muted-foreground">
												{tr.cpu_time != null ? `${tr.cpu_time} ms` : "—"}
											</td>
											<td className="px-4 py-3 text-muted-foreground">
												{tr.memory != null
													? `${(tr.memory / (1024 * 1024)).toFixed(2)} MB`
													: "—"}
											</td>
										</tr>
									);
								})}
							</tbody>
						</table>
					</div>
				</div>
			)}
		</div>
	);
}
