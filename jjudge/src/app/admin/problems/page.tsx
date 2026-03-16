"use client";

import { useEffect, useMemo, useState } from "react";

import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";

type Problem = {
	id: number;
	title?: string;
	description?: string;
	difficulty?: number;
	time_limit?: number;
	memory_limit?: number;
	tags?: string[];
	testcase_groups?: Array<{
		testcases?: Array<{
			in_key?: string;
			out_key?: string;
		}>;
	}>;
};

type Testcase = {
	id: string;
	ordinal: number;
	inFile: File | null;
	outFile: File | null;
};

type TestcaseGroup = {
	id: string;
	ordinal: number;
	name: string;
	points: number;
	testcases: Testcase[];
};

const createTestcase = (index: number): Testcase => ({
	id: crypto.randomUUID(),
	ordinal: index,
	inFile: null,
	outFile: null,
});

const createGroup = (index: number): TestcaseGroup => ({
	id: crypto.randomUUID(),
	ordinal: index,
	name: index === 0 ? "Sample" : `Group ${index + 1}`,
	points: 100,
	testcases: [createTestcase(0)],
});

const getInputKey = (groupIndex: number, testcaseIndex: number) =>
	`group_${groupIndex + 1}_case_${testcaseIndex + 1}.in`;

const getOutputKey = (groupIndex: number, testcaseIndex: number) =>
	`group_${groupIndex + 1}_case_${testcaseIndex + 1}.out`;

const normalizeProblems = (payload: unknown): Problem[] => {
	if (Array.isArray(payload)) return payload as Problem[];
	if (payload && typeof payload === "object") {
		const maybeWrapped = payload as {
			problems?: Problem[];
			data?: Problem[];
			items?: Problem[];
		};
		if (Array.isArray(maybeWrapped.problems)) return maybeWrapped.problems;
		if (Array.isArray(maybeWrapped.data)) return maybeWrapped.data;
		if (Array.isArray(maybeWrapped.items)) return maybeWrapped.items;
	}
	return [];
};

