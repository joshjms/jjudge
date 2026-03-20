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
	visibility?: string;
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
	const [visibility, setVisibility] = useState<"public" | "private">("public");
	const [groups, setGroups] = useState<TestcaseGroup[]>([createGroup(0)]);
	const [uploadMode, setUploadMode] = useState<"individual" | "zip">("individual");
	const [zipFile, setZipFile] = useState<File | null>(null);
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
			visibility,
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
	}, [description, difficulty, groups, memoryLimit, tags, timeLimit, title, visibility]);

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
		setVisibility("public");
		setGroups([createGroup(0)]);
		setUploadMode("individual");
		setZipFile(null);
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
			setVisibility(detailed?.visibility === "private" ? "private" : "public");
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

		try {
			setSubmitStatus("Saving...");

			if (uploadMode === "zip") {
				if (!zipFile) {
					setSubmitStatus(null);
					setError("Please select a ZIP file.");
					return;
				}
				const zipMetadata = {
					...metadata,
					testcase_groups: groups.map((group, groupIndex) => ({
						ordinal: groupIndex,
						name: group.name,
						points: group.points,
					})),
				};
				formData.append("metadata", JSON.stringify(zipMetadata));
				formData.append("testcases_zip", zipFile);
				if (editingId !== null) {
					await api.put(`/problems/${editingId}/zip`, formData, { headers: authHeaders });
				} else {
					await api.post("/problems/zip", formData, { headers: authHeaders });
				}
			} else {
				formData.append("metadata", JSON.stringify(metadata));
				groups.forEach((group, groupIndex) => {
					group.testcases.forEach((testcase, testcaseIndex) => {
						if (testcase.inFile) formData.append(getInputKey(groupIndex, testcaseIndex), testcase.inFile);
						if (testcase.outFile) formData.append(getOutputKey(groupIndex, testcaseIndex), testcase.outFile);
					});
				});
				if (editingId !== null) {
					await api.put(`/problems/${editingId}`, formData, { headers: authHeaders });
				} else {
					await api.post("/problems", formData, { headers: authHeaders });
				}
			}

			setSubmitStatus(editingId !== null ? "Update succeeded." : "Upload succeeded.");
			await loadProblems();
		} catch (submitError) {
			console.error(submitError);
			setSubmitStatus("Save failed. Check the data and try again.");
		}
	};

	return (
		<div className="space-y-8">
			{/* Header */}
			<div className="flex items-baseline gap-4 border-b border-border/60 pb-5">
				<h1 className="font-display text-5xl text-foreground">PROBLEMS</h1>
				<span className="text-xs font-mono text-muted-foreground tracking-widest">
					{problems.length} TOTAL
				</span>
			</div>

			{!hasToken ? (
				<p className="border border-destructive/50 bg-destructive/10 px-4 py-3 text-xs font-mono text-destructive">
					Sign in to access admin tools.
				</p>
			) : null}

			<div className="grid gap-10 lg:grid-cols-[1.1fr,0.9fr]">
				<section className="space-y-4">
					<div className="flex items-center justify-between border-b border-border/60 pb-2">
						<span className="text-[10px] font-mono text-muted-foreground tracking-widest">
							EXISTING PROBLEMS
						</span>
						{(loading || loadingProblem) && (
							<span className="text-[10px] font-mono text-muted-foreground tracking-widest">LOADING…</span>
						)}
					</div>

					{problems.length === 0 ? (
						<p className="py-8 text-center font-display text-3xl text-muted-foreground/30">NONE YET</p>
					) : (
						<div className="flex flex-col divide-y divide-border/40">
							{problems.map((problem) => (
								<div key={problem.id} className="group flex items-center gap-3 py-3 px-1 hover:bg-muted/30 transition-colors">
									<div className="flex-1 min-w-0">
										<p className="text-sm font-semibold text-foreground truncate">{problem.title ?? "Untitled"}</p>
										<p className="text-[10px] font-mono text-muted-foreground/60 tracking-widest mt-0.5">
											{problem.tags?.length ? problem.tags.join(", ") : "no tags"} · {problem.visibility ?? "public"}
										</p>
									</div>
									<div className="flex items-center gap-2 shrink-0">
										<button
											type="button"
											className="border border-border/60 px-2.5 py-1 text-[10px] font-mono tracking-widest text-muted-foreground hover:border-primary/60 hover:text-primary transition-colors"
											onClick={() => handleEdit(problem)}
										>
											EDIT
										</button>
										<button
											type="button"
											className="border border-destructive/40 px-2.5 py-1 text-[10px] font-mono tracking-widest text-destructive/70 hover:border-destructive hover:text-destructive transition-colors"
											onClick={() => handleDelete(problem.id)}
										>
											DEL
										</button>
									</div>
								</div>
							))}
						</div>
					)}

					{error && (
						<p className="border border-destructive/50 bg-destructive/10 px-3 py-2 text-xs font-mono text-destructive">
							{error}
						</p>
					)}
				</section>

				<section className="space-y-6 border border-border/60 bg-background/70 p-6">
					<div className="flex items-center justify-between gap-3 border-b border-border/60 pb-4">
						<span className="text-[10px] font-mono tracking-widest text-muted-foreground">
							{editingId !== null ? "EDIT PROBLEM" : "NEW PROBLEM"}
						</span>
						{editingId !== null && (
							<button
								type="button"
								className="border border-border/60 px-2.5 py-1 text-[10px] font-mono tracking-widest text-muted-foreground hover:border-primary/60 hover:text-primary transition-colors"
								onClick={resetForm}
							>
								CANCEL
							</button>
						)}
					</div>
					<form className="space-y-5" onSubmit={handleSubmit}>
						<div className="grid gap-4 md:grid-cols-2">
							<label className="space-y-1.5">
								<span className="text-[10px] font-mono font-semibold uppercase tracking-widest text-muted-foreground">Title</span>
								<input
									className="w-full border border-border/60 bg-background px-3 py-2 text-sm outline-none focus:border-primary/60 transition-colors"
									value={title}
									onChange={(event) => setTitle(event.target.value)}
								/>
							</label>
							<label className="space-y-1.5">
								<span className="text-[10px] font-mono font-semibold uppercase tracking-widest text-muted-foreground">Difficulty</span>
								<input
									className="w-full border border-border/60 bg-background px-3 py-2 text-sm outline-none focus:border-primary/60 transition-colors"
									type="number"
									value={difficulty}
									onChange={(event) =>
										setDifficulty(Number(event.target.value || 0))
									}
								/>
							</label>
							<label className="space-y-1.5 md:col-span-2">
								<span className="text-[10px] font-mono font-semibold uppercase tracking-widest text-muted-foreground">Description</span>
								<textarea
									className="w-full border border-border/60 bg-background px-3 py-2 text-sm min-h-[120px] outline-none focus:border-primary/60 transition-colors"
									value={description}
									onChange={(event) => setDescription(event.target.value)}
								/>
							</label>
							<label className="space-y-1.5">
								<span className="text-[10px] font-mono font-semibold uppercase tracking-widest text-muted-foreground">Time limit (ms)</span>
								<input
									className="w-full border border-border/60 bg-background px-3 py-2 text-sm outline-none focus:border-primary/60 transition-colors"
									type="number"
									value={timeLimit}
									onChange={(event) =>
										setTimeLimit(Number(event.target.value || 0))
									}
								/>
							</label>
							<label className="space-y-1.5">
								<span className="text-[10px] font-mono font-semibold uppercase tracking-widest text-muted-foreground">Memory limit (bytes)</span>
								<input
									className="w-full border border-border/60 bg-background px-3 py-2 text-sm outline-none focus:border-primary/60 transition-colors"
									type="number"
									value={memoryLimit}
									onChange={(event) =>
										setMemoryLimit(Number(event.target.value || 0))
									}
								/>
							</label>
							<label className="space-y-1.5 md:col-span-2">
								<span className="text-[10px] font-mono font-semibold uppercase tracking-widest text-muted-foreground">Tags (comma-separated)</span>
								<input
									className="w-full border border-border/60 bg-background px-3 py-2 text-sm outline-none focus:border-primary/60 transition-colors"
									value={tags}
									onChange={(event) => setTags(event.target.value)}
								/>
							</label>
							<label className="space-y-1.5 md:col-span-2">
								<span className="text-[10px] font-mono font-semibold uppercase tracking-widest text-muted-foreground">Visibility</span>
								<select
									className="w-full border border-border/60 bg-background px-3 py-2 text-sm outline-none focus:border-primary/60 transition-colors"
									value={visibility}
									onChange={(event) => setVisibility(event.target.value as "public" | "private")}
								>
									<option value="public">Public</option>
									<option value="private">Private</option>
								</select>
							</label>
						</div>

						<div className="space-y-5">
							<div className="flex items-center justify-between border-b border-border/60 pb-2">
								<span className="text-[10px] font-mono tracking-widest text-muted-foreground">TESTCASE GROUPS</span>
								<div className="flex items-center gap-2">
									<button
										type="button"
										className={`px-2.5 py-1 text-[10px] font-mono tracking-widest border transition-colors ${uploadMode === "individual" ? "border-primary/60 text-primary bg-primary/5" : "border-border/60 text-muted-foreground hover:border-primary/60 hover:text-primary"}`}
										onClick={() => setUploadMode("individual")}
									>
										INDIVIDUAL
									</button>
									<button
										type="button"
										className={`px-2.5 py-1 text-[10px] font-mono tracking-widest border transition-colors ${uploadMode === "zip" ? "border-primary/60 text-primary bg-primary/5" : "border-border/60 text-muted-foreground hover:border-primary/60 hover:text-primary"}`}
										onClick={() => setUploadMode("zip")}
									>
										ZIP
									</button>
									<button
										type="button"
										className="border border-border/60 px-2.5 py-1 text-[10px] font-mono tracking-widest text-muted-foreground hover:border-primary/60 hover:text-primary transition-colors"
										onClick={addGroup}
									>
										+ ADD GROUP
									</button>
								</div>
							</div>

							{uploadMode === "zip" && (
								<div className="border border-border/60 p-4 space-y-3">
									<label className="space-y-1.5 block">
										<span className="text-[10px] font-mono font-semibold uppercase tracking-widest text-muted-foreground">ZIP file</span>
										<input
											className="w-full text-sm"
											type="file"
											accept=".zip,.tar.gz,.tgz"
											onChange={(event) => setZipFile(event.target.files?.[0] ?? null)}
										/>
									</label>
									<p className="text-[10px] font-mono text-muted-foreground/60 tracking-wide">
										Accepts <code className="text-foreground/70">.zip</code> or <code className="text-foreground/70">.tar.gz</code>. Files must be named <code className="text-foreground/70">&lt;subtask&gt;_&lt;testcase&gt;.in</code> / <code className="text-foreground/70">.out</code> — e.g. <code className="text-foreground/70">0_0.in</code>, <code className="text-foreground/70">1_2.out</code>
									</p>
								</div>
							)}

							{groups.map((group, groupIndex) => (
								<div
									key={group.id}
									className="border border-border/60 p-4 space-y-4"
								>
									<div className="flex flex-wrap gap-4 items-end">
										<label className="flex-1 space-y-1.5 min-w-[200px]">
											<span className="text-[10px] font-mono font-semibold uppercase tracking-widest text-muted-foreground">Group name</span>
											<input
												className="w-full border border-border/60 bg-background px-3 py-2 text-sm outline-none focus:border-primary/60 transition-colors"
												value={group.name}
												onChange={(event) =>
													updateGroup(group.id, (current) => ({
														...current,
														name: event.target.value,
													}))
												}
											/>
										</label>
										<label className="space-y-1.5 w-32">
											<span className="text-[10px] font-mono font-semibold uppercase tracking-widest text-muted-foreground">Points</span>
											<input
												className="w-full border border-border/60 bg-background px-3 py-2 text-sm outline-none focus:border-primary/60 transition-colors"
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
												className="border border-destructive/40 px-2.5 py-1 text-[10px] font-mono tracking-widest text-destructive/70 hover:border-destructive hover:text-destructive transition-colors"
												onClick={() => removeGroup(group.id)}
											>
												REMOVE
											</button>
										)}
									</div>

									{uploadMode === "individual" && (
									<div className="space-y-3">
										<div className="flex items-center justify-between">
											<span className="text-[10px] font-mono tracking-widest text-muted-foreground/60">TESTCASES</span>
											<button
												type="button"
												className="border border-border/60 px-2.5 py-1 text-[10px] font-mono tracking-widest text-muted-foreground hover:border-primary/60 hover:text-primary transition-colors"
												onClick={() => addTestcase(group.id)}
											>
												+ ADD
											</button>
										</div>
										{group.testcases.map((testcase, testcaseIndex) => (
											<div
												key={testcase.id}
												className="border border-border/60 p-3 space-y-3"
											>
												<div className="flex flex-wrap gap-4 items-end justify-between">
													<span className="text-[10px] font-mono text-muted-foreground/60 tracking-widest">
														CASE {testcaseIndex + 1} — keys auto-generated
													</span>
													{group.testcases.length > 1 && (
														<button
															type="button"
															className="border border-destructive/40 px-2.5 py-1 text-[10px] font-mono tracking-widest text-destructive/70 hover:border-destructive hover:text-destructive transition-colors"
															onClick={() =>
																removeTestcase(group.id, testcase.id)
															}
														>
															REMOVE
														</button>
													)}
												</div>

												<div className="grid gap-4 md:grid-cols-2">
													<label className="space-y-1.5">
														<span className="text-[10px] font-mono font-semibold uppercase tracking-widest text-muted-foreground">Input file</span>
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
													<label className="space-y-1.5">
														<span className="text-[10px] font-mono font-semibold uppercase tracking-widest text-muted-foreground">Output file</span>
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
									)}
								</div>
							))}
						</div>

						<div className="flex flex-wrap items-center gap-4 pt-2">
							<button
								type="submit"
								className="border border-foreground/60 bg-foreground text-background px-4 py-2 text-[10px] font-mono tracking-widest hover:bg-foreground/90 transition-colors disabled:opacity-40"
								disabled={!hasToken}
							>
								{editingId !== null ? "UPDATE PROBLEM" : "CREATE PROBLEM"}
							</button>
							{submitStatus && (
								<span className="text-[10px] font-mono text-muted-foreground tracking-widest">{submitStatus}</span>
							)}
						</div>
					</form>
				</section>
			</div>
		</div>
	);
}
