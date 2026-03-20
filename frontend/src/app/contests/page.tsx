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
		return { label: "UPCOMING", className: "text-sky-600 border-sky-500/40 bg-sky-500/5" };
	}
	if (now <= end) {
		return { label: "ACTIVE", className: "text-emerald-600 border-emerald-500/40 bg-emerald-500/5" };
	}
	return { label: "ENDED", className: "text-muted-foreground border-border/60" };
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
		<section className="mx-auto max-w-4xl px-6 py-10">
			{/* Header */}
			<div className="mb-8 flex items-baseline gap-4 border-b border-border/60 pb-5">
				<h1 className="font-display text-5xl text-foreground">CONTESTS</h1>
				<span className="text-xs font-mono text-muted-foreground tracking-widest">
					{contests.length} TOTAL
				</span>
			</div>

			{data === null ? (
				<p className="text-sm font-mono text-muted-foreground">
					Failed to load contests. Check that the API server is running.
				</p>
			) : contests.length === 0 ? (
				<div className="flex flex-col items-center gap-3 py-20 text-center">
					<span className="font-display text-4xl text-muted-foreground/30">NO CONTESTS</span>
					<p className="text-xs font-mono text-muted-foreground/50">No contests available yet.</p>
				</div>
			) : (
				<div className="flex flex-col divide-y divide-border/40">
					{contests.map((contest) => {
						const status = getStatus(contest.start_time, contest.end_time);
						return (
							<Link
								key={contest.id}
								href={`/contests/${contest.id}`}
								className="group grid grid-cols-[1fr_auto] items-center gap-4 py-4 px-1 hover:bg-muted/30 transition-colors"
							>
								{/* Left: title + meta */}
								<div className="flex flex-col gap-1.5 min-w-0">
									<span className="text-sm font-semibold text-foreground group-hover:text-primary transition-colors truncate">
										{contest.title}
									</span>
									<div className="flex flex-wrap items-center gap-3 text-[10px] font-mono text-muted-foreground/60 tracking-widest">
										<span>{contest.scoring_type.toUpperCase()}</span>
										<span>START {formatDate(contest.start_time)}</span>
										<span>END {formatDate(contest.end_time)}</span>
									</div>
								</div>

								{/* Right: status badge */}
								<span
									className={`text-[10px] font-mono tracking-widest border px-2.5 py-1 shrink-0 ${status.className}`}
								>
									{status.label}
								</span>
							</Link>
						);
					})}
				</div>
			)}
		</section>
	);
}
