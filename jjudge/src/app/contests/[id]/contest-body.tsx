"use client";

import Link from "next/link";
import { useEffect, useState } from "react";

import { RegisterButton } from "./register-button";

// ─── Types ───────────────────────────────────────────────────────────────────

export type ContestProblem = {
	contest_id: number;
	problem_id: number;
	ordinal: number;
	max_points: number;
	problem?: { id: number; title?: string };
};

export type Contest = {
	id: number;
	title: string;
	description?: string;
	start_time: string;
	end_time: string;
	scoring_type: "icpc" | "ioi";
	visibility: string;
	problems?: ContestProblem[];
};

type ContestState = "upcoming" | "active" | "ended";

// ─── Helpers ─────────────────────────────────────────────────────────────────

function getState(startMs: number, endMs: number, now: number): ContestState {
	if (now < startMs) return "upcoming";
	if (now <= endMs) return "active";
	return "ended";
}

function pad(n: number) {
	return String(n).padStart(2, "0");
}

function toParts(ms: number) {
	const total = Math.max(0, Math.floor(ms / 1000));
	return {
		h: pad(Math.floor(total / 3600)),
		m: pad(Math.floor((total % 3600) / 60)),
		s: pad(total % 60),
	};
}

function ordinalLabel(n: number): string {
	if (n < 26) return String.fromCharCode(65 + n);
	return (
		String.fromCharCode(65 + Math.floor(n / 26) - 1) +
		String.fromCharCode(65 + (n % 26))
	);
}

const fmtDate = (v: string) =>
	new Intl.DateTimeFormat(undefined, {
		year: "numeric",
		month: "short",
		day: "2-digit",
		hour: "2-digit",
		minute: "2-digit",
	}).format(new Date(v));

function durationLabel(startMs: number, endMs: number): string {
	const ms = endMs - startMs;
	const totalMinutes = Math.round(ms / 60000);
	const h = Math.floor(totalMinutes / 60);
	const m = totalMinutes % 60;
	if (h === 0) return `${m}m`;
	if (m === 0) return `${h}h`;
	return `${h}h ${m}m`;
}

// ─── Sub-components ───────────────────────────────────────────────────────────

function TimeUnit({ value, label }: { value: string; label: string }) {
	return (
		<div className="flex flex-col items-center gap-2">
			<div className="flex h-16 w-20 items-center justify-center border border-border/70 bg-muted/30 font-mono text-4xl font-bold tabular-nums sm:h-20 sm:w-24 sm:text-5xl">
				{value}
			</div>
			<span className="text-[10px] font-semibold uppercase tracking-[0.2em] text-muted-foreground">
				{label}
			</span>
		</div>
	);
}

function HeroCountdown({ remaining }: { remaining: number }) {
	const { h, m, s } = toParts(remaining);
	return (
		<div className="border border-border/70 bg-muted/20 px-8 py-8">
			<p className="mb-6 text-xs font-semibold uppercase tracking-[0.25em] text-muted-foreground">
				Contest starts in
			</p>
			<div className="flex flex-wrap items-end gap-2">
				<TimeUnit value={h} label="Hours" />
				<span className="mb-[26px] text-3xl font-bold text-border">:</span>
				<TimeUnit value={m} label="Minutes" />
				<span className="mb-[26px] text-3xl font-bold text-border">:</span>
				<TimeUnit value={s} label="Seconds" />
			</div>
		</div>
	);
}

function InlineCountdown({
	label,
	remaining,
	colorClass,
}: {
	label: string;
	remaining: number;
	colorClass: string;
}) {
	const { h, m, s } = toParts(remaining);
	return (
		<div className={`flex items-center gap-2 text-sm ${colorClass}`}>
			<span className="font-semibold">{label}</span>
			<span className="font-mono font-bold tabular-nums">
				{h}:{m}:{s}
			</span>
		</div>
	);
}

function ContestMeta({
	contest,
}: {
	contest: Contest;
}) {
	const startMs = new Date(contest.start_time).getTime();
	const endMs = new Date(contest.end_time).getTime();

	return (
		<dl className="grid grid-cols-2 gap-x-6 gap-y-3 border border-border/70 bg-muted/20 px-6 py-4 text-sm sm:grid-cols-4">
			<div>
				<dt className="text-[10px] font-semibold uppercase tracking-[0.15em] text-muted-foreground">
					Start
				</dt>
				<dd className="mt-0.5 font-medium">{fmtDate(contest.start_time)}</dd>
			</div>
			<div>
				<dt className="text-[10px] font-semibold uppercase tracking-[0.15em] text-muted-foreground">
					End
				</dt>
				<dd className="mt-0.5 font-medium">{fmtDate(contest.end_time)}</dd>
			</div>
			<div>
				<dt className="text-[10px] font-semibold uppercase tracking-[0.15em] text-muted-foreground">
					Duration
				</dt>
				<dd className="mt-0.5 font-medium">{durationLabel(startMs, endMs)}</dd>
			</div>
			<div>
				<dt className="text-[10px] font-semibold uppercase tracking-[0.15em] text-muted-foreground">
					Scoring
				</dt>
				<dd className="mt-0.5 font-medium uppercase">
					{contest.scoring_type}
				</dd>
			</div>
		</dl>
	);
}

