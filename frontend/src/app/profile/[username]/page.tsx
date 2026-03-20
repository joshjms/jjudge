import { notFound } from "next/navigation";
import Image from "next/image";

import { api, getPublicApiBaseUrl } from "@/lib/api";
import { ProfileEditButton } from "./profile-edit-button";

type UserProfile = {
	username: string;
	name?: string | null;
	bio?: string | null;
	github?: string | null;
	codeforces?: string | null;
	atcoder?: string | null;
	website?: string | null;
	created_at?: string | null;
	role?: string | null;
	avatar_url?: string | null;
};

const formatDate = (value?: string | null) => {
	if (!value) return "—";
	return new Intl.DateTimeFormat(undefined, {
		year: "numeric",
		month: "short",
		day: "2-digit",
	}).format(new Date(value));
};

const formatYear = (value?: string | null) => {
	if (!value) return "—";
	return new Date(value).getFullYear().toString();
};

async function fetchUser(username: string) {
	try {
		return await api.get<UserProfile>(`/users/${username}`, { cache: "no-store" });
	} catch {
		return null;
	}
}

export async function generateMetadata({ params }: { params: Promise<{ username: string }> }) {
	const { username } = await params;
	const user = await fetchUser(username);
	return {
		title: user?.name ? `${user.name} (@${user.username})` : `@${username} · Profile`,
		description: `Profile for ${user?.name ?? username}`,
	};
}

const socials = [
	{
		key: "github" as keyof UserProfile,
		tag: "GH",
		label: "GitHub",
		href: (v: string) => `https://github.com/${v}`,
	},
	{
		key: "codeforces" as keyof UserProfile,
		tag: "CF",
		label: "Codeforces",
		href: (v: string) => `https://codeforces.com/profile/${v}`,
	},
	{
		key: "atcoder" as keyof UserProfile,
		tag: "AC",
		label: "AtCoder",
		href: (v: string) => `https://atcoder.jp/users/${v}`,
	},
	{
		key: "website" as keyof UserProfile,
		tag: "WW",
		label: "Website",
		href: (v: string) => (v.startsWith("http") ? v : `https://${v}`),
	},
];

