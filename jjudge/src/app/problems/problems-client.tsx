"use client";

import Link from "next/link";
import { useState, useMemo } from "react";
import { Search, X } from "lucide-react";

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

const difficultyColor: Record<string, string> = {
	easy: "text-emerald-500 border-emerald-500/30 bg-emerald-500/5",
	medium: "text-amber-500 border-amber-500/30 bg-amber-500/5",
	hard: "text-rose-500 border-rose-500/30 bg-rose-500/5",
};

function getDifficultyStyle(difficulty?: string | unknown) {
	const key = difficulty != null ? String(difficulty).toLowerCase() : "";
	return difficultyColor[key] ?? "text-muted-foreground border-border/60";
}

export default function ProblemsClient({
	problems,
	difficulties,
	allTags,
}: {
	problems: Problem[];
	difficulties: string[];
	allTags: string[];
}) {
	const [search, setSearch] = useState("");
	const [activeDifficulty, setActiveDifficulty] = useState<string | null>(null);
	const [activeTags, setActiveTags] = useState<Set<string>>(new Set());

	const toggleTag = (tag: string) => {
		setActiveTags((prev) => {
			const next = new Set(prev);
			if (next.has(tag)) next.delete(tag);
			else next.add(tag);
			return next;
		});
	};

	const filtered = useMemo(() => {
		const q = search.trim().toLowerCase();
		return problems.filter((p) => {
			if (q && !p.title?.toLowerCase().includes(q)) return false;
			if (activeDifficulty && String(p.difficulty ?? "").toLowerCase() !== activeDifficulty.toLowerCase()) return false;
			if (activeTags.size > 0) {
				const pTags = new Set((p.tags ?? []).map((t) => String(t).toLowerCase()));
				for (const t of activeTags) {
					if (!pTags.has(t.toLowerCase())) return false;
				}
			}
			return true;
		});
	}, [problems, search, activeDifficulty, activeTags]);

	const hasFilters = search.trim() || activeDifficulty || activeTags.size > 0;

	const clearAll = () => {
		setSearch("");
		setActiveDifficulty(null);
		setActiveTags(new Set());
	};

	return (
		<div className="flex flex-col gap-5">

			{/* ── Search + filters ── */}
			<div className="flex flex-col gap-3">

				{/* Search input */}
				<div className="flex items-center border border-border/70 bg-card focus-within:border-primary/60 transition-colors">
					<span className="pl-4 pr-2 text-xs font-mono text-primary select-none">{">"}_</span>
					<input
						type="text"
						value={search}
						onChange={(e) => setSearch(e.target.value)}
						placeholder="search problems..."
						className="flex-1 bg-transparent py-2.5 pr-4 text-sm font-mono text-foreground placeholder:text-muted-foreground/50 outline-none"
					/>
					{search && (
						<button
							onClick={() => setSearch("")}
							className="pr-3 text-muted-foreground hover:text-foreground transition-colors"
						>
							<X className="h-3.5 w-3.5" />
						</button>
					)}
					<div className="border-l border-border/60 px-3 py-2.5">
						<Search className="h-3.5 w-3.5 text-muted-foreground" />
					</div>
				</div>

				{/* Difficulty pills */}
				<div className="flex flex-wrap items-center gap-2">
					<span className="text-[10px] font-mono text-muted-foreground/60 tracking-widest mr-1">DIFFICULTY</span>
					<button
						onClick={() => setActiveDifficulty(null)}
						className={`px-3 py-1 text-[10px] font-mono tracking-widest border transition-colors ${
							activeDifficulty === null
								? "border-primary text-primary bg-primary/10"
								: "border-border/60 text-muted-foreground hover:border-border hover:text-foreground"
						}`}
					>
						ALL
					</button>
					{difficulties.map((d) => (
						<button
							key={d}
							onClick={() => setActiveDifficulty(activeDifficulty === d ? null : d)}
							className={`px-3 py-1 text-[10px] font-mono tracking-widest border transition-colors ${
								activeDifficulty === d
									? getDifficultyStyle(d) + " !border-current"
									: "border-border/60 text-muted-foreground hover:border-border hover:text-foreground"
							}`}
						>
							{String(d).toUpperCase()}
						</button>
					))}
				</div>

				{/* Tag pills */}
				{allTags.length > 0 && (
					<div className="flex flex-wrap items-center gap-2">
						<span className="text-[10px] font-mono text-muted-foreground/60 tracking-widest mr-1">TAGS</span>
						{allTags.map((tag) => (
							<button
								key={tag}
								onClick={() => toggleTag(tag)}
								className={`px-2.5 py-1 text-[10px] font-mono border transition-colors ${
									activeTags.has(tag)
										? "border-primary text-primary bg-primary/10"
										: "border-border/60 text-muted-foreground hover:border-border hover:text-foreground"
								}`}
							>
								{tag}
							</button>
						))}
					</div>
				)}
			</div>

			{/* ── Results bar ── */}
			<div className="flex items-center justify-between border-b border-border/60 pb-2">
				<span className="text-[10px] font-mono text-muted-foreground tracking-widest">
					{filtered.length === problems.length
						? `${problems.length} PROBLEMS`
						: `${filtered.length} / ${problems.length} PROBLEMS`}
				</span>
				{hasFilters && (
					<button
						onClick={clearAll}
						className="flex items-center gap-1 text-[10px] font-mono text-muted-foreground hover:text-primary transition-colors tracking-widest"
					>
						<X className="h-3 w-3" /> CLEAR FILTERS
					</button>
				)}
			</div>

			{/* ── Problem table ── */}
			{filtered.length > 0 ? (
				<div className="flex flex-col divide-y divide-border/40">
					{filtered.map((problem, idx) => (
						<Link
							key={problem.id}
							href={`/problems/${problem.id}`}
							className="group grid grid-cols-[2.5rem_1fr_auto] items-center gap-4 py-3.5 px-1 hover:bg-muted/30 transition-colors"
						>
							{/* Index */}
							<span className="text-xs font-mono text-muted-foreground/40 text-right pr-2 group-hover:text-muted-foreground transition-colors">
								{String(idx + 1).padStart(3, "0")}
							</span>

							{/* Title + tags */}
							<div className="flex flex-col gap-1 min-w-0">
								<span className="text-sm font-semibold text-foreground group-hover:text-primary transition-colors truncate">
									{problem.title}
								</span>
								{problem.tags && problem.tags.length > 0 && (
									<div className="flex flex-wrap gap-1">
										{problem.tags.map((tag) => (
											<span
												key={tag}
												className="text-[10px] font-mono text-muted-foreground/60 border border-border/40 px-1.5 py-0.5"
											>
												{tag}
											</span>
										))}
									</div>
								)}
							</div>

							{/* Difficulty */}
							{problem.difficulty && (
								<span
									className={`text-[10px] font-mono tracking-widest border px-2.5 py-1 shrink-0 ${getDifficultyStyle(problem.difficulty)}`}
								>
									{String(problem.difficulty).toUpperCase()}
								</span>
							)}
						</Link>
					))}
				</div>
			) : (
				<div className="flex flex-col items-center gap-3 py-20 text-center">
					<span className="font-display text-4xl text-muted-foreground/30">NO RESULTS</span>
					<p className="text-xs font-mono text-muted-foreground/50">
						No problems match your filters.{" "}
						<button onClick={clearAll} className="text-primary hover:underline">
							Clear filters
						</button>
					</p>
				</div>
			)}
		</div>
	);
}