function ProblemsTable({
	contest,
	problems,
}: {
	contest: Contest;
	problems: ContestProblem[];
}) {
	if (problems.length === 0) {
		return (
			<p className="text-sm text-muted-foreground">No problems added yet.</p>
		);
	}

	return (
		<div className="overflow-hidden border border-border/70">
			<table className="min-w-full divide-y divide-border/70 text-sm">
				<thead className="bg-muted/70 text-xs uppercase tracking-wide text-muted-foreground">
					<tr>
						<th className="w-12 px-4 py-3 text-left font-semibold">#</th>
						<th className="px-4 py-3 text-left font-semibold">Problem</th>
						<th className="px-4 py-3 text-right font-semibold">Points</th>
					</tr>
				</thead>
				<tbody className="divide-y divide-border/70">
					{problems.map((cp) => (
						<tr key={cp.problem_id} className="hover:bg-muted/40">
							<td className="px-4 py-3 font-bold text-muted-foreground">
								{ordinalLabel(cp.ordinal)}
							</td>
							<td className="px-4 py-3">
								<Link
									href={`/contests/${contest.id}/problems/${cp.problem_id}`}
									className="font-semibold text-foreground transition hover:text-primary"
								>
									{cp.problem?.title ?? `Problem ${cp.problem_id}`}
								</Link>
							</td>
							<td className="px-4 py-3 text-right text-muted-foreground">
								{cp.max_points}
							</td>
						</tr>
					))}
				</tbody>
			</table>
		</div>
	);
}

// ─── Nav links (leaderboard + submissions) ────────────────────────────────────

function NavLinks({ contest }: { contest: Contest }) {
	return (
		<div className="flex flex-wrap gap-2">
			<Link
				href={`/contests/${contest.id}/leaderboard`}
				className="border border-border/70 px-3 py-2 text-sm font-semibold text-foreground transition hover:border-primary/60 hover:bg-muted/60"
			>
				Leaderboard
			</Link>
			<Link
				href={`/contests/${contest.id}/submissions`}
				className="border border-border/70 px-3 py-2 text-sm font-semibold text-foreground transition hover:border-primary/60 hover:bg-muted/60"
			>
				Submissions
			</Link>
		</div>
	);
}

// ─── State-specific layouts ───────────────────────────────────────────────────

function UpcomingLayout({
	contest,
	sortedProblems,
	now,
}: {
	contest: Contest;
	sortedProblems: ContestProblem[];
	now: number;
}) {
	const startMs = new Date(contest.start_time).getTime();

	return (
		<div className="flex flex-col gap-8">
			{/* Header */}
			<div className="space-y-3">
				<div className="flex flex-wrap items-center gap-2">
					<span className="border border-sky-500/40 bg-sky-500/10 px-2 py-0.5 text-[11px] font-semibold uppercase tracking-wide text-sky-700">
						Upcoming
					</span>
					<span className="border border-border/60 px-2 py-0.5 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
						{contest.scoring_type}
					</span>
				</div>
				<h1 className="text-3xl font-bold leading-tight sm:text-4xl">
					{contest.title}
				</h1>
				{contest.description && (
					<p className="text-sm leading-relaxed text-muted-foreground">
						{contest.description}
					</p>
				)}
			</div>

			{/* Hero countdown */}
			<HeroCountdown remaining={startMs - now} />

			{/* Registration */}
			<div className="flex flex-col gap-3">
				<p className="text-sm text-muted-foreground">
					Register now to participate when the contest begins.
				</p>
				<RegisterButton contestId={contest.id} />
			</div>

			{/* Contest meta */}
			<ContestMeta contest={contest} />

			{/* Problems locked */}
			<div>
				<h2 className="mb-3 text-lg font-semibold">Problems</h2>
				<div className="flex items-center gap-3 border border-border/70 bg-muted/20 px-6 py-8 text-sm text-muted-foreground">
					<svg
						xmlns="http://www.w3.org/2000/svg"
						width="16"
						height="16"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						strokeWidth="2"
						strokeLinecap="round"
						strokeLinejoin="round"
						className="shrink-0"
					>
						<rect width="18" height="11" x="3" y="11" rx="2" ry="2" />
						<path d="M7 11V7a5 5 0 0 1 10 0v4" />
					</svg>
					<span>
						{sortedProblems.length > 0
							? `${sortedProblems.length} problem${sortedProblems.length === 1 ? "" : "s"} — revealed when the contest begins.`
							: "Problems will be revealed when the contest begins."}
					</span>
				</div>
			</div>
		</div>
	);
}

