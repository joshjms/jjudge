import Link from "next/link";

import { api } from "@/lib/api";
import {
	Clock9,
} from "lucide-react";

type Problem = {
	id?: string;
	title?: string;
	slug?: string;
	difficulty?: string;
	tags?: string[];
};

const fetchProblems = async () => {
	try {
		return await api.get<Problem[]>("/problems", { cache: "no-store" });
	} catch {
		return null;
	}
};

export default async function Home() {
	const problems = await fetchProblems();

	return (
		<>
		<section className="px-24 py-10 md:py-16 lg:py-24 mx-auto max-w-6xl">
			{
				problems ? (
					<div className="grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-3">
						{problems.map((problem) => (
							<Link
								key={problem.id}
								href={`/problems/${problem.id}`}
								className="group flex flex-col border border-border/70 p-6 transition hover:border-primary/60 hover:bg-muted/50"
							>
								<h2 className="mb-2 text-lg font-semibold transition-colors group-hover:text-primary">
									{problem.title}
								</h2>
								<p className="mt-auto flex items-center text-sm text-muted-foreground">
									<Clock9 className="mr-2 h-4 w-4" />
									Difficulty: {problem.difficulty || "Unknown"}
								</p>
							</Link>
						))}
					</div>
				) : (
					<p>Failed to load problems.</p>
				)
			}
		</section>
		</>
	);
}
