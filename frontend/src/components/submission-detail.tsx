"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { ChevronDown, ChevronRight } from "lucide-react";

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
	const [tcExpanded, setTcExpanded] = useState(false);
	const [expandedRows, setExpandedRows] = useState<Set<number>>(new Set());

	const toggleRow = (idx: number) => {
		setExpandedRows((prev) => {
			const next = new Set(prev);
			if (next.has(idx)) next.delete(idx);
			else next.add(idx);
			return next;
		});
	};

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
	const verdictClass = verdictStyles[verdict] ?? "text-muted-foreground border-border/60";

	return (
		<>
			<section className="border border-border/70 bg-card/70">
				<div className="grid gap-0 divide-y divide-border/70 sm:grid-cols-2 sm:divide-y-0 sm:divide-x">
					<div className="p-5">
						<p className="text-xs uppercase tracking-wide text-muted-foreground">User</p>
						<p className="text-lg font-semibold text-foreground">
							{submission.username ?? (submission.user_id ? `User #${submission.user_id}` : "—")}
						</p>
					</div>
					<div className="p-5">
						<p className="text-xs uppercase tracking-wide text-muted-foreground">Verdict</p>
						<span className={`inline-flex border px-3 py-1 text-xs font-mono tracking-widest uppercase ${verdictClass}`}>
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
						<p className="text-lg font-semibold text-foreground">{submission.score ?? "—"}</p>
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
							{submission.language?.toUpperCase?.() ?? "—"}
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
					{/* Section header — click to toggle all results */}
					<button
						type="button"
						onClick={() => setTcExpanded((v) => !v)}
						className="flex w-full items-center justify-between px-6 py-4 hover:bg-muted/30 transition-colors"
					>
						<div className="flex items-center gap-3">
							<span className="text-[10px] font-mono font-semibold uppercase tracking-widest text-muted-foreground">
								Testcase Results
							</span>
							<span className="text-[10px] font-mono text-muted-foreground/50 tracking-widest">
								{submission.testcase_results.length} CASES
							</span>
						</div>
						{tcExpanded ? (
							<ChevronDown className="h-3.5 w-3.5 text-muted-foreground" />
						) : (
							<ChevronRight className="h-3.5 w-3.5 text-muted-foreground" />
						)}
					</button>

					{tcExpanded && (
						<div className="border-t border-border/70 divide-y divide-border/40">
							{submission.testcase_results.map((tc, idx) => {
								const tcVerdict = tc.verdict?.toUpperCase?.() ?? "—";
								const tcVerdictClass = verdictStyles[tcVerdict] ?? "text-muted-foreground border-border/60";
								const hasDetails =
									tc.input !== undefined ||
									tc.expected_output !== undefined ||
									tc.actual_output !== undefined;
								const rowOpen = expandedRows.has(idx);

								return (
									<div key={tc.testcase_id ?? idx}>
										{/* Row summary */}
										<button
											type="button"
											disabled={!hasDetails}
											onClick={() => hasDetails && toggleRow(idx)}
											className={`flex w-full items-center gap-4 px-6 py-3 text-left transition-colors ${hasDetails ? "hover:bg-muted/30 cursor-pointer" : "cursor-default"}`}
										>
											<span className="w-7 shrink-0 text-[10px] font-mono text-muted-foreground/40 text-right">
												{String(idx + 1).padStart(2, "0")}
											</span>
											<span className={`shrink-0 border px-2.5 py-0.5 text-[10px] font-mono tracking-widest ${tcVerdictClass}`}>
												{tcVerdict}
											</span>
											<span className="font-mono text-xs text-muted-foreground">{formatCpuTime(tc.cpu_time)}</span>
											<span className="font-mono text-xs text-muted-foreground">{formatMemory(tc.memory)}</span>
											{tc.error_message && (
												<span className="text-[11px] text-muted-foreground truncate">{tc.error_message}</span>
											)}
											{hasDetails && (
												<span className="ml-auto shrink-0">
													{rowOpen ? (
														<ChevronDown className="h-3 w-3 text-muted-foreground/40" />
													) : (
														<ChevronRight className="h-3 w-3 text-muted-foreground/40" />
													)}
												</span>
											)}
										</button>

										{/* Expanded I/O details */}
										{hasDetails && rowOpen && (
											<div className="border-t border-border/40 bg-muted/20 px-6 py-4 grid gap-3 sm:grid-cols-3">
												{tc.input !== undefined && (
													<div>
														<p className="mb-1.5 text-[10px] font-mono uppercase tracking-widest text-muted-foreground/60">Input</p>
														<pre className="overflow-x-auto border border-border/60 bg-background px-3 py-2 text-xs font-mono">{tc.input}</pre>
													</div>
												)}
												{tc.expected_output !== undefined && (
													<div>
														<p className="mb-1.5 text-[10px] font-mono uppercase tracking-widest text-muted-foreground/60">Expected</p>
														<pre className="overflow-x-auto border border-border/60 bg-background px-3 py-2 text-xs font-mono">{tc.expected_output}</pre>
													</div>
												)}
												{tc.actual_output !== undefined && (
													<div>
														<p className="mb-1.5 text-[10px] font-mono uppercase tracking-widest text-muted-foreground/60">Output</p>
														<pre className="overflow-x-auto border border-border/60 bg-background px-3 py-2 text-xs font-mono">{tc.actual_output}</pre>
													</div>
												)}
											</div>
										)}
									</div>
								);
							})}
						</div>
					)}
				</section>
			)}
		</>
	);
}