function ActiveLayout({
	contest,
	sortedProblems,
	now,
}: {
	contest: Contest;
	sortedProblems: ContestProblem[];
	now: number;
}) {
	const endMs = new Date(contest.end_time).getTime();

	return (
		<div className="flex flex-col gap-8">
			{/* Header */}
			<div className="space-y-3">
				<div className="flex flex-wrap items-center gap-2">
					<span className="flex items-center gap-1.5 border border-emerald-500/40 bg-emerald-500/10 px-2 py-0.5 text-[11px] font-semibold uppercase tracking-wide text-emerald-700">
						<span className="inline-block h-1.5 w-1.5 rounded-full bg-emerald-500" />
						Active
					</span>
					<span className="border border-border/60 px-2 py-0.5 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
						{contest.scoring_type}
					</span>
				</div>
				<h1 className="text-3xl font-bold leading-tight sm:text-4xl">
					{contest.title}
				</h1>
				<InlineCountdown
					label="Time remaining"
					remaining={endMs - now}
					colorClass="text-emerald-700"
				/>
				{contest.description && (
					<p className="text-sm leading-relaxed text-muted-foreground">
						{contest.description}
					</p>
				)}
			</div>

			{/* Actions */}
			<div className="flex flex-wrap items-center gap-4">
				<RegisterButton contestId={contest.id} />
				<div className="h-4 w-px bg-border/60" />
				<NavLinks contest={contest} />
			</div>

			{/* Problems */}
			<div>
				<h2 className="mb-3 text-lg font-semibold">Problems</h2>
				<ProblemsTable contest={contest} problems={sortedProblems} />
			</div>
		</div>
	);
}

function EndedLayout({
	contest,
	sortedProblems,
}: {
	contest: Contest;
	sortedProblems: ContestProblem[];
}) {
	return (
		<div className="flex flex-col gap-8">
			{/* Header */}
			<div className="space-y-3">
				<div className="flex flex-wrap items-center gap-2">
					<span className="border border-border/60 bg-muted/50 px-2 py-0.5 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
						Ended
					</span>
					<span className="border border-border/60 px-2 py-0.5 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
						{contest.scoring_type}
					</span>
				</div>
				<h1 className="text-3xl font-bold leading-tight sm:text-4xl">
					{contest.title}
				</h1>
				<p className="text-sm text-muted-foreground">
					{fmtDate(contest.start_time)} → {fmtDate(contest.end_time)}
				</p>
				{contest.description && (
					<p className="text-sm leading-relaxed text-muted-foreground">
						{contest.description}
					</p>
				)}
			</div>

			{/* Nav */}
			<NavLinks contest={contest} />

			{/* Problems */}
			<div>
				<h2 className="mb-3 text-lg font-semibold">Problems</h2>
				<ProblemsTable contest={contest} problems={sortedProblems} />
			</div>
		</div>
	);
}

// ─── Main export ──────────────────────────────────────────────────────────────

export function ContestBody({ contest }: { contest: Contest }) {
	const [now, setNow] = useState(() => Date.now());

	useEffect(() => {
		const timer = setInterval(() => setNow(Date.now()), 1000);
		return () => clearInterval(timer);
	}, []);

	const startMs = new Date(contest.start_time).getTime();
	const endMs = new Date(contest.end_time).getTime();
	const state = getState(startMs, endMs, now);

	const sortedProblems = (contest.problems ?? [])
		.slice()
		.sort((a, b) => a.ordinal - b.ordinal);

	return (
		<div className="mx-auto w-full max-w-5xl px-4 py-12 sm:px-6">
			{state === "upcoming" && (
				<UpcomingLayout
					contest={contest}
					sortedProblems={sortedProblems}
					now={now}
				/>
			)}
			{state === "active" && (
				<ActiveLayout
					contest={contest}
					sortedProblems={sortedProblems}
					now={now}
				/>
			)}
			{state === "ended" && (
				<EndedLayout contest={contest} sortedProblems={sortedProblems} />
			)}
		</div>
	);
}
