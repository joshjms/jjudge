"use client";

import { useEffect, useMemo, useState } from "react";

import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";

type Problem = {
	id: number;
	title?: string;
	difficulty?: number;
	visibility?: string;
	approval_status: string;
	creator_id?: number;
};

type Contest = {
	id: number;
	title: string;
	approval_status: string;
	owner_id?: number;
	scoring_type?: string;
	visibility?: string;
	start_time?: string;
	end_time?: string;
};

function ApprovalBadge({ status }: { status: string }) {
	const colors: Record<string, string> = {
		pending: "border-amber-500/50 bg-amber-500/10 text-amber-700",
		approved: "border-emerald-500/50 bg-emerald-500/10 text-emerald-700",
		rejected: "border-destructive/50 bg-destructive/10 text-destructive",
	};
	const cls = colors[status] ?? "border-border/70 bg-muted/40 text-muted-foreground";
	return (
		<span className={`border px-2 py-0.5 text-xs font-semibold uppercase tracking-wide ${cls}`}>
			{status}
		</span>
	);
}

export default function AdminApprovalsPage() {
	const auth = useAuth();
	const hasToken = Boolean(auth.token);

	const authHeaders = useMemo(
		() => (auth.token ? { Authorization: `Bearer ${auth.token}` } : undefined),
		[auth.token],
	);

	const [problems, setProblems] = useState<Problem[]>([]);
	const [contests, setContests] = useState<Contest[]>([]);
	const [loading, setLoading] = useState(false);
	const [error, setError] = useState<string | null>(null);
	const [status, setStatus] = useState<string | null>(null);

	const loadPending = async () => {
		if (!hasToken) return;
		setLoading(true);
		setError(null);
		try {
			const [problemsData, contestsData] = await Promise.all([
				api.get<{ items: Problem[] }>("/admin/approvals/problems", { headers: authHeaders }),
				api.get<{ items: Contest[] }>("/admin/approvals/contests", { headers: authHeaders }),
			]);
			setProblems(problemsData.items ?? []);
			setContests(contestsData.items ?? []);
		} catch {
			setError("Failed to load pending approvals.");
		} finally {
			setLoading(false);
		}
	};

	useEffect(() => {
		void loadPending();
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, [hasToken]);

	const approveProblem = async (id: number) => {
		if (!authHeaders) return;
		setStatus(null);
		try {
			await api.post(`/admin/approvals/problems/${id}/approve`, {}, { headers: authHeaders });
			setStatus(`Problem #${id} approved.`);
			await loadPending();
		} catch {
			setError("Failed to approve problem.");
		}
	};

	const rejectProblem = async (id: number) => {
		if (!authHeaders) return;
		setStatus(null);
		try {
			await api.post(`/admin/approvals/problems/${id}/reject`, {}, { headers: authHeaders });
			setStatus(`Problem #${id} rejected.`);
			await loadPending();
		} catch {
			setError("Failed to reject problem.");
		}
	};

	const approveContest = async (id: number) => {
		if (!authHeaders) return;
		setStatus(null);
		try {
			await api.post(`/admin/approvals/contests/${id}/approve`, {}, { headers: authHeaders });
			setStatus(`Contest #${id} approved.`);
			await loadPending();
		} catch {
			setError("Failed to approve contest.");
		}
	};

	const rejectContest = async (id: number) => {
		if (!authHeaders) return;
		setStatus(null);
		try {
			await api.post(`/admin/approvals/contests/${id}/reject`, {}, { headers: authHeaders });
			setStatus(`Contest #${id} rejected.`);
			await loadPending();
		} catch {
			setError("Failed to reject contest.");
		}
	};

	if (!hasToken) {
		return (
			<div className="px-4 py-12">
				<p className="text-sm text-muted-foreground">Please log in as admin to manage approvals.</p>
			</div>
		);
	}

	return (
		<div className="flex flex-col gap-10">
			<div className="space-y-2">
				<p className="text-xs font-semibold uppercase tracking-[0.3em] text-primary">Admin</p>
				<h1 className="text-3xl font-bold leading-tight">Approvals</h1>
				<p className="text-sm text-muted-foreground">
					Review and approve or reject pending problems and contests submitted by managers.
				</p>
			</div>

			{error && (
				<p className="border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive">
					{error}
				</p>
			)}
			{status && (
				<p className="border border-emerald-500/50 bg-emerald-500/10 px-3 py-2 text-sm text-emerald-700">
					{status}
				</p>
			)}

			{/* Pending Problems */}
			<div className="border border-border/70">
				<div className="border-b border-border/70 bg-muted/60 px-6 py-4">
					<h2 className="font-semibold">Pending Problems</h2>
				</div>
				{loading ? (
					<div className="px-6 py-10 text-sm text-muted-foreground">Loading...</div>
				) : problems.length === 0 ? (
					<div className="px-6 py-10 text-sm text-muted-foreground">No pending problems.</div>
				) : (
					<div className="divide-y divide-border/70">
						{problems.map((problem) => (
							<div key={problem.id} className="flex flex-wrap items-center gap-4 px-6 py-4">
								<div className="min-w-0 flex-1">
									<div className="flex items-center gap-2">
										<p className="font-semibold">
											#{problem.id} — {problem.title ?? "Untitled"}
										</p>
										<ApprovalBadge status={problem.approval_status} />
									</div>
									<p className="text-xs text-muted-foreground">
										Difficulty: {problem.difficulty ?? "—"} · Visibility:{" "}
										{problem.visibility ?? "public"}
										{problem.creator_id ? ` · Creator ID: ${problem.creator_id}` : ""}
									</p>
								</div>
								<div className="flex gap-2">
									<button
										onClick={() => approveProblem(problem.id)}
										className="border border-emerald-500/50 px-3 py-1.5 text-xs font-semibold text-emerald-700 hover:bg-emerald-500/10"
									>
										Approve
									</button>
									<button
										onClick={() => rejectProblem(problem.id)}
										className="border border-destructive/50 px-3 py-1.5 text-xs font-semibold text-destructive hover:bg-destructive/10"
									>
										Reject
									</button>
								</div>
							</div>
						))}
					</div>
				)}
			</div>

			{/* Pending Contests */}
			<div className="border border-border/70">
				<div className="border-b border-border/70 bg-muted/60 px-6 py-4">
					<h2 className="font-semibold">Pending Contests</h2>
				</div>
				{loading ? (
					<div className="px-6 py-10 text-sm text-muted-foreground">Loading...</div>
				) : contests.length === 0 ? (
					<div className="px-6 py-10 text-sm text-muted-foreground">No pending contests.</div>
				) : (
					<div className="divide-y divide-border/70">
						{contests.map((contest) => (
							<div key={contest.id} className="flex flex-wrap items-center gap-4 px-6 py-4">
								<div className="min-w-0 flex-1">
									<div className="flex items-center gap-2">
										<p className="font-semibold">
											#{contest.id} — {contest.title}
										</p>
										<ApprovalBadge status={contest.approval_status} />
									</div>
									<p className="text-xs text-muted-foreground">
										{contest.scoring_type?.toUpperCase() ?? "—"} · {contest.visibility ?? "public"}
										{contest.start_time
											? ` · ${new Date(contest.start_time).toLocaleDateString()}`
											: ""}
										{contest.owner_id ? ` · Owner ID: ${contest.owner_id}` : ""}
									</p>
								</div>
								<div className="flex gap-2">
									<button
										onClick={() => approveContest(contest.id)}
										className="border border-emerald-500/50 px-3 py-1.5 text-xs font-semibold text-emerald-700 hover:bg-emerald-500/10"
									>
										Approve
									</button>
									<button
										onClick={() => rejectContest(contest.id)}
										className="border border-destructive/50 px-3 py-1.5 text-xs font-semibold text-destructive hover:bg-destructive/10"
									>
										Reject
									</button>
								</div>
							</div>
						))}
					</div>
				)}
			</div>
		</div>
	);
}
