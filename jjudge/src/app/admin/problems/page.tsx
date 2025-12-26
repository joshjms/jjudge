"use client";

import { useEffect, useMemo, useState } from "react";

import { Button } from "@/components/ui/button";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";

type Problem = {
	id: number;
	title?: string;
	slug?: string;
	description?: string;
	difficulty?: string;
	time_limit?: number;
	memory_limit?: number;
	tags?: string[];
};

type FormState = {
	title: string;
	slug: string;
	description: string;
	difficulty: string;
	time_limit: string;
	memory_limit: string;
	tags: string;
};

const emptyForm: FormState = {
	title: "",
	slug: "",
	description: "",
	difficulty: "",
	time_limit: "",
	memory_limit: "",
	tags: "",
};

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
	const [error, setError] = useState<string | null>(null);
	const [uploadError, setUploadError] = useState<string | null>(null);
	const [uploadMessage, setUploadMessage] = useState<string | null>(null);
	const [form, setForm] = useState<FormState>(emptyForm);
	const [editingId, setEditingId] = useState<number | null>(null);
	const [testcaseFile, setTestcaseFile] = useState<File | null>(null);
	const [uploading, setUploading] = useState(false);

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
		setUploadError(null);
		setUploadMessage(null);
	};

	const handleEdit = (problem: Problem) => {
		setEditingId(problem.id);
		setForm({
			title: problem.title ?? "",
			slug: problem.slug ?? "",
			description: problem.description ?? "",
			difficulty: problem.difficulty ?? "",
			time_limit: problem.time_limit?.toString() ?? "",
			memory_limit: problem.memory_limit?.toString() ?? "",
			tags: problem.tags?.join(", ") ?? "",
		});
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
			slug: form.slug || undefined,
			description: form.description,
			difficulty: form.difficulty || undefined,
			time_limit: form.time_limit ? Number(form.time_limit) : undefined,
			memory_limit: form.memory_limit ? Number(form.memory_limit) : undefined,
			tags: parseTags(form.tags),
		};

		try {
			if (editingId !== null) {
				await api.put(`/problems/${editingId}`, payload, { headers: authHeaders });
			} else {
				await api.post("/problems", payload, { headers: authHeaders });
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

	const handleUploadTestcases = async () => {
		if (!hasToken) {
			setUploadError("You must be signed in to upload testcases.");
			return;
		}
		if (editingId === null) {
			setUploadError("Select a problem to edit before uploading testcases.");
			return;
		}
		if (!testcaseFile) {
			setUploadError("Pick a .zip file with testcases.");
			return;
		}
		setUploadError(null);
		setUploadMessage(null);
		setUploading(true);
		try {
			const data = new FormData();
			data.append("file", testcaseFile);
			await api.post(`/problems/${editingId}/testcases`, data, { headers: authHeaders });
			setUploadMessage("Testcases updated from zip.");
			setTestcaseFile(null);
		} catch (err) {
			setUploadError("Upload failed. Verify the zip and try again.");
		} finally {
			setUploading(false);
		}
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
							{loading && <span className="text-xs text-muted-foreground">Loading…</span>}
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
													{problem.slug && (
														<p className="text-xs text-muted-foreground">/{problem.slug}</p>
													)}
												</td>
												<td className="border border-border/70 px-3 py-2 text-sm">
													{problem.difficulty || "—"}
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
									<label className="text-sm font-semibold text-foreground">Slug</label>
									<input
										value={form.slug}
										onChange={(e) => setForm((prev) => ({ ...prev, slug: e.target.value }))}
										className="w-full border border-border/70 bg-background px-3 py-2 text-sm outline-none focus:border-primary focus:ring-2 focus:ring-primary/30 rounded-none"
										placeholder="unique-slug"
									/>
								</div>
								<div className="space-y-2">
									<label className="text-sm font-semibold text-foreground">Difficulty</label>
									<input
										value={form.difficulty}
										onChange={(e) =>
											setForm((prev) => ({ ...prev, difficulty: e.target.value }))
										}
										className="w-full border border-border/70 bg-background px-3 py-2 text-sm outline-none focus:border-primary focus:ring-2 focus:ring-primary/30 rounded-none"
										placeholder="Easy / Medium / Hard"
									/>
								</div>
							</div>

							<div className="grid gap-4 sm:grid-cols-2">
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
								<label className="text-sm font-semibold text-foreground">Testcases (.zip)</label>
								<input
									type="file"
									accept=".zip"
									onChange={(e) => {
										setTestcaseFile(e.target.files?.[0] ?? null);
										setUploadError(null);
										setUploadMessage(null);
									}}
									className="w-full border border-border/70 bg-background px-3 py-2 text-sm outline-none focus:border-primary focus:ring-2 focus:ring-primary/30 rounded-none file:mr-3 file:border-0 file:bg-muted file:px-3 file:py-2 file:text-sm file:font-semibold file:text-foreground file:rounded-none"
								/>
								<p className="text-xs text-muted-foreground">
									Attach a zip to replace testcases for the selected problem.
								</p>
								<div className="flex flex-wrap items-center gap-3">
									<Button
										type="button"
										variant="outline"
										className="rounded-none px-4 py-2"
										disabled={uploading || editingId === null}
										onClick={handleUploadTestcases}
									>
										{uploading ? "Uploading..." : "Upload testcases"}
									</Button>
									{testcaseFile && (
										<span className="text-xs text-muted-foreground">{testcaseFile.name}</span>
									)}
								</div>
								{uploadError && (
									<p className="border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive">
										{uploadError}
									</p>
								)}
								{uploadMessage && (
									<p className="border border-emerald-500/50 bg-emerald-500/10 px-3 py-2 text-sm text-emerald-700">
										{uploadMessage}
									</p>
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
