import Link from "next/link";
import { notFound } from "next/navigation";

import { api } from "@/lib/api";
import { SubmissionDetail } from "@/components/submission-detail";

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
	testcase_results?: {
		testcase_id: number;
		verdict: string;
		cpu_time: number;
		memory: number;
		input?: string;
		expected_output?: string;
		actual_output?: string;
		error_message?: string;
	}[];
};

type Problem = {
	id: number;
	title?: string;
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

export async function generateMetadata({ params }: { params: Promise<{ id: string; submissionId: string }> }) {
	const { id, submissionId } = await params;
	const submission = await fetchSubmission(submissionId);
	const title = submission
		? `Submission #${submission.id} · Problem ${submission.problem_id}`
		: `Submission #${submissionId}`;

	return {
		title,
		description: `Submission ${submissionId} for problem ${id}`,
	};
}

export default async function SubmissionDetailsPage({
	params,
}: {
	params: Promise<{ id: string; submissionId: string }>;
}) {
	const { id, submissionId } = await params;
	const [problem, submission] = await Promise.all([fetchProblem(id), fetchSubmission(submissionId)]);

	if (!problem || !submission) {
		notFound();
	}

	return (
		<div className="mx-auto flex w-full max-w-5xl flex-col gap-8 px-4 py-12 sm:px-6">
			<div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
				<div className="space-y-1">
					<p className="text-xs font-semibold uppercase tracking-[0.25em] text-primary">
						Submission #{submission.id}
					</p>
					<h1 className="text-3xl font-bold leading-tight sm:text-4xl">
						{problem.title ?? "Untitled problem"}
					</h1>
					<p className="text-sm text-muted-foreground">Problem {problem.id}</p>
				</div>
				<div className="flex flex-wrap items-center gap-2">
					<Link
						href={`/problems/${problem.id}`}
						className="border border-border/70 px-3 py-2 text-sm font-semibold text-foreground transition hover:border-primary/60 hover:bg-muted/60"
					>
						View problem
					</Link>
					<Link
						href={`/problems/${problem.id}/submissions`}
						className="border border-border/70 px-3 py-2 text-sm font-semibold text-foreground transition hover:border-primary/60 hover:bg-muted/60"
					>
						All submissions
					</Link>
				</div>
			</div>

			<SubmissionDetail initialSubmission={submission} />

			<section className="border border-border/70 bg-background/70 px-6 py-6">
				<div className="mb-3 flex items-center justify-between">
					<h2 className="text-xl font-semibold">Source code</h2>
					<span className="text-xs text-muted-foreground">
						{submission.language?.toUpperCase?.() ?? "—"}
					</span>
				</div>
				{submission.code ? (
					<pre className="overflow-x-auto border border-border/70 bg-muted/50 px-4 py-4 text-sm text-foreground">
						<code>{submission.code}</code>
					</pre>
				) : (
					<p className="text-sm text-muted-foreground">Source code not available.</p>
				)}
			</section>
		</div>
	);
}