export default function AdminProblemsPage() {
	const auth = useAuth();
	const [problems, setProblems] = useState<Problem[]>([]);
	const [loading, setLoading] = useState(false);
	const [loadingProblem, setLoadingProblem] = useState(false);
	const [error, setError] = useState<string | null>(null);
	const [editingId, setEditingId] = useState<number | null>(null);

	const [title, setTitle] = useState("Sample Problem");
	const [description, setDescription] = useState("Explain the task here.");
	const [difficulty, setDifficulty] = useState(0);
	const [timeLimit, setTimeLimit] = useState(1000);
	const [memoryLimit, setMemoryLimit] = useState(268435456);
	const [tags, setTags] = useState("arrays, math");
	const [groups, setGroups] = useState<TestcaseGroup[]>([createGroup(0)]);
	const [submitStatus, setSubmitStatus] = useState<string | null>(null);

	const hasToken = Boolean(auth.token);

	const authHeaders = useMemo(
		() => (auth.token ? { Authorization: `Bearer ${auth.token}` } : undefined),
		[auth.token],
	);

	const metadata = useMemo(() => {
		const normalizedTags = tags
			.split(",")
			.map((tag) => tag.trim())
			.filter(Boolean);

		return {
			title,
			description,
			difficulty,
			time_limit: timeLimit,
			memory_limit: memoryLimit,
			tags: normalizedTags,
			testcase_groups: groups.map((group, groupIndex) => ({
				ordinal: groupIndex,
				name: group.name,
				points: group.points,
				testcases: group.testcases.map((testcase, testcaseIndex) => ({
					ordinal: testcaseIndex,
					in_key: getInputKey(groupIndex, testcaseIndex),
					out_key: getOutputKey(groupIndex, testcaseIndex),
				})),
			})),
		};
	}, [description, difficulty, groups, memoryLimit, tags, timeLimit, title]);

	const loadProblems = async () => {
		if (!hasToken) return;
		setLoading(true);
		setError(null);
		try {
			const data = await api.get<unknown>("/problems", {
				headers: authHeaders,
			});
			setProblems(normalizeProblems(data));
		} catch {
			setError("Failed to load problems.");
		} finally {
			setLoading(false);
		}
	};

	useEffect(() => {
		void loadProblems();
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, [hasToken]);

	const resetForm = () => {
		setEditingId(null);
		setTitle("Sample Problem");
		setDescription("Explain the task here.");
		setDifficulty(0);
		setTimeLimit(1000);
		setMemoryLimit(268435456);
		setTags("arrays, math");
		setGroups([createGroup(0)]);
		setSubmitStatus(null);
		setError(null);
	};

	const handleEdit = async (problem: Problem) => {
		if (!hasToken) {
			setError("You must be signed in to manage problems.");
			return;
		}
		setEditingId(problem.id);
		setLoadingProblem(true);
		setError(null);
		try {
			const detailed = await api.get<Problem>(`/problems/${problem.id}`, {
				headers: authHeaders,
			});
			setTitle(detailed?.title ?? "");
			setDescription(detailed?.description ?? "");
			setDifficulty(detailed?.difficulty ?? 0);
			setTimeLimit(detailed?.time_limit ?? 1000);
			setMemoryLimit(detailed?.memory_limit ?? 268435456);
			setTags(detailed?.tags?.join(", ") ?? "");
			if (detailed?.testcase_groups?.length) {
				setGroups(
					detailed.testcase_groups.map((group, groupIndex) => ({
						id: crypto.randomUUID(),
						ordinal: groupIndex,
						name: groupIndex === 0 ? "Sample" : `Group ${groupIndex + 1}`,
						points: 100,
						testcases: (group.testcases ?? []).map((_, tcIndex) =>
							createTestcase(tcIndex),
						),
					})),
				);
			} else {
				setGroups([createGroup(0)]);
			}
		} catch {
			setError("Failed to load full problem details.");
		} finally {
			setLoadingProblem(false);
		}
	};

	const handleDelete = async (id: number) => {
		if (!hasToken) {
			setError("You must be signed in to manage problems.");
			return;
		}
		if (!confirm("Delete this problem? This cannot be undone.")) return;
		setError(null);
		try {
			await api.delete(`/problems/${id}`, { headers: authHeaders });
			if (editingId === id) {
				resetForm();
			}
			await loadProblems();
		} catch {
			setError("Delete failed. Try again.");
		}
	};

	const updateGroup = (
		groupId: string,
		updater: (group: TestcaseGroup) => TestcaseGroup,
	) => {
		setGroups((current) =>
			current.map((group) => (group.id === groupId ? updater(group) : group)),
		);
	};

	const updateTestcase = (
		groupId: string,
		testcaseId: string,
		updater: (testcase: Testcase) => Testcase,
	) => {
		setGroups((current) =>
			current.map((group) => {
				if (group.id !== groupId) {
					return group;
				}

				return {
					...group,
					testcases: group.testcases.map((testcase) =>
						testcase.id === testcaseId ? updater(testcase) : testcase,
					),
				};
			}),
		);
	};

	const addGroup = () => {
		setGroups((current) => [...current, createGroup(current.length)]);
	};

	const removeGroup = (groupId: string) => {
		setGroups((current) => current.filter((group) => group.id !== groupId));
	};

	const addTestcase = (groupId: string) => {
		updateGroup(groupId, (group) => ({
			...group,
			testcases: [...group.testcases, createTestcase(group.testcases.length)],
		}));
	};

	const removeTestcase = (groupId: string, testcaseId: string) => {
		updateGroup(groupId, (group) => ({
			...group,
			testcases: group.testcases.filter((testcase) => testcase.id !== testcaseId),
		}));
	};

	const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
		event.preventDefault();
		setSubmitStatus(null);
		setError(null);

		if (!hasToken) {
			setError("You must be signed in to create or update problems.");
			return;
		}

		const formData = new FormData();
		formData.append("metadata", JSON.stringify(metadata));

		groups.forEach((group, groupIndex) => {
			group.testcases.forEach((testcase, testcaseIndex) => {
				const inKey = getInputKey(groupIndex, testcaseIndex);
				const outKey = getOutputKey(groupIndex, testcaseIndex);

				if (testcase.inFile) {
					formData.append(inKey, testcase.inFile);
				}
				if (testcase.outFile) {
					formData.append(outKey, testcase.outFile);
				}
			});
		});

		try {
			setSubmitStatus("Saving...");
			if (editingId !== null) {
				await api.put(`/problems/${editingId}`, formData, { headers: authHeaders });
			} else {
				await api.post("/problems", formData, { headers: authHeaders });
			}
			setSubmitStatus(editingId !== null ? "Update succeeded." : "Upload succeeded.");
			await loadProblems();
		} catch (submitError) {
			console.error(submitError);
			setSubmitStatus("Save failed. Check the data and try again.");
		}
	};

	return (
		<div className="mx-auto max-w-6xl px-4 py-10 space-y-8">
			<header className="space-y-2">
				<p className="text-xs font-semibold uppercase tracking-[0.3em] text-primary">
					Admin
				</p>
				<h1 className="text-3xl font-bold">Create Problem</h1>
				<p className="text-sm text-muted-foreground">
					Fill in the problem details and attach input/output files for each testcase.
				</p>
			</header>

			{!hasToken ? (
				<p className="border border-destructive/50 bg-destructive/10 px-4 py-3 text-sm text-destructive">
					Sign in to access admin tools.
				</p>
			) : null}

			<div className="grid gap-10 lg:grid-cols-[1.1fr,0.9fr]">
				<section className="space-y-4">
					<div className="flex items-center justify-between">
						<h2 className="text-xl font-semibold">Existing problems</h2>
						{(loading || loadingProblem) && (
							<span className="text-xs text-muted-foreground">Loading…</span>
						)}
					</div>
					<div className="overflow-x-auto border border-border/70">
						<table className="w-full border-collapse text-sm">
							<thead className="bg-muted/60">
								<tr>
									<th className="border border-border/70 px-3 py-2 text-left font-semibold">
										Title
									</th>
									<th className="border border-border/70 px-3 py-2 text-left font-semibold">
										Difficulty
									</th>
									<th className="border border-border/70 px-3 py-2 text-left font-semibold">
										Tags
									</th>
									<th className="border border-border/70 px-3 py-2 text-left font-semibold">
										Actions
									</th>
								</tr>
							</thead>
							<tbody>
								{problems.length === 0 ? (
									<tr>
										<td
											className="border border-border/70 px-3 py-3 text-muted-foreground"
											colSpan={4}
										>
											No problems found.
										</td>
									</tr>
								) : (
									problems.map((problem) => (
										<tr key={problem.id} className="hover:bg-muted/50">
											<td className="border border-border/70 px-3 py-2">
												<div className="font-semibold text-foreground">
													{problem.title ?? "Untitled"}
												</div>
											</td>
											<td className="border border-border/70 px-3 py-2 text-sm">
												{problem.difficulty ?? "—"}
											</td>
											<td className="border border-border/70 px-3 py-2 text-sm text-muted-foreground">
												{problem.tags?.length ? problem.tags.join(", ") : "—"}
											</td>
											<td className="border border-border/70 px-3 py-2">
												<div className="flex flex-wrap items-center gap-2">
													<button
														type="button"
														className="border border-border/70 px-3 py-1 text-xs"
														onClick={() => handleEdit(problem)}
													>
														Edit
													</button>
													<button
														type="button"
														className="border border-border/70 px-3 py-1 text-xs text-destructive"
														onClick={() => handleDelete(problem.id)}
													>
														Delete
													</button>
												</div>
											</td>
										</tr>
									))
								)}
							</tbody>
						</table>
					</div>
					{error && (
						<p className="border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive">
							{error}
						</p>
					)}
				</section>

				<section className="space-y-6 border border-border/70 bg-background/70 p-6">
					<div className="flex items-center justify-between gap-3">
						<h2 className="text-xl font-semibold">
							{editingId !== null ? "Edit problem" : "Problem details"}
						</h2>
						{editingId !== null && (
							<button
								type="button"
								className="border border-border/70 px-3 py-1 text-sm"
								onClick={resetForm}
							>
								Cancel edit
							</button>
						)}
					</div>
					<form className="space-y-6" onSubmit={handleSubmit}>
						<div className="grid gap-4 md:grid-cols-2">
							<label className="space-y-1">
								<span className="text-sm font-medium">Title</span>
								<input
									className="w-full border border-border/70 bg-background px-3 py-2"
									value={title}
									onChange={(event) => setTitle(event.target.value)}
								/>
							</label>
							<label className="space-y-1">
								<span className="text-sm font-medium">Difficulty</span>
								<input
									className="w-full border border-border/70 bg-background px-3 py-2"
									type="number"
									value={difficulty}
									onChange={(event) =>
										setDifficulty(Number(event.target.value || 0))
									}
								/>
							</label>
							<label className="space-y-1 md:col-span-2">
								<span className="text-sm font-medium">Description</span>
								<textarea
									className="w-full border border-border/70 bg-background px-3 py-2 min-h-[120px]"
									value={description}
									onChange={(event) => setDescription(event.target.value)}
								/>
							</label>
							<label className="space-y-1">
								<span className="text-sm font-medium">Time limit (ms)</span>
								<input
									className="w-full border border-border/70 bg-background px-3 py-2"
									type="number"
									value={timeLimit}
									onChange={(event) =>
										setTimeLimit(Number(event.target.value || 0))
									}
								/>
							</label>
							<label className="space-y-1">
								<span className="text-sm font-medium">Memory limit (bytes)</span>
								<input
									className="w-full border border-border/70 bg-background px-3 py-2"
									type="number"
									value={memoryLimit}
									onChange={(event) =>
										setMemoryLimit(Number(event.target.value || 0))
									}
								/>
							</label>
							<label className="space-y-1 md:col-span-2">
								<span className="text-sm font-medium">Tags (comma-separated)</span>
								<input
									className="w-full border border-border/70 bg-background px-3 py-2"
									value={tags}
									onChange={(event) => setTags(event.target.value)}
								/>
							</label>
						</div>

						<div className="space-y-6">
									<div className="flex items-center justify-between">
										<h3 className="text-lg font-semibold">Testcase groups</h3>
										<button
											type="button"
											className="border border-border/70 px-3 py-1 text-sm"
											onClick={addGroup}
										>
											Add group
										</button>
									</div>

							{groups.map((group, groupIndex) => (
								<div
									key={group.id}
									className="border border-border/70 p-4 space-y-4"
								>
									<div className="flex flex-wrap gap-4 items-end">
										<label className="flex-1 space-y-1 min-w-[200px]">
											<span className="text-sm font-medium">Group name</span>
											<input
												className="w-full border border-border/70 bg-background px-3 py-2"
												value={group.name}
												onChange={(event) =>
													updateGroup(group.id, (current) => ({
														...current,
														name: event.target.value,
													}))
												}
											/>
										</label>
										<label className="space-y-1 w-32">
											<span className="text-sm font-medium">Points</span>
											<input
												className="w-full border border-border/70 bg-background px-3 py-2"
												type="number"
												value={group.points}
												onChange={(event) =>
													updateGroup(group.id, (current) => ({
														...current,
														points: Number(event.target.value || 0),
													}))
												}
											/>
										</label>
										{groups.length > 1 && (
											<button
												type="button"
												className="border border-border/70 px-3 py-1 text-sm"
												onClick={() => removeGroup(group.id)}
											>
												Remove
											</button>
										)}
									</div>

									<div className="space-y-3">
										<div className="flex items-center justify-between">
											<h4 className="font-medium">Testcases</h4>
											<button
												type="button"
												className="border border-border/70 px-3 py-1 text-sm"
												onClick={() => addTestcase(group.id)}
											>
												Add testcase
											</button>
										</div>
								{group.testcases.map((testcase, testcaseIndex) => (
									<div
										key={testcase.id}
										className="border border-border/70 p-3 space-y-3"
									>
										<div className="flex flex-wrap gap-4 items-end justify-between">
											<div className="text-xs text-muted-foreground">
												Keys are generated automatically.
											</div>
											{group.testcases.length > 1 && (
												<button
													type="button"
													className="border border-border/70 px-3 py-1 text-sm"
													onClick={() =>
														removeTestcase(group.id, testcase.id)
													}
												>
													Remove
												</button>
											)}
										</div>

										<div className="grid gap-4 md:grid-cols-2">
											<label className="space-y-1">
												<span className="text-sm font-medium">Input file</span>
												<input
													className="w-full text-sm"
													type="file"
													name={getInputKey(groupIndex, testcaseIndex)}
													onChange={(event) =>
														updateTestcase(group.id, testcase.id, (current) => ({
															...current,
															inFile: event.target.files?.[0] ?? null,
														}))
													}
												/>
											</label>
											<label className="space-y-1">
												<span className="text-sm font-medium">Output file</span>
												<input
													className="w-full text-sm"
													type="file"
													name={getOutputKey(groupIndex, testcaseIndex)}
													onChange={(event) =>
														updateTestcase(group.id, testcase.id, (current) => ({
															...current,
															outFile: event.target.files?.[0] ?? null,
														}))
													}
												/>
											</label>
										</div>
									</div>
								))}
									</div>
								</div>
							))}
						</div>

						<div className="flex flex-wrap items-center gap-4">
							<button
								type="submit"
								className="border border-border/70 bg-foreground text-background px-4 py-2 text-sm"
								disabled={!hasToken}
							>
								{editingId !== null ? "Update problem" : "Create problem"}
							</button>
							{submitStatus && (
								<span className="text-sm text-muted-foreground">{submitStatus}</span>
							)}
						</div>
					</form>
				</section>
			</div>
		</div>
	);
}
