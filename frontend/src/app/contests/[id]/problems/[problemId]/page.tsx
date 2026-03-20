import { notFound } from "next/navigation";
import ReactMarkdown from "react-markdown";
import rehypeKatex from "rehype-katex";
import remarkGfm from "remark-gfm";
import remarkMath from "remark-math";
import "katex/dist/katex.min.css";

import { api } from "@/lib/api";

import { ContestSubmissionForm } from "./contest-submission-form";

type Problem = {
	id: number;
	title?: string;
	description?: string;
	difficulty?: number;
	time_limit?: number;
	memory_limit?: number;
	tags?: string[];
};

export const dynamic = "force-dynamic";

async function fetchProblem(id: string): Promise<Problem | null> {
	try {
		return await api.get<Problem>(`/problems/${id}`, { cache: "no-store" });
	} catch {
		return null;
	}
}

export async function generateMetadata({
	params,
}: {
	params: Promise<{ id: string; problemId: string }>;
}) {
	const { problemId } = await params;
	const problem = await fetchProblem(problemId);
	if (!problem) return { title: "Problem not found" };
	return {
		title: problem.title ? `${problem.title} · Problem ${problem.id}` : `Problem ${problem.id}`,
	};
}

export default async function ContestProblemPage({
	params,
}: {
	params: Promise<{ id: string; problemId: string }>;
}) {
	const { id: contestId, problemId } = await params;
	const problem = await fetchProblem(problemId);

	if (!problem) notFound();

	const description = problem.description?.trim();

	return (
		<div className="mx-auto flex w-full max-w-5xl flex-col gap-6 px-4 py-12 sm:px-6">
			<div className="space-y-3">
				<h1 className="text-4xl font-bold tracking-tight sm:text-5xl">
					{problem.title ?? "Untitled problem"}
				</h1>
				<div className="flex flex-wrap items-center gap-3 text-sm text-muted-foreground">
					<span>
						Time Limit: {problem.time_limit} ms | Memory Limit: {problem.memory_limit ? Math.round(problem.memory_limit / 1048576) : 0} MB
					</span>
				</div>
				<div className="flex flex-wrap items-center gap-3 text-sm text-muted-foreground">
					{problem.difficulty && (
						<span className="border border-border/70 px-3 py-1 text-xs font-semibold tracking-wide">
							{problem.difficulty}
						</span>
					)}
					{problem.tags?.map((tag) => (
						<span
							key={tag}
							className="border border-border/80 px-3 py-1 text-[11px] uppercase tracking-wide"
						>
							{tag}
						</span>
					))}
				</div>
			</div>

			<div>
				{description ? (
					<div className="markdown-content mt-3">
						<ReactMarkdown
							remarkPlugins={[remarkGfm, remarkMath]}
							rehypePlugins={[rehypeKatex]}
							skipHtml
						>
							{description}
						</ReactMarkdown>
					</div>
				) : (
					<p className="mt-3 text-sm leading-relaxed text-muted-foreground">
						No statement available yet.
					</p>
				)}
			</div>

			<ContestSubmissionForm
				contestId={Number(contestId)}
				problemId={problem.id}
			/>
		</div>
	);
}