export default async function ProfilePage({ params }: { params: Promise<{ username: string }> }) {
	const { username } = await params;
	const user = await fetchUser(username);

	if (!user) notFound();

	const activeSocials = socials.filter((s) => user[s.key]);
	const initial = (user.name || user.username || "?").slice(0, 1).toUpperCase();
	const displayUsername = user.username.toUpperCase();
	const avatarSrc = user.avatar_url ? `${getPublicApiBaseUrl()}/users/${user.username}/avatar` : null;

	return (
		<>
			<style>{`
				/* ── Animations ── */
				@keyframes rise {
					from { opacity: 0; transform: translateY(20px); }
					to   { opacity: 1; transform: translateY(0); }
				}
				@keyframes fade-in {
					from { opacity: 0; }
					to   { opacity: 1; }
				}
				@keyframes pulse-glow {
					0%, 100% { box-shadow: 0 0 0 0 color-mix(in oklch, var(--color-primary) 35%, transparent); }
					50%       { box-shadow: 0 0 0 6px color-mix(in oklch, var(--color-primary) 0%, transparent); }
				}
				@keyframes scan {
					0%   { top: -4px; opacity: 0; }
					5%   { opacity: 1; }
					95%  { opacity: 0.6; }
					100% { top: 110%; opacity: 0; }
				}
				@keyframes draw-accent {
					from { width: 0; }
					to   { width: 100%; }
				}

				/* ── Staggered reveals ── */
				.r1 { animation: rise 0.55s cubic-bezier(0.22,1,0.36,1) 0.00s both; }
				.r2 { animation: rise 0.55s cubic-bezier(0.22,1,0.36,1) 0.08s both; }
				.r3 { animation: rise 0.55s cubic-bezier(0.22,1,0.36,1) 0.16s both; }
				.r4 { animation: rise 0.55s cubic-bezier(0.22,1,0.36,1) 0.24s both; }
				.r5 { animation: rise 0.55s cubic-bezier(0.22,1,0.36,1) 0.32s both; }
				.rf { animation: fade-in 0.4s ease 0.05s both; }

				/* ── Hero ── */
				.hero-card {
					position: relative;
					overflow: hidden;
					border: 1px solid color-mix(in oklch, var(--color-border) 80%, transparent);
					background-color: var(--color-card);
				}

				/* Fine crosshatch grid */
				.hero-bg {
					position: absolute;
					inset: 0;
					background-image:
						linear-gradient(color-mix(in oklch, var(--color-primary) 5%, transparent) 1px, transparent 1px),
						linear-gradient(90deg, color-mix(in oklch, var(--color-primary) 5%, transparent) 1px, transparent 1px);
					background-size: 28px 28px;
				}
				.hero-glow {
					position: absolute;
					inset: 0;
					background: radial-gradient(
						ellipse 60% 80% at 100% 0%,
						color-mix(in oklch, var(--color-primary) 12%, transparent),
						transparent 70%
					);
					pointer-events: none;
				}
				.hero-vignette {
					position: absolute;
					inset: 0;
					background: linear-gradient(
						180deg,
						transparent 50%,
						color-mix(in oklch, var(--color-card) 70%, transparent) 100%
					);
					pointer-events: none;
				}

				/* Amber top bar with draw animation */
				.hero-topbar {
					position: absolute;
					top: 0; left: 0;
					height: 2px;
					width: 100%;
					background: linear-gradient(90deg, var(--color-primary), color-mix(in oklch, var(--color-primary) 30%, transparent));
					animation: draw-accent 0.6s ease 0.1s both;
				}

				/* Dark-mode scan line */
				.dark .hero-scan {
					position: absolute;
					left: 0; right: 0;
					height: 2px;
					background: linear-gradient(90deg,
						transparent 0%,
						color-mix(in oklch, var(--color-primary) 50%, transparent) 40%,
						color-mix(in oklch, var(--color-primary) 50%, transparent) 60%,
						transparent 100%
					);
					animation: scan 5s ease-in-out 0.9s 1 forwards;
					pointer-events: none;
					z-index: 10;
				}

				/* ── Avatar ── */
				.avatar-wrap {
					position: relative;
					display: inline-block;
					padding: 4px;
				}
				.avatar-corner {
					position: absolute;
					width: 10px;
					height: 10px;
					border-color: var(--color-primary);
					opacity: 0.9;
				}
				.ac-tl { top: 0; left: 0;  border-top: 1.5px solid; border-left: 1.5px solid; }
				.ac-tr { top: 0; right: 0; border-top: 1.5px solid; border-right: 1.5px solid; }
				.ac-bl { bottom: 0; left: 0;  border-bottom: 1.5px solid; border-left: 1.5px solid; }
				.ac-br { bottom: 0; right: 0; border-bottom: 1.5px solid; border-right: 1.5px solid; }
				.avatar-inner {
					display: flex;
					align-items: center;
					justify-content: center;
					border: 1.5px solid color-mix(in oklch, var(--color-primary) 55%, transparent);
					background: color-mix(in oklch, var(--color-primary) 10%, transparent);
					font-family: var(--font-display);
					color: var(--color-primary);
					animation: pulse-glow 3s ease-in-out 1.5s 2;
				}

				/* ── Social tags ── */
				.soc-tag {
					display: inline-flex;
					align-items: center;
					gap: 0;
					border: 1px solid var(--color-border);
					font-family: var(--font-mono);
					font-size: 0.68rem;
					letter-spacing: 0.06em;
					text-transform: uppercase;
					color: var(--color-muted-foreground);
					text-decoration: none;
					transition: border-color 0.15s, color 0.15s, background 0.15s;
					background: var(--color-background);
					white-space: nowrap;
					overflow: hidden;
				}
				.soc-tag:hover {
					border-color: var(--color-primary);
					color: var(--color-foreground);
				}
				.soc-tag:hover .soc-abbr {
					background: var(--color-primary);
					color: var(--color-primary-foreground);
				}
				.soc-abbr {
					display: inline-flex;
					align-items: center;
					padding: 0.3rem 0.5rem;
					background: color-mix(in oklch, var(--color-primary) 15%, transparent);
					color: var(--color-primary);
					font-weight: 700;
					font-size: 0.6rem;
					letter-spacing: 0.12em;
					transition: background 0.15s, color 0.15s;
					border-right: 1px solid color-mix(in oklch, var(--color-border) 80%, transparent);
				}
				.soc-val {
					padding: 0.3rem 0.6rem;
				}

				/* ── Panels ── */
				.panel {
					display: flex;
					flex-direction: column;
					border: 1px solid color-mix(in oklch, var(--color-border) 75%, transparent);
					background: color-mix(in oklch, var(--color-background) 85%, transparent);
				}
				.panel-head {
					display: flex;
					align-items: center;
					gap: 0.5rem;
					padding: 0.55rem 1.25rem;
					border-bottom: 1px solid color-mix(in oklch, var(--color-border) 60%, transparent);
					background: color-mix(in oklch, var(--color-muted) 35%, transparent);
				}
				.panel-prompt {
					font-family: var(--font-mono);
					font-size: 0.6rem;
					letter-spacing: 0.25em;
					text-transform: uppercase;
					color: var(--color-primary);
					user-select: none;
				}
				.panel-title {
					font-family: var(--font-mono);
					font-size: 0.65rem;
					letter-spacing: 0.22em;
					text-transform: uppercase;
					color: var(--color-foreground);
					font-weight: 600;
				}
				.panel-rule {
					flex: 1;
					height: 1px;
					background: repeating-linear-gradient(
						90deg,
						color-mix(in oklch, var(--color-border) 60%, transparent) 0px,
						color-mix(in oklch, var(--color-border) 60%, transparent) 4px,
						transparent 4px,
						transparent 8px
					);
					margin-left: 0.5rem;
				}

				/* ── ID table ── */
				.id-tbl {
					width: 100%;
					font-family: var(--font-mono);
					font-size: 0.78rem;
					border-collapse: collapse;
				}
				.id-tbl tr + tr td {
					border-top: 1px solid color-mix(in oklch, var(--color-border) 45%, transparent);
				}
				.id-tbl td {
					padding: 0.45rem 0;
					vertical-align: middle;
				}
				.id-tbl td:first-child {
					font-size: 0.58rem;
					letter-spacing: 0.18em;
					text-transform: uppercase;
					color: var(--color-muted-foreground);
					padding-right: 1.25rem;
					white-space: nowrap;
					width: 1%;
				}
				.id-tbl td:last-child {
					color: var(--color-foreground);
				}

				/* ── Admin hero overrides ── */
				.admin-hero .hero-topbar {
					background: linear-gradient(90deg, #ff4400, rgba(255,68,0,0.15));
				}
				.admin-hero .hero-glow {
					background: radial-gradient(
						ellipse 60% 80% at 100% 0%,
						rgba(255, 80, 0, 0.18),
						transparent 70%
					);
				}
				.admin-hero .hero-bg {
					background-image:
						linear-gradient(rgba(255,68,0,0.05) 1px, transparent 1px),
						linear-gradient(90deg, rgba(255,68,0,0.05) 1px, transparent 1px);
					background-size: 28px 28px;
				}
				.admin-hero .avatar-corner { border-color: #ff4400; }
				.admin-hero .avatar-inner {
					border-color: rgba(255,68,0,0.55);
					background: rgba(255,68,0,0.1);
					color: #ff4400;
				}

				/* ── Status bar ── */
				.status-bar {
					display: flex;
					align-items: center;
					justify-content: space-between;
					padding: 0.45rem 1.25rem;
					border-top: 1px solid color-mix(in oklch, var(--color-border) 55%, transparent);
					background: color-mix(in oklch, var(--color-muted) 25%, transparent);
				}
				.status-dot {
					display: inline-block;
					width: 5px;
					height: 5px;
					background: var(--color-primary);
					border-radius: 50%;
					margin-right: 0.45rem;
					animation: blink 2s step-end infinite;
				}
			`}</style>

			<div className="mx-auto w-full max-w-5xl px-4 py-10 sm:px-6 flex flex-col gap-5">

				{/* ── HERO ── */}
				<div className={`hero-card r1${user.role === "admin" ? " admin-hero" : ""}`}>
					<div className="hero-bg" />
					<div className="hero-glow" />
					<div className="hero-vignette" />
					<div className="hero-topbar" />
					<div className="hero-scan" />

					<div className="relative z-10 px-6 pb-0 pt-7">
						<div className="flex flex-col gap-6 sm:flex-row sm:items-start sm:justify-between">

							{/* Left: identity */}
							<div className="flex flex-col gap-1 min-w-0">
								{/* Badges */}
								<div className="flex flex-wrap items-center gap-2 mb-3">
									<span className="font-mono text-[9px] tracking-[0.35em] uppercase text-primary border border-primary/40 bg-primary/10 px-2 py-0.5">
										PLAYER.FILE
									</span>
									{user.role && user.role !== "user" && user.role !== "admin" && (
										<span className="font-mono text-[9px] tracking-[0.3em] uppercase text-muted-foreground border border-border/50 px-2 py-0.5">
											{user.role.toUpperCase()}
										</span>
									)}
								</div>

								{/* Username — oversized display type */}
								<h1
									className="font-display leading-[0.88] tracking-wide text-foreground break-all"
									style={{
										fontSize: "clamp(3.2rem, 11vw, 7rem)",
										textShadow: "0 0 80px color-mix(in oklch, var(--color-primary) 18%, transparent)",
									}}
								>
									{displayUsername}
								</h1>

								{/* Real name */}
								{user.name && (
									<p className="font-mono text-sm text-muted-foreground mt-1.5 tracking-wider">
										{user.name}
									</p>
								)}

								{/* Meta row */}
								<div className="flex flex-wrap items-center gap-x-5 gap-y-1 mt-4">
									<span className="font-mono text-[10px] text-muted-foreground tracking-widest flex items-center gap-1.5">
										<span className="text-primary text-[10px]">◈</span>
										JOINED {formatDate(user.created_at).toUpperCase()}
									</span>
									<span className="font-mono text-[10px] text-muted-foreground tracking-widest flex items-center gap-1.5">
										<span className="text-primary text-[10px]">◈</span>
										CLASS {formatYear(user.created_at)}
									</span>
								</div>

							</div>

							{/* Right: avatar */}
							<div className="flex flex-col items-start gap-3 sm:items-end shrink-0">
								<div className="avatar-wrap">
									<div className="ac-tl avatar-corner" />
									<div className="ac-tr avatar-corner" />
									<div className="ac-bl avatar-corner" />
									<div className="ac-br avatar-corner" />
									{avatarSrc ? (
										<div
											className="avatar-inner overflow-hidden"
											style={{ width: "92px", height: "92px" }}
										>
											<Image
												src={avatarSrc}
												alt={`${user.username} avatar`}
												width={92}
												height={92}
												className="w-full h-full object-cover"
												unoptimized
											/>
										</div>
									) : (
										<div
											className="avatar-inner"
											style={{ width: "92px", height: "92px", fontSize: "3.6rem", lineHeight: 1 }}
										>
											{initial}
										</div>
									)}
								</div>

								<ProfileEditButton username={user.username} />
							</div>
						</div>
					</div>

					{/* Socials strip */}
					{activeSocials.length > 0 && (
						<div
							className="rf relative z-10 flex flex-wrap items-center gap-2 border-t border-border/40 px-6 py-3 mt-5"
							style={{ background: "color-mix(in oklch, var(--color-background) 55%, transparent)" }}
						>
							{activeSocials.map((s) => (
								<a
									key={s.key}
									href={s.href(user[s.key] as string)}
									target="_blank"
									rel="noopener noreferrer"
									className="soc-tag"
								>
									<span className="soc-abbr">{s.tag}</span>
									<span className="soc-val">{user[s.key]}</span>
								</a>
							))}
						</div>
					)}
				</div>

				{/* ── BODY GRID ── */}
				<div className="grid gap-5 lg:grid-cols-[1fr_268px]">

					{/* About */}
					<section className="r3 panel">
						<div className="panel-head">
							<span className="panel-prompt">//</span>
							<span className="panel-title">ABOUT</span>
							<div className="panel-rule" />
						</div>
						<div className="px-5 py-5 flex-1">
							{user.bio ? (
								<p className="font-mono text-sm leading-relaxed text-muted-foreground whitespace-pre-line">
									{user.bio}
								</p>
							) : (
								<p className="font-mono text-sm text-muted-foreground/40 italic">
									{"// no bio provided"}
								</p>
							)}
						</div>
					</section>

					{/* ID File */}
					<aside className="r4 panel">
						<div className="panel-head">
							<span className="panel-prompt">//</span>
							<span className="panel-title">ID FILE</span>
							<div className="panel-rule" />
						</div>

						<div className="px-5 py-4 flex-1">
							<table className="id-tbl">
								<tbody>
									<tr>
										<td>Username</td>
										<td className="text-primary">@{user.username}</td>
									</tr>
									<tr>
										<td>Enrolled</td>
										<td>{formatDate(user.created_at)}</td>
									</tr>
									<tr>
										<td>Class</td>
										<td>{formatYear(user.created_at)}</td>
									</tr>
									{user.role && (
										<tr>
											<td>Role</td>
											<td className="capitalize">{user.role}</td>
										</tr>
									)}
									{activeSocials.map((s) => (
										<tr key={s.key}>
											<td>{s.label}</td>
											<td>
												<a
													href={s.href(user[s.key] as string)}
													target="_blank"
													rel="noopener noreferrer"
													className="text-primary hover:underline underline-offset-2 block truncate max-w-[150px]"
												>
													{user[s.key]}
												</a>
											</td>
										</tr>
									))}
								</tbody>
							</table>
						</div>

						<div className="status-bar">
							<span className="font-mono text-[9px] text-muted-foreground/50 tracking-widest uppercase">
								STATUS
							</span>
							<span className="font-mono text-[9px] tracking-widest uppercase text-primary flex items-center">
								<span className="status-dot" />
								ACTIVE
							</span>
						</div>
					</aside>

				</div>
			</div>
		</>
	);
}
