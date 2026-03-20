"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { notFound } from "next/navigation";

import { api, ApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import "katex/dist/katex.min.css";
import ReactMarkdown from "react-markdown";
import rehypeKatex from "rehype-katex";
import remarkGfm from "remark-gfm";
import remarkMath from "remark-math";

import { SubmissionForm } from "./submission-form";

type Problem = {
	id: number;
	title?: string;
	difficulty?: string;
	tags?: string[];
	description?: string;
	time_limit?: number;
	memory_limit?: number;
	approval_status?: string;
	creator_id?: number;
};

export function ProblemContent({ id }: { id: string }) {
	const auth = useAuth();
	const [problem, setProblem] = useState<Problem | null>(null);
	const [notFoundError, setNotFoundError] = useState(false);

	useEffect(() => {
		const headers: Record<string, string> = {};
		if (auth.token) {
			headers["Authorization"] = `Bearer ${auth.token}`;
		}

		api.get<Problem>(`/problems/${id}`, { headers })
			.then(setProblem)
			.catch((err) => {
				if (err instanceof ApiError && err.status === 404) {
					setNotFoundError(true);
				}
			});
	}, [id, auth.token]);

	if (notFoundError) {
		notFound();
	}

	if (!problem) {
		return (
			<div className="mx-auto flex w-full max-w-5xl flex-col gap-6 px-4 py-12 sm:px-6">
				<p className="text-sm text-muted-foreground">Loading...</p>
			</div>
		);
	}

	const description = problem.description?.trim();
	const isApproved = problem.approval_status === "approved";

	return (
		<div className="mx-auto flex w-full max-w-5xl flex-col gap-6 px-4 py-12 sm:px-6">
			<div className="space-y-3">
				<h1 className="text-4xl font-bold tracking-tight sm:text-5xl">
					{problem.title ?? "Untitled problem"}
				</h1>
				{!isApproved && (
					<div className="border border-yellow-500/50 bg-yellow-500/10 px-3 py-2 text-sm text-yellow-700 dark:text-yellow-400">
						This problem is pending approval and not yet visible to the public.
					</div>
				)}
				<div className="flex flex-wrap items-center gap-3 text-sm text-muted-foreground">
					<span>Time Limit: {problem.time_limit} ms | Memory Limit: {problem.memory_limit ? Math.round(problem.memory_limit / 1048576) : 0} MB</span>
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

			{isApproved ? (
				<SubmissionForm problemId={problem.id} />
			) : (
				<div className="mt-12 bg-card/70 p-6 text-sm text-muted-foreground">
					Submissions are not available until this problem is approved.
				</div>
			)}
		</div>
	);
}
