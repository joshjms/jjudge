"use client";

import { useCallback, useEffect, useRef, useState } from "react";

import { api } from "@/lib/api";

type TestcaseResult = {
	testcase_id: number;
	verdict: string;
	cpu_time: number;
	memory: number;
	input?: string;
	expected_output?: string;
	actual_output?: string;
	error_message?: string;
};

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
	testcase_results?: TestcaseResult[];
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
	if (value === undefined || value === null) return "\u2014";
	return `${value} ms`;
};

const formatMemory = (value?: number) => {
	if (value === undefined || value === null) return "\u2014";
	const mb = value / (1024 * 1024);
	return `${mb.toFixed(2)} MB`;
};

const formatTests = (passed?: number, total?: number) => {
	if (passed === undefined || total === undefined) return "\u2014";
	return `${passed}/${total}`;
};

const formatDate = (value?: string) => {
	if (!value) return "\u2014";
	return new Intl.DateTimeFormat(undefined, {
		year: "numeric",
		month: "short",
		day: "2-digit",
		hour: "2-digit",
		minute: "2-digit",
	}).format(new Date(value));
};

const isPending = (verdict?: string) => {
	const v = verdict?.toUpperCase?.();
	return v === "PENDING" || v === "JUDGING";
};

type SubmissionDetailProps = {
	initialSubmission: Submission;
};

