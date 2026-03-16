import Link from "next/link";

import { api } from "@/lib/api";

type Contest = {
	id: number;
	title: string;
	description?: string;
	start_time: string;
	end_time: string;
	scoring_type: "icpc" | "ioi";
	visibility: string;
};

type ContestListResponse = {
	items: Contest[];
	page: number;
	limit: number;
	total: number;
};

function getStatus(startTime: string, endTime: string): { label: string; className: string } {
	const now = Date.now();
	const start = new Date(startTime).getTime();
	const end = new Date(endTime).getTime();

	if (now < start) {
		return {
			label: "Upcoming",
			className: "border-sky-500/40 bg-sky-500/10 text-sky-700",
		};
	}
	if (now <= end) {
		return {
			label: "Active",
			className: "border-emerald-500/40 bg-emerald-500/10 text-emerald-700",
		};
	}
	return {
		label: "Ended",
		className: "border-border/70 bg-muted/50 text-muted-foreground",
	};
}

const formatDate = (value: string) =>
	new Intl.DateTimeFormat(undefined, {
		year: "numeric",
		month: "short",
		day: "2-digit",
		hour: "2-digit",
		minute: "2-digit",
	}).format(new Date(value));

export const dynamic = "force-dynamic";

async function fetchContests(): Promise<ContestListResponse | null> {
	try {
		return await api.get<ContestListResponse>("/contests?limit=50", { cache: "no-store" });
	} catch {
		return null;
	}
}

export default async function ContestsPage() {
	const data = await fetchContests();
	const contests = data?.items ?? [];

	return (
		<div className="mx-auto flex w-full max-w-6xl flex-col gap-8 px-4 py-12 sm:px-6">
			<div className="space-y-2">
				<p className="text-xs font-semibold uppercase tracking-[0.25em] text-primary">Contests</p>
				<h1 className="text-3xl font-bold leading-tight sm:text-4xl">All contests</h1>
				<p className="text-sm text-muted-foreground">
					Timed competitive events grouping multiple problems.
				</p>
			</div>

			{data === null ? (
				<div className="border border-border/70 px-6 py-10 text-center text-sm text-destructive">
					Failed to load contests.
				</div>
			) : contests.length === 0 ? (
				<div className="border border-border/70 px-6 py-10 text-center text-sm text-muted-foreground">
					No contests yet.
				</div>
			) : (
				<div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
					{contests.map((contest) => {
						const status = getStatus(contest.start_time, contest.end_time);
						return (
							<Link
								key={contest.id}
								href={`/contests/${contest.id}`}
								className="group flex flex-col gap-3 border border-border/70 p-5 transition hover:border-primary/60 hover:bg-muted/50"
							>
								<div className="flex items-start justify-between gap-2">
									<h2 className="text-base font-semibold transition-colors group-hover:text-primary">
										{contest.title}
									</h2>
									<span
										className={`shrink-0 border px-2 py-0.5 text-[11px] font-semibold uppercase tracking-wide ${status.className}`}
									>
										{status.label}
									</span>
								</div>

								<div className="flex flex-wrap gap-2">
									<span className="border border-border/60 px-2 py-0.5 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
										{contest.scoring_type === "icpc" ? "ICPC" : "IOI"}
									</span>
								</div>

								<div className="mt-auto space-y-1 text-xs text-muted-foreground">
									<p>Start: {formatDate(contest.start_time)}</p>
									<p>End: {formatDate(contest.end_time)}</p>
								</div>
							</Link>
						);
					})}
				</div>
			)}
		</div>
	);
}
