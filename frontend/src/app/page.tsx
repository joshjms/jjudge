import Link from "next/link";

export default function Home() {
	return (
		<div className="flex flex-col min-h-[calc(100vh-4rem)]">

			{/* ── Hero ── */}
			<section className="mx-auto max-w-6xl w-full px-6 pt-16 pb-8 md:pt-28 md:pb-12 flex-1 flex flex-col justify-center">
				<div className="grid grid-cols-1 lg:grid-cols-12 gap-10 lg:gap-16 items-center">

					{/* Left */}
					<div className="lg:col-span-7 xl:col-span-8 flex flex-col gap-6">
						<h1
							className="font-display leading-[0.92] tracking-wide text-foreground animate-in-2"
							style={{ fontSize: "clamp(4rem, 12vw, 9rem)" }}
						>
							JJUDGE
						</h1>

						<p className="max-w-sm text-base text-muted-foreground leading-relaxed animate-in-3 font-mono">
							An online judge for competitive programming.
							Submit code, receive instant verdicts, track your progress.
						</p>

						<div className="flex flex-wrap gap-3 animate-in-4">
							<Link
								href="/problems"
								className="inline-flex items-center gap-2 px-6 py-3 bg-primary text-primary-foreground text-sm font-mono font-semibold tracking-wide transition-all hover:opacity-90"
							>
								Browse problems →
							</Link>
							<Link
								href="/register"
								className="inline-flex items-center px-6 py-3 border border-border text-foreground text-sm font-mono tracking-wide transition-all hover:border-primary/60 hover:text-primary"
							>
								Create account
							</Link>
						</div>
					</div>

					{/* Right: Status panel */}
					<div className="lg:col-span-5 xl:col-span-4 animate-in-4">
						<div className="border border-border/70 bg-card">
							<div className="flex items-center gap-2 px-5 py-3 border-b border-border/60 bg-muted/30">
								<span className="font-mono text-[10px] tracking-[0.25em] uppercase text-muted-foreground">
									// status
								</span>
								<span className="ml-auto w-1.5 h-1.5 rounded-full bg-primary" style={{ animation: "blink 2s step-end infinite" }} />
							</div>

							<div className="px-5 py-4 flex flex-col gap-3 font-mono text-xs">
								<Row label="Judge" value="Online" accent />
								<Row label="Execution" value="Sandboxed" />
								<Row label="Scoring" value="ICPC · IOI" />
								<Row label="Contests" value="Open" />
							</div>

							<div className="px-5 py-3 border-t border-border/60 bg-muted/20">
								<p className="font-mono text-[11px] text-muted-foreground leading-relaxed">
									Code runs in isolated containers with strict CPU and memory limits.
								</p>
							</div>
						</div>
					</div>
				</div>
			</section>

			{/* ── Features ── */}
			<section className="mx-auto max-w-6xl w-full px-6 pb-16 animate-in-6">
				<div className="grid grid-cols-1 md:grid-cols-3 divide-y md:divide-y-0 md:divide-x divide-border/50 border border-border/60">
					<FeatureCard
						index="01"
						label="Instant verdicts"
						desc="Submit a solution and get a verdict in seconds. AC, WA, TLE, MLE — every result explained."
					/>
					<FeatureCard
						index="02"
						label="Real contests"
						desc="Compete in timed contests with full leaderboards. ICPC and IOI scoring both supported."
					/>
					<FeatureCard
						index="03"
						label="Your profile"
						desc="Track submissions, view your history, and share your competitive programming links."
					/>
				</div>
			</section>

			{/* ── Footer ── */}
			<div className="border-t border-border/40">
				<div className="mx-auto max-w-6xl px-6 py-3 flex items-center gap-4 text-xs text-muted-foreground/50 font-mono">
					<span>// built by josh</span>
					<span>──</span>
					<Link href="mailto:joshjms1607@gmail.com" className="hover:text-primary transition-colors">
						joshjms1607@gmail.com
					</Link>
					<span className="ml-auto">jjudge v1.0.0</span>
				</div>
			</div>
		</div>
	);
}

function Row({ label, value, accent }: { label: string; value: string; accent?: boolean }) {
	return (
		<div className="flex items-baseline justify-between">
			<span className="tracking-widest text-muted-foreground uppercase">{label}</span>
			<span className={accent ? "text-primary" : "text-foreground/70"}>{value}</span>
		</div>
	);
}

function FeatureCard({ index, label, desc }: { index: string; label: string; desc: string }) {
	return (
		<div className="group p-6 bg-card/50 hover:bg-card transition-colors">
			<p className="text-[10px] text-primary/50 font-mono mb-3 tracking-widest">{index}</p>
			<h3 className="font-display text-2xl text-foreground mb-2 group-hover:text-primary transition-colors tracking-wide">
				{label.toUpperCase()}
			</h3>
			<p className="text-xs text-muted-foreground leading-relaxed font-mono">{desc}</p>
		</div>
	);
}