export function SubmissionDetail({ initialSubmission }: SubmissionDetailProps) {
	const [submission, setSubmission] = useState(initialSubmission);
	const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

	const fetchSubmission = useCallback(async () => {
		try {
			const data = await api.get<Submission>(`/submissions/${initialSubmission.id}`, {
				cache: "no-store",
			});
			if (data) setSubmission(data);
		} catch {
			// silently ignore
		}
	}, [initialSubmission.id]);

	useEffect(() => {
		if (isPending(submission.verdict)) {
			pollRef.current = setInterval(fetchSubmission, 2000);
		} else if (pollRef.current) {
			clearInterval(pollRef.current);
			pollRef.current = null;
		}

		return () => {
			if (pollRef.current) {
				clearInterval(pollRef.current);
				pollRef.current = null;
			}
		};
	}, [submission.verdict, fetchSubmission]);

	const verdict = submission.verdict?.toUpperCase?.() ?? "PENDING";
	const verdictClass = verdictStyles[verdict] ?? "border-border/70 bg-muted/50 text-foreground";

	return (
		<>
			<section className="border border-border/70 bg-card/70">
				<div className="grid gap-0 divide-y divide-border/70 sm:grid-cols-2 sm:divide-y-0 sm:divide-x">
					<div className="p-5">
						<p className="text-xs uppercase tracking-wide text-muted-foreground">User</p>
						<p className="text-lg font-semibold text-foreground">
							{submission.username ?? (submission.user_id ? `User #${submission.user_id}` : "\u2014")}
						</p>
					</div>
					<div className="p-5">
						<p className="text-xs uppercase tracking-wide text-muted-foreground">Verdict</p>
						<span className={`inline-flex border px-3 py-1 text-xs font-semibold uppercase ${verdictClass}`}>
							{verdict}
						</span>
						{isPending(verdict) && (
							<span className="ml-2 inline-block h-2 w-2 animate-pulse rounded-full bg-blue-500" />
						)}
						{submission.message && (
							<p className="mt-2 text-sm text-muted-foreground">{submission.message}</p>
						)}
					</div>
					<div className="p-5">
						<p className="text-xs uppercase tracking-wide text-muted-foreground">Score</p>
						<p className="text-lg font-semibold text-foreground">{submission.score ?? "\u2014"}</p>
					</div>
					<div className="p-5">
						<p className="text-xs uppercase tracking-wide text-muted-foreground">Tests</p>
						<p className="text-lg font-semibold text-foreground">
							{formatTests(submission.tests_passed, submission.tests_total)}
						</p>
					</div>
					<div className="p-5">
						<p className="text-xs uppercase tracking-wide text-muted-foreground">CPU time</p>
						<p className="text-lg font-semibold text-foreground">
							{formatCpuTime(submission.cpu_time)}
						</p>
					</div>
					<div className="p-5">
						<p className="text-xs uppercase tracking-wide text-muted-foreground">Memory</p>
						<p className="text-lg font-semibold text-foreground">
							{formatMemory(submission.memory)}
						</p>
					</div>
					<div className="p-5">
						<p className="text-xs uppercase tracking-wide text-muted-foreground">Language</p>
						<p className="text-lg font-semibold text-foreground">
							{submission.language?.toUpperCase?.() ?? "\u2014"}
						</p>
					</div>
					<div className="p-5">
						<p className="text-xs uppercase tracking-wide text-muted-foreground">Submitted at</p>
						<p className="text-lg font-semibold text-foreground">
							{formatDate(submission.created_at)}
						</p>
					</div>
				</div>
			</section>

			{submission.testcase_results && submission.testcase_results.length > 0 && (
				<section className="border border-border/70 bg-card/70">
					<div className="border-b border-border/70 px-6 py-4">
						<h2 className="text-xl font-semibold">Test case results</h2>
					</div>
					<div className="overflow-x-auto">
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
								{submission.testcase_results.map((tc, idx) => {
									const tcVerdict = tc.verdict?.toUpperCase?.() ?? "\u2014";
									const tcVerdictClass = verdictStyles[tcVerdict] ?? "border-border/70 bg-muted/50 text-foreground";
									return (
										<tr key={tc.testcase_id ?? idx} className="hover:bg-muted/40">
											<td className="px-4 py-3 font-semibold text-muted-foreground">{idx + 1}</td>
											<td className="px-4 py-3">
												<span className={`inline-flex border px-3 py-1 text-xs font-semibold uppercase ${tcVerdictClass}`}>
													{tcVerdict}
												</span>
												{tc.error_message && (
													<p className="mt-1 text-[11px] text-muted-foreground">{tc.error_message}</p>
												)}
											</td>
											<td className="px-4 py-3 text-muted-foreground">{formatCpuTime(tc.cpu_time)}</td>
											<td className="px-4 py-3 text-muted-foreground">{formatMemory(tc.memory)}</td>
										</tr>
									);
								})}
							</tbody>
						</table>
					</div>

					{submission.testcase_results.some((tc) => tc.input || tc.expected_output || tc.actual_output) && (
						<div className="border-t border-border/70">
							<div className="border-b border-border/70 px-6 py-3">
								<h3 className="text-sm font-semibold">Test case details</h3>
							</div>
							<div className="divide-y divide-border/70">
								{submission.testcase_results
									.filter((tc) => tc.input || tc.expected_output || tc.actual_output)
									.map((tc, idx) => {
										const tcVerdict = tc.verdict?.toUpperCase?.() ?? "\u2014";
										const tcVerdictClass = verdictStyles[tcVerdict] ?? "border-border/70 bg-muted/50 text-foreground";
										return (
											<div key={tc.testcase_id ?? idx} className="px-6 py-4">
												<div className="mb-3 flex items-center gap-3">
													<span className="text-sm font-semibold text-muted-foreground">
														Test {idx + 1}
													</span>
													<span className={`inline-flex border px-2 py-0.5 text-[11px] font-semibold uppercase ${tcVerdictClass}`}>
														{tcVerdict}
													</span>
												</div>
												<div className="grid gap-3 sm:grid-cols-3">
													{tc.input !== undefined && (
														<div>
															<p className="mb-1 text-xs uppercase tracking-wide text-muted-foreground">Input</p>
															<pre className="overflow-x-auto border border-border/70 bg-muted/50 px-3 py-2 text-xs">{tc.input}</pre>
														</div>
													)}
													{tc.expected_output !== undefined && (
														<div>
															<p className="mb-1 text-xs uppercase tracking-wide text-muted-foreground">Expected</p>
															<pre className="overflow-x-auto border border-border/70 bg-muted/50 px-3 py-2 text-xs">{tc.expected_output}</pre>
														</div>
													)}
													{tc.actual_output !== undefined && (
														<div>
															<p className="mb-1 text-xs uppercase tracking-wide text-muted-foreground">Output</p>
															<pre className="overflow-x-auto border border-border/70 bg-muted/50 px-3 py-2 text-xs">{tc.actual_output}</pre>
														</div>
													)}
												</div>
											</div>
										);
									})}
							</div>
						</div>
					)}
				</section>
			)}
		</>
	);
}
