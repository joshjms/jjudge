import Link from "next/link";

export default function Home() {
	return (
		<div className="flex flex-col min-h-[calc(100vh-4rem)]">

			{/* ── System status bar ── */}
			<div className="border-b border-border/60 bg-muted/30">
				<div className="mx-auto max-w-6xl px-6 py-2 flex items-center gap-4 text-xs text-muted-foreground font-mono animate-in-1">
					<span className="text-primary font-semibold">SYSTEM:ONLINE</span>
					<span className="opacity-40">──</span>
					<span>v1.0.0</span>
					<span className="opacity-40">──</span>
					<span>C++20 · Python 3 · Java</span>
					<span className="ml-auto text-primary cursor-blink">▌</span>
				</div>
			</div>

			{/* ── Hero ── */}
			<section className="mx-auto max-w-6xl w-full px-6 pt-12 pb-4 md:pt-20 md:pb-8 flex-1 flex flex-col justify-center">
				<div className="grid grid-cols-1 lg:grid-cols-12 gap-8 lg:gap-12 items-start">

					{/* Left: Big text */}
					<div className="lg:col-span-7 xl:col-span-8">
						<p className="text-xs text-muted-foreground tracking-widest mb-5 animate-in-1">
							[ ONLINE JUDGE // HOBBY PROJECT ]
						</p>

						<h1
							className="font-display leading-none tracking-tight text-foreground flicker"
							style={{ fontSize: "clamp(5rem, 14vw, 10.5rem)" }}
						>
							<span className="block animate-in-2">CODE.</span>
							<span className="block text-primary animate-in-3">COMPETE.</span>
							<span className="block animate-in-4">CONQUER.</span>
						</h1>

						<p className="mt-7 max-w-md text-sm text-muted-foreground leading-relaxed animate-in-5">
							A precision online judge built for competitive programmers.
							Submit solutions, receive instant verdicts, sharpen your skills.
						</p>

						<div className="mt-9 flex flex-wrap gap-3 animate-in-6">
							<Link
								href="/problems"
								className="inline-flex items-center gap-2 px-6 py-3 bg-primary text-primary-foreground text-sm font-mono font-bold tracking-wide transition-all hover:opacity-90 hover:gap-3"
							>
								START SOLVING <span>→</span>
							</Link>
							<Link
								href="/register"
								className="inline-flex items-center gap-2 px-6 py-3 border border-primary/50 text-primary text-sm font-mono tracking-wide transition-all hover:border-primary hover:bg-primary/5"
							>
								CREATE ACCOUNT
							</Link>
						</div>
					</div>

					{/* Right: Terminal panel */}
					<div className="lg:col-span-5 xl:col-span-4 animate-in-5">
						<div className="border border-border bg-card p-5">
							<div className="flex items-center gap-2 mb-4 pb-3 border-b border-border/60">
								<span className="text-xs text-muted-foreground tracking-widest">// STATUS</span>
								<span className="ml-auto w-1.5 h-1.5 rounded-full bg-primary cursor-blink" />
							</div>

							<div className="space-y-3 font-mono text-sm">
								<div className="flex justify-between items-baseline">
									<span className="text-muted-foreground text-xs tracking-widest">JUDGE</span>
									<span className="text-primary text-xs">ONLINE</span>
								</div>
								<div className="flex justify-between items-baseline">
									<span className="text-muted-foreground text-xs tracking-widest">EXECUTION</span>
									<span className="text-foreground/70 text-xs">SANDBOXED</span>
								</div>
								<div className="flex justify-between items-baseline">
									<span className="text-muted-foreground text-xs tracking-widest">SCORING</span>
									<span className="text-foreground/70 text-xs">ICPC · IOI</span>
								</div>
								<div className="flex justify-between items-baseline">
									<span className="text-muted-foreground text-xs tracking-widest">LANGUAGES</span>
									<span className="text-foreground/70 text-xs">C++ · PY · JAVA</span>
								</div>
							</div>

							<div className="mt-5 pt-4 border-t border-border/60">
								<p className="text-xs text-muted-foreground leading-relaxed">
									Code executes in isolated containers with strict resource limits.
									Every submission is fair.
								</p>
							</div>
						</div>
					</div>
				</div>
			</section>

			{/* ── Divider ── */}
			<div className="mx-auto max-w-6xl w-full px-6 animate-in-7">
				<div className="border-t border-border/60 flex items-center gap-3 py-0">
					<span className="text-xs text-muted-foreground/40 tracking-[0.3em] py-3">──────</span>
					<span className="text-xs text-muted-foreground/40 tracking-widest">FEATURES</span>
					<span className="text-xs text-muted-foreground/40 tracking-[0.3em]">──────</span>
				</div>
			</div>

			{/* ── Feature strip ── */}
			<section className="mx-auto max-w-6xl w-full px-6 pb-16 animate-in-7">
				<div className="grid grid-cols-1 md:grid-cols-3 divide-y md:divide-y-0 md:divide-x divide-border/60 border border-border/60">
					<FeatureCard
						index="01"
						label="MULTI-LANGUAGE"
						desc="C++20, Python 3, Java. Compiled and interpreted, with more languages planned."
					/>
					<FeatureCard
						index="02"
						label="REAL-TIME JUDGING"
						desc="Verdicts in seconds. Code runs inside sandboxed containers with strict resource limits."
					/>
					<FeatureCard
						index="03"
						label="ICPC & IOI SCORING"
						desc="Full support for both ICPC-style and IOI partial-score contest formats."
					/>
				</div>
			</section>

			{/* ── Footer line ── */}
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

function FeatureCard({
	index,
	label,
	desc,
}: {
	index: string;
	label: string;
	desc: string;
}) {
	return (
		<div className="group p-6 bg-card/50 hover:bg-card transition-colors">
			<p className="text-xs text-primary/60 font-mono mb-3 tracking-widest">{index}</p>
			<h3 className="font-display text-2xl text-foreground mb-2 group-hover:text-primary transition-colors">
				{label}
			</h3>
			<p className="text-xs text-muted-foreground leading-relaxed">{desc}</p>
		</div>
	);
}
