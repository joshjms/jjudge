import Link from "next/link";
import { notFound } from "next/navigation";

import { api } from "@/lib/api";
import "katex/dist/katex.min.css";
import ReactMarkdown from "react-markdown";
import rehypeKatex from "rehype-katex";
import remarkGfm from "remark-gfm";
import remarkMath from "remark-math";

import { SubmissionForm } from "./submission-form";

type Problem = {
	id: number;
	title?: string;
	slug?: string;
	difficulty?: string;
	tags?: string[];
	description?: string;
	time_limit?: number;
	memory_limit?: number;
};

export const dynamic = "force-dynamic";

async function fetchProblem(id: string | number) {
	try {
		return await api.get<Problem>(`/problems/${id}`, { cache: "no-store" });
	} catch {
		return null;
	}
}

export async function generateMetadata({ params }: { params: Promise<{ id: string }> }) {
	const { id } = await params;
	const problem = await fetchProblem(id);
	if (!problem) return { title: "Problem not found" };

	return {
		title: problem.title ? `${problem.title} · Problem ${problem.id}` : `Problem ${problem.id}`,
		description: problem.description?.slice(0, 140),
	};
}

export default async function ProblemPage({ params }: { params: Promise<{ id: string }> }) {
	const { id } = await params;
	const problem = await fetchProblem(id);

	if (!problem) {
		notFound();
	}
	const description = problem.description?.trim();

	return (
		<div className="mx-auto flex w-full max-w-5xl flex-col gap-6 px-4 py-12 sm:px-6">
			<div className="space-y-3">
				<h1 className="text-4xl font-bold tracking-tight sm:text-5xl">
					{problem.title ?? "Untitled problem"}
				</h1>
				<div className="flex flex-wrap items-center gap-3 text-sm text-muted-foreground">
					<span>Time Limit: {problem.time_limit} ms | Memory Limit: {problem.memory_limit} MB</span>
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

            <div className="flex flex-wrap items-center gap-3">
                <Link
                    href={`/problems/${problem.id}/submissions`}
                    className="border border-border/70 px-3 py-2 text-sm font-semibold text-foreground transition hover:border-primary/60 hover:bg-muted/60"
                >
                    All submissions
                </Link>
                <Link
                    href={`/problems/${problem.id}/submissions/mine`}
                    className="bg-primary px-3 py-2 text-sm font-semibold text-primary-foreground transition hover:bg-primary/90"
                >
                    My submissions
                </Link>
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

			<SubmissionForm problemId={problem.id} />
		</div>
	);
}
