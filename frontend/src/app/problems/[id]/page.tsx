import { api } from "@/lib/api";

import { ProblemContent } from "./problem-content";

type Problem = {
	id: number;
	title?: string;
	description?: string;
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
	return <ProblemContent id={id} />;
}
