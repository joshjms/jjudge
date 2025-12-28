"use client";

import { useEffect, useMemo, useState } from "react";

import { Button } from "@/components/ui/button";
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
	testcase_groups?: TestcaseGroup[];
};

type TestcaseGroup = {
	order_id: number;
	name: string;
	points: number;
	testcases: Testcase[];
};

type Testcase = {
	order_id: number;
	input: string;
	output: string;
	is_hidden: boolean;
};

type TestcaseGroupForm = {
	name: string;
	points: string;
	testcases: TestcaseForm[];
};

type TestcaseForm = {
	input: string;
	output: string;
	is_hidden: boolean;
};

type FormState = {
	title: string;
	description: string;
	difficulty: string;
	time_limit: string;
	memory_limit: string;
	tags: string;
	testcase_groups: TestcaseGroupForm[];
};

const emptyForm: FormState = {
	title: "",
	description: "",
	difficulty: "",
	time_limit: "",
	memory_limit: "",
	tags: "",
	testcase_groups: [],
};

const toFormGroups = (groups?: TestcaseGroup[]) =>
	groups?.map((group) => ({
		name: group.name ?? "",
		points: group.points?.toString() ?? "",
		testcases:
			group.testcases?.map((tc) => ({
				input: tc.input ?? "",
				output: tc.output ?? "",
				is_hidden: Boolean(tc.is_hidden),
			})) ?? [
				{
					input: "",
					output: "",
					is_hidden: false,
				},
			],
	})) ?? [];

const parseTags = (value: string) =>
	value
		.split(",")
		.map((tag) => tag.trim())
		.filter(Boolean);

