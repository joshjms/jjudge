"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useMemo, useState } from "react";

import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";

type Submission = {
	id: number;
	problem_id: number;
	user_id?: number;
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

type Problem = {
	id: number;
	title?: string;
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

const decodeUserIdFromToken = (token?: string | null) => {
	if (!token) return null;
	const [, payload] = token.split(".");
	if (!payload) return null;

	try {
		const normalized = payload.replace(/-/g, "+").replace(/_/g, "/");
		const padded = normalized.padEnd(Math.ceil(normalized.length / 4) * 4, "=");
		const decoded = atob(padded);
		const parsed = JSON.parse(decoded);
		const rawId = parsed?.user_id ?? parsed?.sub ?? parsed?.id;

		if (typeof rawId === "number") return rawId;
		if (typeof rawId === "string") {
			const numeric = Number(rawId);
			return Number.isFinite(numeric) ? numeric : null;
		}
		return null;
	} catch {
		return null;
	}
};

export default function MyProblemSubmissionsPage() {
	const params = useParams<{ id: string }>();
	const auth = useAuth();
	const userId = useMemo(() => decodeUserIdFromToken(auth.token), [auth.token]);
	const [problem, setProblem] = useState<Problem | null>(null);
	const [submissions, setSubmissions] = useState<Submission[] | null>(null);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);

	useEffect(() => {
		const load = async () => {
			const problemId = params?.id;
			if (!problemId) return;

			if (!auth.token || !userId) {
				setError("Please sign in to view your submissions.");
				setLoading(false);
				return;
			}

			try {
				const [problemResponse, submissionsResponse] = await Promise.all([
					api.get<Problem>(`/problems/${problemId}`, { cache: "no-store" }),
					api.get<Submission[]>("/submissions", {
						query: { problem_id: problemId, user_id: userId },
						headers: { Authorization: `Bearer ${auth.token}` },
						cache: "no-store",
					}),
				]);

				setProblem(problemResponse ?? null);
				setSubmissions(submissionsResponse ?? []);
				setError(null);
			} catch {
				setError("Failed to load your submissions for this problem.");
			} finally {
				setLoading(false);
			}
		};

		load();
	}, [auth.token, params?.id, userId]);

	const sortedSubmissions =
		submissions?.slice().sort((a, b) => {
			const timeA = a.created_at ? new Date(a.created_at).getTime() : 0;
			const timeB = b.created_at ? new Date(b.created_at).getTime() : 0;
			return timeB - timeA;
		}) ?? [];

	const headingTitle = problem
		? `${problem.title ?? "Untitled problem"} · Your submissions`
		: "Your submissions";

	return (
		<div className="mx-auto flex w-full max-w-5xl flex-col gap-8 px-4 py-12 sm:px-6">
			<div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
				<div className="space-y-2">
					<p className="text-xs font-semibold uppercase tracking-[0.25em] text-primary">
						Problem {params?.id}
					</p>
					<h1 className="text-3xl font-bold leading-tight sm:text-4xl">{headingTitle}</h1>
					<p className="text-sm text-muted-foreground">Only your submissions for this problem.</p>
				</div>
				<div className="flex flex-wrap items-center gap-2">
					<Link
						href={`/problems/${params?.id}`}
						className="rounded-md border border-border/70 px-3 py-2 text-sm font-semibold text-foreground transition hover:border-primary/60 hover:bg-muted/60"
					>
						View problem
					</Link>
					<Link
						href={`/problems/${params?.id}/submissions`}
						className="rounded-md border border-border/70 px-3 py-2 text-sm font-semibold text-foreground transition hover:border-primary/60 hover:bg-muted/60"
					>
						All submissions
					</Link>
				</div>
			</div>

			{loading ? (
				<div className="rounded-xl border border-border/70 bg-card/70 px-6 py-10 text-center text-sm text-muted-foreground">
					Loading your submissions...
				</div>
			) : error ? (
				<div className="rounded-xl border border-destructive/50 bg-destructive/10 px-6 py-10 text-center text-sm text-destructive">
					{error}
				</div>
			) : (
				<div className="overflow-hidden rounded-xl border border-border/70 bg-card/70">
					{sortedSubmissions.length > 0 ? (
						<div className="overflow-x-auto">
							<table className="min-w-full divide-y divide-border/70 text-sm">
								<thead className="bg-muted/70 text-xs uppercase tracking-wide text-muted-foreground">
									<tr>
										<th className="px-4 py-3 text-left font-semibold">ID</th>
										<th className="px-4 py-3 text-left font-semibold">Verdict</th>
										<th className="px-4 py-3 text-left font-semibold">Score</th>
										<th className="px-4 py-3 text-left font-semibold">Tests</th>
										<th className="px-4 py-3 text-left font-semibold">Time</th>
										<th className="px-4 py-3 text-left font-semibold">Memory</th>
										<th className="px-4 py-3 text-left font-semibold">Language</th>
										<th className="px-4 py-3 text-left font-semibold">Submitted at</th>
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
												<td className="px-4 py-3">
													<span
														className={`inline-flex rounded-full border px-3 py-1 text-xs font-semibold uppercase ${verdictClass}`}
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
											</tr>
										);
									})}
								</tbody>
							</table>
						</div>
					) : (
						<div className="px-6 py-10 text-center text-sm text-muted-foreground">
							No submissions yet for this problem.
						</div>
					)}
				</div>
			)}
		</div>
	);
}
