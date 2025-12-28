import Link from "next/link";
import { notFound } from "next/navigation";
import { createElement, type ReactNode } from "react";
import type { RootContent } from "hast";

import { createStarryNight, common } from "@wooorm/starry-night";
import { api } from "@/lib/api";

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
	code?: string;
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

async function fetchProblem(id: string | number) {
	try {
		return await api.get<Problem>(`/problems/${id}`, { cache: "no-store" });
	} catch {
		return null;
	}
}

async function fetchSubmission(id: string | number) {
	try {
		return await api.get<Submission>(`/submissions/${id}`, { cache: "no-store" });
	} catch {
		return null;
	}
}

const languageScopes: Record<string, string> = {
	cpp: "source.cpp",
	cpp20: "source.cpp",
	c: "source.c",
	python: "source.python",
	py: "source.python",
	javascript: "source.js",
	js: "source.js",
	typescript: "source.ts",
	ts: "source.ts",
	go: "source.go",
	rust: "source.rust",
};

type HastNode = RootContent;

const renderNode = (node: HastNode | null | undefined, key: number): ReactNode => {
	if (!node) return null;
	if (node.type === "text") return node.value;
	if (node.type === "element") {
		const Tag = node.tagName || "span";
		const className = Array.isArray(node.properties?.className)
			? node.properties.className.join(" ")
			: node.properties?.className;
		const children = node.children?.map((child, index) => renderNode(child, index));
		return createElement(Tag, { key, className }, children);
	}
	return null;
};

export async function generateMetadata({ params }: { params: Promise<{ submissionId: string }> }) {
	const { submissionId } = await params;
	const submission = await fetchSubmission(submissionId);
	const title = submission
		? `Submission #${submission.id} · Problem ${submission.problem_id}`
		: `Submission #${submissionId}`;

	return {
		title,
		description: `Submission ${submissionId}`,
	};
}

export default async function SubmissionDetailsPage({
	params,
}: {
	params: Promise<{ submissionId: string }>;
}) {
	const { submissionId } = await params;
	const submission = await fetchSubmission(submissionId);

	if (!submission) {
		notFound();
	}

	const problem = await fetchProblem(submission.problem_id);
	const verdict = submission.verdict?.toUpperCase?.() ?? "PENDING";
	const verdictClass = verdictStyles[verdict] ?? "border-border/70 bg-muted/50 text-foreground";

	let highlightedCode: ReactNode = null;
	if (submission.code) {
		const starryNight = await createStarryNight(common);
		const normalizedLang = submission.language?.toLowerCase?.() ?? "";
		const scopeId =
			languageScopes[normalizedLang] ?? (normalizedLang ? `source.${normalizedLang}` : "text.plain");
		const scope = starryNight.flagToScope(scopeId) ?? starryNight.flagToScope("text.plain");

		if (scope) {
			const tree = starryNight.highlight(submission.code, scope);
			highlightedCode = tree.children?.map((node, index) => renderNode(node, index));
		} else {
			highlightedCode = submission.code;
		}
	}

	return (
		<div className="mx-auto flex w-full max-w-5xl flex-col gap-8 px-4 py-12 sm:px-6">
			<div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
				<div className="space-y-1">
					<p className="text-xs font-semibold uppercase tracking-[0.25em] text-primary">
						Submission #{submission.id}
					</p>
					<h1 className="text-3xl font-bold leading-tight sm:text-4xl">
						{problem?.title ?? "Untitled problem"}
					</h1>
					<p className="text-sm text-muted-foreground">
						Problem {problem?.id ?? submission.problem_id}
					</p>
				</div>
				<div className="flex flex-wrap items-center gap-2">
					{problem && (
						<Link
							href={`/problems/${problem.id}`}
							className="border border-border/70 px-3 py-2 text-sm font-semibold text-foreground transition hover:border-primary/60 hover:bg-muted/60"
						>
							View problem
						</Link>
					)}
					{problem && (
						<Link
							href={`/problems/${problem.id}/submissions`}
							className="border border-border/70 px-3 py-2 text-sm font-semibold text-foreground transition hover:border-primary/60 hover:bg-muted/60"
						>
							All submissions
						</Link>
					)}
				</div>
			</div>

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
						<span className={`inline-flex border px-3 py-1 text-xs font-semibold uppercase ${verdictClass}`}>
							{verdict}
						</span>
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

			<section className="border border-border/70 bg-background/70 px-6 py-6">
				<div className="mb-3 flex items-center justify-between">
					<h2 className="text-xl font-semibold">Source code</h2>
					<span className="text-xs text-muted-foreground">
						{submission.language?.toUpperCase?.() ?? "—"}
					</span>
				</div>
				{submission.code ? (
					<pre className="starry-code overflow-x-auto border border-border/70 bg-muted/50 px-4 py-4 text-sm text-foreground">
						<code className="block whitespace-pre">{highlightedCode}</code>
					</pre>
				) : (
					<p className="text-sm text-muted-foreground">Source code not available.</p>
				)}
			</section>
		</div>
	);
}
