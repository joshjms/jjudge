"use client";

import { useEffect, useMemo, useState } from "react";

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
};

type Problem = {
	id: number;
	title?: string;
};

type ContestProblem = {
	contest_id: number;
	problem_id: number;
	ordinal: number;
	max_points: number;
	problem?: Problem;
};

type ContestListResponse = {
	items: Contest[];
};

type ProblemListResponse = {
	items: Problem[];
};

export default function AdminContestsPage() {
	const auth = useAuth();
	const hasToken = Boolean(auth.token);

	const authHeaders = useMemo(
		() => (auth.token ? { Authorization: `Bearer ${auth.token}` } : undefined),
		[auth.token],
	);

	// Contest list
	const [contests, setContests] = useState<Contest[]>([]);
	const [loading, setLoading] = useState(false);
	const [error, setError] = useState<string | null>(null);
	const [status, setStatus] = useState<string | null>(null);

	// Create/edit form
	const [editingId, setEditingId] = useState<number | null>(null);
	const [title, setTitle] = useState("New Contest");
	const [description, setDescription] = useState("");
	const [startTime, setStartTime] = useState("");
	const [endTime, setEndTime] = useState("");
	const [scoringType, setScoringType] = useState("icpc");
	const [visibility, setVisibility] = useState("public");

	// Problem management panel
	const [managingContestId, setManagingContestId] = useState<number | null>(null);
	const [contestProblems, setContestProblems] = useState<ContestProblem[]>([]);
	const [allProblems, setAllProblems] = useState<Problem[]>([]);
	const [addProblemId, setAddProblemId] = useState("");
	const [addOrdinal, setAddOrdinal] = useState(0);
	const [addMaxPoints, setAddMaxPoints] = useState(100);
	const [problemStatus, setProblemStatus] = useState<string | null>(null);

	const loadContests = async () => {
		if (!hasToken) return;
		setLoading(true);
		setError(null);
		try {
			const data = await api.get<ContestListResponse>("/contests?limit=100", {
				headers: authHeaders,
			});
			setContests(data.items ?? []);
		} catch {
			setError("Failed to load contests.");
		} finally {
			setLoading(false);
		}
	};

	const loadAllProblems = async () => {
		if (!hasToken) return;
		try {
			const data = await api.get<ProblemListResponse>("/problems?limit=100", {
				headers: authHeaders,
			});
			setAllProblems(data.items ?? []);
		} catch {
			// ignore
		}
	};

	useEffect(() => {
		void loadContests();
		void loadAllProblems();
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, [hasToken]);

	const resetForm = () => {
		setEditingId(null);
		setTitle("New Contest");
		setDescription("");
		setStartTime("");
		setEndTime("");
		setScoringType("icpc");
		setVisibility("public");
		setStatus(null);
		setError(null);
	};

	const handleEdit = (contest: Contest) => {
		setEditingId(contest.id);
		setTitle(contest.title);
		setDescription(contest.description ?? "");
		setStartTime(contest.start_time.slice(0, 16));
		setEndTime(contest.end_time.slice(0, 16));
		setScoringType(contest.scoring_type);
		setVisibility(contest.visibility);
		setStatus(null);
		setError(null);
	};

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		if (!authHeaders) return;

		setStatus(null);
		setError(null);

		const payload = {
			title,
			description,
			start_time: new Date(startTime).toISOString(),
			end_time: new Date(endTime).toISOString(),
			scoring_type: scoringType,
			visibility,
		};

		try {
			if (editingId !== null) {
				await api.put(`/contests/${editingId}`, payload, { headers: authHeaders });
				setStatus("Contest updated.");
			} else {
				await api.post("/contests", payload, { headers: authHeaders });
				setStatus("Contest created.");
			}
			resetForm();
			await loadContests();
		} catch {
			setError("Failed to save contest.");
		}
	};

	const handleDelete = async (id: number) => {
		if (!authHeaders) return;
		if (!confirm("Delete this contest?")) return;
		try {
			await api.delete(`/contests/${id}`, { headers: authHeaders });
			setContests((prev) => prev.filter((c) => c.id !== id));
		} catch {
			setError("Failed to delete contest.");
		}
	};

	// Problem management
	const openProblemPanel = async (contestId: number) => {
		setManagingContestId(contestId);
		setProblemStatus(null);
		await loadContestProblems(contestId);
	};

	const loadContestProblems = async (contestId: number) => {
		if (!authHeaders) return;
		try {
			const data = await api.get<ContestProblem[]>(`/contests/${contestId}/problems`, {
				headers: authHeaders,
			});
			setContestProblems(data ?? []);
		} catch {
			setContestProblems([]);
		}
	};

	const handleAddProblem = async (e: React.FormEvent) => {
		e.preventDefault();
		if (!authHeaders || managingContestId === null) return;
		setProblemStatus(null);
		try {
			await api.post(
				`/contests/${managingContestId}/problems`,
				{
					problem_id: Number(addProblemId),
					ordinal: addOrdinal,
					max_points: addMaxPoints,
				},
				{ headers: authHeaders },
			);
			setProblemStatus("Problem added.");
			setAddProblemId("");
			await loadContestProblems(managingContestId);
		} catch {
			setProblemStatus("Failed to add problem.");
		}
	};

	const handleRemoveProblem = async (problemId: number) => {
		if (!authHeaders || managingContestId === null) return;
		try {
			await api.delete(`/contests/${managingContestId}/problems/${problemId}`, {
				headers: authHeaders,
			});
			await loadContestProblems(managingContestId);
		} catch {
			setProblemStatus("Failed to remove problem.");
		}
	};

	const handleRejudge = async (problemId: number) => {
		if (!authHeaders || managingContestId === null) return;
		if (!confirm("Rejudge all submissions for this problem?")) return;
		try {
			await api.post(
				`/contests/${managingContestId}/problems/${problemId}/rejudge`,
				{},
				{ headers: authHeaders },
			);
			setProblemStatus("Rejudge enqueued.");
		} catch {
			setProblemStatus("Failed to rejudge.");
		}
	};

	if (!hasToken) {
		return (
			<div className="mx-auto max-w-3xl px-4 py-12">
				<p className="text-sm text-muted-foreground">Please log in as admin to manage contests.</p>
			</div>
		);
	}

	return (
		<div className="mx-auto flex w-full max-w-6xl flex-col gap-10 px-4 py-12 sm:px-6">
			<div className="space-y-2">
				<p className="text-xs font-semibold uppercase tracking-[0.25em] text-primary">Admin</p>
				<h1 className="text-3xl font-bold leading-tight sm:text-4xl">Contest Management</h1>
			</div>

			{/* Create / Edit Form */}
			<div className="border border-border/70 p-6">
				<h2 className="mb-5 text-lg font-semibold">
					{editingId !== null ? `Edit Contest #${editingId}` : "Create Contest"}
				</h2>
				<form onSubmit={handleSubmit} className="flex flex-col gap-4">
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
					{status && (
						<p className="border border-emerald-500/50 bg-emerald-500/10 px-3 py-2 text-sm text-emerald-700">
							{status}
						</p>
					)}

					<div className="flex gap-3">
						<button
							type="submit"
							className="bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground hover:bg-primary/90"
						>
							{editingId !== null ? "Update" : "Create"}
						</button>
						{editingId !== null && (
							<button
								type="button"
								onClick={resetForm}
								className="border border-border/70 px-4 py-2 text-sm font-semibold hover:bg-muted/50"
							>
								Cancel
							</button>
						)}
					</div>
				</form>
			</div>

			{/* Contest List */}
			<div className="border border-border/70">
				<div className="border-b border-border/70 px-6 py-4">
					<h2 className="font-semibold">All Contests</h2>
				</div>
				{loading ? (
					<div className="px-6 py-10 text-sm text-muted-foreground">Loading...</div>
				) : contests.length === 0 ? (
					<div className="px-6 py-10 text-sm text-muted-foreground">No contests yet.</div>
				) : (
					<div className="divide-y divide-border/70">
						{contests.map((contest) => (
							<div key={contest.id} className="flex flex-wrap items-center gap-4 px-6 py-4">
								<div className="min-w-0 flex-1">
									<p className="font-semibold">
										#{contest.id} — {contest.title}
									</p>
									<p className="text-xs text-muted-foreground">
										{contest.scoring_type.toUpperCase()} · {contest.visibility} ·{" "}
										{new Date(contest.start_time).toLocaleDateString()} →{" "}
										{new Date(contest.end_time).toLocaleDateString()}
									</p>
								</div>
								<div className="flex flex-wrap gap-2">
									<button
										onClick={() => openProblemPanel(contest.id)}
										className="border border-border/70 px-3 py-1.5 text-xs font-semibold hover:bg-muted/50"
									>
										Problems
									</button>
									<button
										onClick={() => handleEdit(contest)}
										className="border border-border/70 px-3 py-1.5 text-xs font-semibold hover:bg-muted/50"
									>
										Edit
									</button>
									<button
										onClick={() => handleDelete(contest.id)}
										className="border border-destructive/50 px-3 py-1.5 text-xs font-semibold text-destructive hover:bg-destructive/10"
									>
										Delete
									</button>
								</div>
							</div>
						))}
					</div>
				)}
			</div>

			{/* Problem Management Panel */}
			{managingContestId !== null && (
				<div className="border border-border/70 p-6">
					<div className="mb-5 flex items-center justify-between">
						<h2 className="text-lg font-semibold">
							Problems — Contest #{managingContestId}
						</h2>
						<button
							onClick={() => setManagingContestId(null)}
							className="text-xs text-muted-foreground underline hover:text-foreground"
						>
							Close
						</button>
					</div>

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
								className="border border-border/60 bg-background px-3 py-2 text-sm w-24"
								value={addOrdinal}
								onChange={(e) => setAddOrdinal(Number(e.target.value))}
							/>
						</label>
						<label className="flex flex-col gap-1 text-sm">
							<span className="font-medium">Max Points</span>
							<input
								type="number"
								min={0}
								className="border border-border/60 bg-background px-3 py-2 text-sm w-24"
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
						<p className="text-sm text-muted-foreground">No problems in this contest.</p>
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
													<div className="flex gap-2">
														<button
															onClick={() => handleRejudge(cp.problem_id)}
															className="border border-border/70 px-2 py-1 text-xs font-semibold hover:bg-muted/50"
														>
															Rejudge
														</button>
														<button
															onClick={() => handleRemoveProblem(cp.problem_id)}
															className="border border-destructive/50 px-2 py-1 text-xs font-semibold text-destructive hover:bg-destructive/10"
														>
															Remove
														</button>
													</div>
												</td>
											</tr>
										))}
								</tbody>
							</table>
						</div>
					)}
				</div>
			)}
		</div>
	);
}
