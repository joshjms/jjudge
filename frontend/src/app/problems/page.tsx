import { api } from "@/lib/api";
import ProblemsClient from "./problems-client";

type Problem = {
	id?: string;
	title?: string;
	slug?: string;
	difficulty?: string;
	tags?: string[];
};

const difficultyOrder: Record<string, number> = {
	easy: 0,
	medium: 1,
	hard: 2,
};

const fetchProblems = async (): Promise<Problem[] | null> => {
	try {
		const payload = await api.get<unknown>("/problems", { cache: "no-store" });
		if (Array.isArray(payload)) return payload as Problem[];
		if (payload && typeof payload === "object") {
			const wrapped = payload as { problems?: Problem[]; data?: Problem[]; items?: Problem[] };
			if (Array.isArray(wrapped.problems)) return wrapped.problems;
			if (Array.isArray(wrapped.data)) return wrapped.data;
			if (Array.isArray(wrapped.items)) return wrapped.items;
		}
		return [];
	} catch {
		return null;
	}
};

export default async function ProblemsPage() {
	const problems = await fetchProblems();

	if (!Array.isArray(problems)) {
		return (
			<section className="mx-auto max-w-4xl px-6 py-12">
				<p className="text-sm font-mono text-muted-foreground">
					Failed to load problems. Check that the API server is running.
				</p>
			</section>
		);
	}

	// Derive filter options from data — coerce to string so .toUpperCase() is always safe
	const difficulties = Array.from(
		new Set(
			problems
				.map((p) => (p.difficulty != null ? String(p.difficulty) : undefined))
				.filter(Boolean) as string[]
		)
	).sort((a, b) => (difficultyOrder[a.toLowerCase()] ?? 99) - (difficultyOrder[b.toLowerCase()] ?? 99));

	const allTags = Array.from(
		new Set(problems.flatMap((p) => (p.tags ?? []).map(String)))
	).sort();

	return (
		<section className="mx-auto max-w-4xl px-6 py-10">
			{/* Header */}
			<div className="mb-8 flex items-baseline gap-4 border-b border-border/60 pb-5">
				<h1 className="font-display text-5xl text-foreground">PROBLEMS</h1>
				<span className="text-xs font-mono text-muted-foreground tracking-widest">
					{problems.length} TOTAL
				</span>
			</div>

			<ProblemsClient
				problems={problems}
				difficulties={difficulties}
				allTags={allTags}
			/>
		</section>
	);
}
