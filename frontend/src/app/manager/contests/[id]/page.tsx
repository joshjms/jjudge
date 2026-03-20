"use client";

import { useEffect, useMemo, useState } from "react";
import { useParams } from "next/navigation";

import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";

type Contest = {
	id: number;
	title: string;
	description?: string;
	start_time: string;
	end_time: string;
	scoring_type: string;
	visibility: string;
	approval_status: string;
	owner_id?: number;
	problems?: ContestProblem[];
};

type Problem = {
	id: number;
	title?: string;
	approval_status?: string;
};

type ContestProblem = {
	contest_id: number;
	problem_id: number;
	ordinal: number;
	max_points: number;
	problem?: Problem;
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

export default function ManagerContestDetailPage() {
	const params = useParams();
	const contestId = Number(params.id);
	const auth = useAuth();
	const hasToken = Boolean(auth.token);

	const authHeaders = useMemo(
		() => (auth.token ? { Authorization: `Bearer ${auth.token}` } : undefined),
		[auth.token],
	);

	const [contest, setContest] = useState<Contest | null>(null);
	const [contestProblems, setContestProblems] = useState<ContestProblem[]>([]);
	const [allProblems, setAllProblems] = useState<Problem[]>([]);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);
	const [status, setStatus] = useState<string | null>(null);

	// Edit form
	const [editing, setEditing] = useState(false);
	const [title, setTitle] = useState("");
	const [description, setDescription] = useState("");
	const [startTime, setStartTime] = useState("");
	const [endTime, setEndTime] = useState("");
	const [scoringType, setScoringType] = useState("icpc");
	const [visibility, setVisibility] = useState("public");

	// Add problem
	const [addProblemId, setAddProblemId] = useState("");
	const [addOrdinal, setAddOrdinal] = useState(0);
	const [addMaxPoints, setAddMaxPoints] = useState(100);
	const [problemStatus, setProblemStatus] = useState<string | null>(null);

	const loadContest = async () => {
		if (!hasToken || !contestId) return;
		setLoading(true);
		setError(null);
		try {
			const data = await api.get<Contest>(`/contests/${contestId}`, { headers: authHeaders });
			setContest(data);
			setTitle(data.title);
			setDescription(data.description ?? "");
			setStartTime(data.start_time.slice(0, 16));
			setEndTime(data.end_time.slice(0, 16));
			setScoringType(data.scoring_type);
			setVisibility(data.visibility);
		} catch {
			setError("Failed to load contest.");
		} finally {
			setLoading(false);
		}
	};

	const loadContestProblems = async () => {
		if (!authHeaders || !contestId) return;
		try {
			const data = await api.get<ContestProblem[]>(`/contests/${contestId}/problems`, {
				headers: authHeaders,
			});
			setContestProblems(data ?? []);
		} catch {
			setContestProblems([]);
		}
	};

	const loadAllProblems = async () => {
		if (!authHeaders) return;
		try {
			const data = await api.get<{ items: Problem[] }>("/manager/problems", {
				headers: authHeaders,
			});
			setAllProblems(data.items ?? []);
		} catch {
			// ignore
		}
	};

	useEffect(() => {
		if (hasToken && contestId) {
			void Promise.all([loadContest(), loadContestProblems(), loadAllProblems()]);
		}
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, [hasToken, contestId]);

	const handleUpdate = async (e: React.FormEvent) => {
		e.preventDefault();
		if (!authHeaders) return;
		setStatus(null);
		setError(null);

		try {
			await api.put(
				`/contests/${contestId}`,
				{
					title,
					description,
					start_time: new Date(startTime).toISOString(),
					end_time: new Date(endTime).toISOString(),
					scoring_type: scoringType,
					visibility,
				},
				{ headers: authHeaders },
			);
			setStatus("Contest updated.");
			setEditing(false);
			await loadContest();
		} catch {
			setError("Failed to update contest.");
		}
	};

	const handleAddProblem = async (e: React.FormEvent) => {
		e.preventDefault();
		if (!authHeaders) return;
		setProblemStatus(null);
		try {
			await api.post(
				`/contests/${contestId}/problems`,
				{
					problem_id: Number(addProblemId),
					ordinal: addOrdinal,
					max_points: addMaxPoints,
				},
				{ headers: authHeaders },
			);
			setProblemStatus("Problem added.");
			setAddProblemId("");
			await loadContestProblems();
		} catch {
			setProblemStatus("Failed to add problem.");
		}
	};

	const handleRemoveProblem = async (problemId: number) => {
		if (!authHeaders) return;
		try {
			await api.delete(`/contests/${contestId}/problems/${problemId}`, {
				headers: authHeaders,
			});
			await loadContestProblems();
		} catch {
			setProblemStatus("Failed to remove problem.");
		}
	};

	if (!hasToken) {
		return (
			<div className="px-4 py-12">
				<p className="text-sm text-muted-foreground">Please log in to manage this contest.</p>
			</div>
		);
	}

	if (loading) {
		return (
			<div className="px-4 py-12 text-sm text-muted-foreground">Loading...</div>
		);
	}

	if (error || !contest) {
		return (
			<div className="px-4 py-12">
				<p className="border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive">
					{error ?? "Contest not found."}
				</p>
			</div>
		);
	}

	return (
		<div className="flex flex-col gap-8">
			{/* Header */}
			<div className="space-y-2">
				<div className="flex items-center gap-3">
					<p className="text-xs font-semibold uppercase tracking-[0.3em] text-primary">Manager</p>
				</div>
				<div className="flex flex-wrap items-center gap-3">
					<h1 className="text-3xl font-bold leading-tight">{contest.title}</h1>
					<ApprovalBadge status={contest.approval_status} />
				</div>
				{contest.approval_status === "pending" && (
					<p className="text-sm text-amber-700 border border-amber-500/50 bg-amber-500/10 px-3 py-2">
						This contest is <strong>pending admin approval</strong> and is not yet visible to the public.
						You can still prepare it by adding problems and setting times.
					</p>
				)}
				{contest.approval_status === "rejected" && (
					<p className="text-sm text-destructive border border-destructive/50 bg-destructive/10 px-3 py-2">
						This contest was <strong>rejected</strong> by an admin. Please contact an admin for more information.
					</p>
				)}
			</div>

			{status && (
				<p className="border border-emerald-500/50 bg-emerald-500/10 px-3 py-2 text-sm text-emerald-700">
					{status}
				</p>
			)}

			{/* Contest Details / Edit Form */}
			<div className="border border-border/70 p-6">
				<div className="mb-4 flex items-center justify-between">
					<h2 className="text-lg font-semibold">Contest Details</h2>
					{!editing && (
						<button
							onClick={() => setEditing(true)}
							className="border border-border/70 px-3 py-1.5 text-xs font-semibold hover:bg-muted/50"
						>
							Edit
						</button>
					)}
				</div>

				{!editing ? (
					<dl className="grid grid-cols-1 gap-3 text-sm sm:grid-cols-2">
						<div>
							<dt className="font-medium text-muted-foreground">Title</dt>
							<dd>{contest.title}</dd>
						</div>
						<div>
							<dt className="font-medium text-muted-foreground">Scoring Type</dt>
							<dd>{contest.scoring_type.toUpperCase()}</dd>
						</div>
						<div>
							<dt className="font-medium text-muted-foreground">Visibility</dt>
							<dd>{contest.visibility}</dd>
						</div>
						<div>
							<dt className="font-medium text-muted-foreground">Start Time</dt>
							<dd>{new Date(contest.start_time).toLocaleString()}</dd>
						</div>
						<div>
							<dt className="font-medium text-muted-foreground">End Time</dt>
							<dd>{new Date(contest.end_time).toLocaleString()}</dd>
						</div>
						{contest.description && (
							<div className="sm:col-span-2">
								<dt className="font-medium text-muted-foreground">Description</dt>
								<dd className="whitespace-pre-line">{contest.description}</dd>
							</div>
						)}
					</dl>
				) : (
					<form onSubmit={handleUpdate} className="flex flex-col gap-4">
						<div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
							<label className="flex flex-col gap-1 text-sm">
								<span className="font-medium">Title</span>
								<input
									className="border border-border/60 bg-background px-3 py-2"
									value={title}
									onChange={(e) => setTitle(e.target.value)}
									required
								/>
							</label>
							<label className="flex flex-col gap-1 text-sm">
								<span className="font-medium">Visibility</span>
								<select
									className="border border-border/60 bg-background px-3 py-2"
									value={visibility}
									onChange={(e) => setVisibility(e.target.value)}
								>
									<option value="public">Public</option>
									<option value="private">Private</option>
								</select>
							</label>
							<label className="flex flex-col gap-1 text-sm">
								<span className="font-medium">Start Time</span>
								<input
									type="datetime-local"
									className="border border-border/60 bg-background px-3 py-2"
									value={startTime}
									onChange={(e) => setStartTime(e.target.value)}
									required
								/>
							</label>
							<label className="flex flex-col gap-1 text-sm">
								<span className="font-medium">End Time</span>
								<input
									type="datetime-local"
									className="border border-border/60 bg-background px-3 py-2"
									value={endTime}
									onChange={(e) => setEndTime(e.target.value)}
									required
								/>
							</label>
							<label className="flex flex-col gap-1 text-sm">
								<span className="font-medium">Scoring Type</span>
								<select
									className="border border-border/60 bg-background px-3 py-2"
									value={scoringType}
									onChange={(e) => setScoringType(e.target.value)}
								>
									<option value="icpc">ICPC (binary + penalty)</option>
									<option value="ioi">IOI (partial score)</option>
								</select>
							</label>
						</div>
						<label className="flex flex-col gap-1 text-sm">
							<span className="font-medium">Description</span>
							<textarea
								className="border border-border/60 bg-background px-3 py-2 text-sm"
								rows={3}
								value={description}
								onChange={(e) => setDescription(e.target.value)}
							/>
						</label>

						{error && (
							<p className="border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive">
								{error}
							</p>
						)}

						<div className="flex gap-3">
							<button
								type="submit"
								className="bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground hover:bg-primary/90"
							>
								Save Changes
							</button>
							<button
								type="button"
								onClick={() => setEditing(false)}
								className="border border-border/70 px-4 py-2 text-sm font-semibold hover:bg-muted/50"
							>
								Cancel
							</button>
						</div>
					</form>
				)}
			</div>

			{/* Problem Management */}
			<div className="border border-border/70 p-6">
				<h2 className="mb-5 text-lg font-semibold">Problems</h2>

				{/* Add problem form */}
				<form onSubmit={handleAddProblem} className="mb-6 flex flex-wrap items-end gap-3">
					<label className="flex flex-col gap-1 text-sm">
						<span className="font-medium">Problem</span>
						<select
							className="border border-border/60 bg-background px-3 py-2 text-sm"
							value={addProblemId}
							onChange={(e) => setAddProblemId(e.target.value)}
							required
						>
							<option value="">Select a problem…</option>
							{allProblems.map((p) => (
								<option key={p.id} value={p.id}>
									#{p.id} — {p.title ?? "Untitled"}
								</option>
							))}
						</select>
					</label>
					<label className="flex flex-col gap-1 text-sm">
						<span className="font-medium">Ordinal (0-based)</span>
						<input
							type="number"
							min={0}
							className="w-24 border border-border/60 bg-background px-3 py-2 text-sm"
							value={addOrdinal}
							onChange={(e) => setAddOrdinal(Number(e.target.value))}
						/>
					</label>
					<label className="flex flex-col gap-1 text-sm">
						<span className="font-medium">Max Points</span>
						<input
							type="number"
							min={0}
							className="w-24 border border-border/60 bg-background px-3 py-2 text-sm"
							value={addMaxPoints}
							onChange={(e) => setAddMaxPoints(Number(e.target.value))}
						/>
					</label>
					<button
						type="submit"
						className="bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground hover:bg-primary/90"
					>
						Add Problem
					</button>
				</form>

				{problemStatus && (
					<p className="mb-4 border border-emerald-500/50 bg-emerald-500/10 px-3 py-2 text-sm text-emerald-700">
						{problemStatus}
					</p>
				)}

				{contestProblems.length === 0 ? (
					<p className="text-sm text-muted-foreground">No problems in this contest yet.</p>
				) : (
					<div className="overflow-hidden border border-border/70">
						<table className="min-w-full divide-y divide-border/70 text-sm">
							<thead className="bg-muted/70 text-xs uppercase tracking-wide text-muted-foreground">
								<tr>
									<th className="px-4 py-3 text-left">Ordinal</th>
									<th className="px-4 py-3 text-left">Problem</th>
									<th className="px-4 py-3 text-left">Max Points</th>
									<th className="px-4 py-3 text-left">Actions</th>
								</tr>
							</thead>
							<tbody className="divide-y divide-border/70">
								{contestProblems
									.slice()
									.sort((a, b) => a.ordinal - b.ordinal)
									.map((cp) => (
										<tr key={cp.problem_id} className="hover:bg-muted/40">
											<td className="px-4 py-3 font-bold">{cp.ordinal}</td>
											<td className="px-4 py-3">
												{cp.problem?.title ?? `Problem #${cp.problem_id}`}
											</td>
											<td className="px-4 py-3 text-muted-foreground">{cp.max_points}</td>
											<td className="px-4 py-3">
												<button
													onClick={() => handleRemoveProblem(cp.problem_id)}
													className="border border-destructive/50 px-2 py-1 text-xs font-semibold text-destructive hover:bg-destructive/10"
												>
													Remove
												</button>
											</td>
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