export default function AdminProblemsPage() {
	const auth = useAuth();
	const [problems, setProblems] = useState<Problem[]>([]);
	const [loading, setLoading] = useState(false);
	const [saving, setSaving] = useState(false);
	const [loadingProblem, setLoadingProblem] = useState(false);
	const [error, setError] = useState<string | null>(null);
	const [form, setForm] = useState<FormState>(emptyForm);
	const [editingId, setEditingId] = useState<number | null>(null);
	const [testcaseFile, setTestcaseFile] = useState<File | null>(null);

	const hasToken = Boolean(auth.token);

	const isFormValid = useMemo(() => {
		return Boolean(form.title && form.description);
	}, [form.description, form.title]);

	const authHeaders = useMemo(
		() => (auth.token ? { Authorization: `Bearer ${auth.token}` } : undefined),
		[auth.token],
	);

	const loadProblems = async () => {
		setLoading(true);
		setError(null);
		try {
			const data = await api.get<Problem[]>("/problems", {
				headers: authHeaders,
			});
			setProblems(data ?? []);
		} catch (err) {
			setError("Failed to load problems.");
		} finally {
			setLoading(false);
		}
	};

	useEffect(() => {
		if (hasToken) {
			void loadProblems();
		}
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, [hasToken]);

	const resetForm = () => {
		setForm(emptyForm);
		setEditingId(null);
		setTestcaseFile(null);
	};

	const handleEdit = async (problem: Problem) => {
		setEditingId(problem.id);
		setLoadingProblem(true);
		try {
			const detailed = await api.get<Problem>(`/problems/${problem.id}`, { headers: authHeaders });
			setForm({
				title: detailed?.title ?? "",
				description: detailed?.description ?? "",
				difficulty: detailed?.difficulty?.toString() ?? "",
				time_limit: detailed?.time_limit?.toString() ?? "",
				memory_limit: detailed?.memory_limit?.toString() ?? "",
				tags: detailed?.tags?.join(", ") ?? "",
				testcase_groups: toFormGroups(detailed?.testcase_groups),
			});
		} catch {
			setForm({
				title: problem.title ?? "",
				description: problem.description ?? "",
				difficulty: problem.difficulty?.toString() ?? "",
				time_limit: problem.time_limit?.toString() ?? "",
				memory_limit: problem.memory_limit?.toString() ?? "",
				tags: problem.tags?.join(", ") ?? "",
				testcase_groups: toFormGroups(problem.testcase_groups),
			});
			setError("Failed to load full problem details; showing cached data.");
		} finally {
			setLoadingProblem(false);
		}
	};

	const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
		event.preventDefault();
		if (!hasToken) {
			setError("You must be signed in to manage problems.");
			return;
		}
		if (!isFormValid) {
			setError("Title and description are required.");
			return;
		}
		setSaving(true);
		setError(null);

		const payload = {
			title: form.title,
			description: form.description,
			difficulty: form.difficulty ? Number(form.difficulty) : undefined,
			time_limit: form.time_limit ? Number(form.time_limit) : undefined,
			memory_limit: form.memory_limit ? Number(form.memory_limit) : undefined,
			tags: parseTags(form.tags),
		};

		const body = new FormData();
		body.append("problem", JSON.stringify(payload));
		if (form.testcase_groups.length > 0) {
			const serializedGroups = form.testcase_groups.map((group, groupIndex) => ({
				order_id: groupIndex + 1,
				name: group.name,
				points: group.points ? Number(group.points) : 0,
				testcases: (group.testcases ?? []).map((tc, tcIndex) => ({
					order_id: tcIndex + 1,
					input: tc.input,
					output: tc.output,
					is_hidden: Boolean(tc.is_hidden),
				})),
			}));
			body.append("testcase_groups", JSON.stringify(serializedGroups));
		}
		if (testcaseFile) {
			body.append("file", testcaseFile);
		}

		try {
			if (editingId !== null) {
				await api.put(`/problems/${editingId}`, body, { headers: authHeaders });
			} else {
				await api.post("/problems", body, { headers: authHeaders });
			}
			resetForm();
			await loadProblems();
		} catch (err) {
			setError("Save failed. Check the data and try again.");
		} finally {
			setSaving(false);
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
		} catch (err) {
			setError("Delete failed. Try again.");
		}
	};

	const addGroup = () => {
		setForm((prev) => ({
			...prev,
			testcase_groups: [
				...prev.testcase_groups,
				{
					name: "",
					points: "",
					testcases: [
						{
							input: "",
							output: "",
							is_hidden: false,
						},
					],
				},
			],
		}));
	};

	const updateGroupField = (index: number, key: "name" | "points", value: string) => {
		setForm((prev) => {
			const nextGroups = [...prev.testcase_groups];
			const group = nextGroups[index];
			if (!group) return prev;
			nextGroups[index] = { ...group, [key]: value };
			return { ...prev, testcase_groups: nextGroups };
		});
	};

	const addTestcase = (groupIndex: number) => {
		setForm((prev) => {
			const nextGroups = [...prev.testcase_groups];
			const group = nextGroups[groupIndex];
			if (!group) return prev;
			nextGroups[groupIndex] = {
				...group,
				testcases: [
					...(group.testcases ?? []),
					{ input: "", output: "", is_hidden: false },
				],
			};
			return { ...prev, testcase_groups: nextGroups };
		});
	};

	const updateTestcaseField = (
		groupIndex: number,
		tcIndex: number,
		key: "input" | "output" | "is_hidden",
		value: string | boolean,
	) => {
		setForm((prev) => {
			const nextGroups = [...prev.testcase_groups];
			const group = nextGroups[groupIndex];
			if (!group) return prev;
			const nextTestcases = [...(group.testcases ?? [])];
			const tc = nextTestcases[tcIndex];
			if (!tc) return prev;
			nextTestcases[tcIndex] = { ...tc, [key]: value };
			nextGroups[groupIndex] = { ...group, testcases: nextTestcases };
			return { ...prev, testcase_groups: nextGroups };
		});
	};

	return (
		<div className="mx-auto max-w-6xl px-4 py-10">
			<div className="mb-8 flex items-center justify-between gap-4">
				<div>
					<p className="text-xs font-semibold uppercase tracking-[0.3em] text-primary">Admin</p>
					<h1 className="text-3xl font-bold leading-tight">Manage Problems</h1>
					<p className="text-sm text-muted-foreground">
						Create new problems, edit existing ones, or remove outdated entries.
					</p>
				</div>
				{editingId !== null ? (
					<Button
						type="button"
						variant="outline"
						className="rounded-none px-4 py-2"
						onClick={resetForm}
					>
						Cancel edit
					</Button>
				) : null}
			</div>

			{!hasToken ? (
				<p className="border border-destructive/50 bg-destructive/10 px-4 py-3 text-sm text-destructive">
					Sign in to access admin tools.
				</p>
			) : (
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
														<Button
															type="button"
															variant="ghost"
															className="rounded-none border border-border/70 px-3 py-2"
															onClick={() => handleEdit(problem)}
														>
															Edit
														</Button>
														<Button
															type="button"
															variant="outline"
															className="rounded-none px-3 py-2 text-destructive"
															onClick={() => handleDelete(problem.id)}
														>
															Delete
														</Button>
													</div>
												</td>
											</tr>
										))
									)}
								</tbody>
							</table>
						</div>
					</section>

					<section className="space-y-4 border border-border/70 bg-background/70 p-6">
						<h2 className="text-xl font-semibold">
							{editingId !== null ? "Edit problem" : "Create problem"}
						</h2>
						<form className="space-y-4" onSubmit={handleSubmit}>
							<div className="space-y-2">
								<label className="text-sm font-semibold text-foreground">Title</label>
								<input
									value={form.title}
									onChange={(e) => setForm((prev) => ({ ...prev, title: e.target.value }))}
									required
									className="w-full border border-border/70 bg-background px-3 py-2 text-sm outline-none focus:border-primary focus:ring-2 focus:ring-primary/30 rounded-none"
									placeholder="Problem title"
								/>
							</div>

							<div className="grid gap-4 sm:grid-cols-2">
								<div className="space-y-2">
									<label className="text-sm font-semibold text-foreground">Difficulty</label>
									<input
										type="number"
										min="0"
										value={form.difficulty}
										onChange={(e) =>
											setForm((prev) => ({ ...prev, difficulty: e.target.value }))
										}
										className="w-full border border-border/70 bg-background px-3 py-2 text-sm outline-none focus:border-primary focus:ring-2 focus:ring-primary/30 rounded-none"
										placeholder="0 = easiest"
									/>
								</div>
								<div className="space-y-2">
									<label className="text-sm font-semibold text-foreground">Time limit (ms)</label>
									<input
										type="number"
										min="0"
										value={form.time_limit}
										onChange={(e) =>
											setForm((prev) => ({ ...prev, time_limit: e.target.value }))
										}
										className="w-full border border-border/70 bg-background px-3 py-2 text-sm outline-none focus:border-primary focus:ring-2 focus:ring-primary/30 rounded-none"
										placeholder="1000"
									/>
								</div>
							</div>

							<div className="grid gap-4 sm:grid-cols-2">
								<div className="space-y-2">
									<label className="text-sm font-semibold text-foreground">Memory limit (MB)</label>
									<input
										type="number"
										min="0"
										value={form.memory_limit}
										onChange={(e) =>
											setForm((prev) => ({ ...prev, memory_limit: e.target.value }))
										}
										className="w-full border border-border/70 bg-background px-3 py-2 text-sm outline-none focus:border-primary focus:ring-2 focus:ring-primary/30 rounded-none"
										placeholder="256"
									/>
								</div>
								<div className="space-y-2">
									<label className="text-sm font-semibold text-foreground">Tags</label>
									<input
										value={form.tags}
										onChange={(e) => setForm((prev) => ({ ...prev, tags: e.target.value }))}
										className="w-full border border-border/70 bg-background px-3 py-2 text-sm outline-none focus:border-primary focus:ring-2 focus:ring-primary/30 rounded-none"
										placeholder="graph, bfs, dp"
									/>
									<p className="text-xs text-muted-foreground">Comma-separated list.</p>
								</div>
							</div>

							<div className="space-y-3">
								<div className="flex flex-wrap items-center justify-between gap-3">
									<div>
										<p className="text-sm font-semibold text-foreground">Testcase groups</p>
										<p className="text-xs text-muted-foreground">
											Add groups with testcases; order follows their position.
										</p>
									</div>
									<Button
										type="button"
										variant="outline"
										className="rounded-none px-3 py-2"
										onClick={addGroup}
									>
										New testcase group +
									</Button>
								</div>

								{form.testcase_groups.length === 0 ? (
									<p className="text-sm text-muted-foreground">
										No testcase groups yet. Click &quot;New testcase group&quot; to begin.
									</p>
								) : (
									<div className="space-y-4">
										{form.testcase_groups.map((group, groupIndex) => (
											<div key={groupIndex} className="border border-border/70 bg-muted/30 p-4">
												<div className="grid gap-4 sm:grid-cols-2">
													<div className="space-y-2">
														<label className="text-sm font-semibold text-foreground">
															Group name
														</label>
														<input
															value={group.name}
															onChange={(e) =>
																updateGroupField(groupIndex, "name", e.target.value)
															}
															className="w-full border border-border/70 bg-background px-3 py-2 text-sm outline-none focus:border-primary focus:ring-2 focus:ring-primary/30 rounded-none"
															placeholder="Sample tests"
														/>
													</div>
													<div className="space-y-2">
														<label className="text-sm font-semibold text-foreground">
															Points
														</label>
														<input
															type="number"
															min="0"
															value={group.points}
															onChange={(e) =>
																updateGroupField(groupIndex, "points", e.target.value)
															}
															className="w-full border border-border/70 bg-background px-3 py-2 text-sm outline-none focus:border-primary focus:ring-2 focus:ring-primary/30 rounded-none"
															placeholder="10"
														/>
													</div>
												</div>

												<div className="mt-4 space-y-3">
													<div className="flex items-center justify-between">
														<p className="text-sm font-semibold text-foreground">Testcases</p>
														<Button
															type="button"
															variant="ghost"
															className="rounded-none border border-border/70 px-3 py-1 text-xs"
															onClick={() => addTestcase(groupIndex)}
														>
															Add testcase +
														</Button>
													</div>
													{group.testcases?.length ? (
														<div className="space-y-3">
															{group.testcases.map((tc, tcIndex) => (
																<div
																	key={tcIndex}
																	className="grid gap-3 border border-border/60 bg-background/80 px-3 py-3 sm:grid-cols-2"
																>
																	<div className="space-y-2">
																		<label className="text-xs font-semibold text-foreground">
																			Input
																		</label>
																		<textarea
																			value={tc.input}
																			onChange={(e) =>
																				updateTestcaseField(
																					groupIndex,
																					tcIndex,
																					"input",
																					e.target.value,
																				)
																			}
																			rows={2}
																			className="w-full border border-border/70 bg-background px-3 py-2 text-sm outline-none focus:border-primary focus:ring-2 focus:ring-primary/30 rounded-none font-mono"
																			placeholder="Input text"
																		/>
																	</div>
																	<div className="space-y-2">
																		<label className="text-xs font-semibold text-foreground">
																			Output
																		</label>
																		<textarea
																			value={tc.output}
																			onChange={(e) =>
																				updateTestcaseField(
																					groupIndex,
																					tcIndex,
																					"output",
																					e.target.value,
																				)
																			}
																			rows={2}
																			className="w-full border border-border/70 bg-background px-3 py-2 text-sm outline-none focus:border-primary focus:ring-2 focus:ring-primary/30 rounded-none font-mono"
																			placeholder="Expected output"
																		/>
																	</div>
																	<label className="flex items-center gap-2 text-xs font-semibold text-foreground">
																		<input
																			type="checkbox"
																			checked={tc.is_hidden}
																			onChange={(e) =>
																				updateTestcaseField(
																					groupIndex,
																					tcIndex,
																					"is_hidden",
																					e.target.checked,
																				)
																			}
																			className="h-4 w-4 border border-border/70"
																		/>
																		Hidden?
																	</label>
																</div>
															))}
														</div>
													) : (
														<p className="text-xs text-muted-foreground">
															No testcases in this group yet.
														</p>
													)}
												</div>
											</div>
										))}
									</div>
								)}
							</div>

							<div className="space-y-2">
								<label className="text-sm font-semibold text-foreground">Description</label>
								<textarea
									value={form.description}
									onChange={(e) =>
										setForm((prev) => ({ ...prev, description: e.target.value }))
									}
									required
									rows={8}
									className="w-full border border-border/70 bg-background px-3 py-2 text-sm outline-none focus:border-primary focus:ring-2 focus:ring-primary/30 rounded-none"
									placeholder="Markdown supported."
								/>
							</div>

							<div className="space-y-2">
								<label className="text-sm font-semibold text-foreground">Testcases (.tar.gz)</label>
								<input
									type="file"
									accept=".tar.gz"
									onChange={(e) => {
										setTestcaseFile(e.target.files?.[0] ?? null);
									}}
									className="w-full border border-border/70 bg-background px-3 py-2 text-sm outline-none focus:border-primary focus:ring-2 focus:ring-primary/30 rounded-none file:mr-3 file:border-0 file:bg-muted file:px-3 file:py-2 file:text-sm file:font-semibold file:text-foreground file:rounded-none"
								/>
								<p className="text-xs text-muted-foreground">
									Optional: attach a .tar.gz archive. It will be stored with the problem on save.
								</p>
								{testcaseFile && (
									<p className="text-xs text-muted-foreground">Selected: {testcaseFile.name}</p>
								)}
							</div>

							{error && (
								<p className="border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive">
									{error}
								</p>
							)}

							<div className="flex flex-wrap items-center gap-3">
								<Button
									type="submit"
									className="rounded-none px-4 py-2"
									disabled={saving || !isFormValid}
								>
									{saving
										? "Saving..."
										: editingId !== null
											? "Update problem"
											: "Create problem"}
								</Button>
								{editingId !== null ? (
									<Button
										type="button"
										variant="ghost"
										className="rounded-none border border-border/70 px-4 py-2"
										onClick={resetForm}
									>
										Reset
									</Button>
								) : null}
							</div>
						</form>
					</section>
				</div>
			)}
		</div>
	);
}
