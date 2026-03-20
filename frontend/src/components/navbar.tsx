"use client";

import Link from "next/link";
import { ComponentProps, useEffect, useMemo, useRef, useState } from "react";
import { Menu, LogOut, User } from "lucide-react";

import { Button } from "@/components/ui/button";
import ThemeToggle from "@/components/theme-toggle";
import {
	NavigationMenu,
	NavigationMenuItem,
	NavigationMenuLink,
	NavigationMenuList,
	NavigationMenuViewport,
} from "@/components/ui/navigation-menu";
import {
	Sheet,
	SheetClose,
	SheetContent,
	SheetTitle,
	SheetTrigger,
} from "@/components/ui/sheet";
import * as VisuallyHidden from "@radix-ui/react-visually-hidden";
import { clearAuth, useAuth } from "@/lib/auth";

type NavLink = {
	href: string;
	label: string;
};

const desktopLinkStyles =
	"inline-flex h-10 items-center px-4 text-xs font-mono tracking-widest text-muted-foreground transition-colors duration-150 hover:text-primary data-[active=true]:text-foreground uppercase";

// ── Avatar dropdown ──────────────────────────────────────────────────────────

function UserMenu({
	initial,
	displayName,
	displayEmail,
	username,
}: {
	initial: string;
	displayName: string;
	displayEmail?: string | null;
	username?: string | null;
}) {
	const [open, setOpen] = useState(false);
	const ref = useRef<HTMLDivElement>(null);

	// Close on outside click
	useEffect(() => {
		if (!open) return;
		const handler = (e: MouseEvent) => {
			if (ref.current && !ref.current.contains(e.target as Node)) {
				setOpen(false);
			}
		};
		document.addEventListener("mousedown", handler);
		return () => document.removeEventListener("mousedown", handler);
	}, [open]);

	// Close on Escape
	useEffect(() => {
		if (!open) return;
		const handler = (e: KeyboardEvent) => e.key === "Escape" && setOpen(false);
		document.addEventListener("keydown", handler);
		return () => document.removeEventListener("keydown", handler);
	}, [open]);

	return (
		<div ref={ref} className="relative">
			{/* Trigger */}
			<button
				aria-label="Account menu"
				aria-expanded={open}
				onClick={() => setOpen((v) => !v)}
				className={`
					flex h-8 w-8 items-center justify-center
					border font-bold uppercase font-mono text-xs
					transition-all duration-150 select-none
					${open
						? "border-primary bg-primary text-primary-foreground"
						: "border-primary/40 bg-primary/10 text-primary hover:border-primary/70 hover:bg-primary/20"
					}
				`}
			>
				{initial}
			</button>

			{/* Dropdown panel */}
			{open && (
				<div
					className="
						absolute right-0 top-[calc(100%+6px)] z-50
						w-52 border border-border/70 bg-background
						shadow-[0_8px_24px_rgba(0,0,0,0.12)]
						animate-in-dropdown
					"
					style={{ animation: "dropdown-appear 0.12s ease-out both" }}
				>
					{/* User info header */}
					<div className="border-b border-border/60 bg-muted/40 px-4 py-3">
						<p className="truncate text-xs font-semibold text-foreground font-mono tracking-wide">
							{displayName}
						</p>
						{username && (
							<p className="truncate text-[10px] text-primary font-mono mt-0.5">
								@{username}
							</p>
						)}
						{displayEmail && (
							<p className="truncate text-[10px] text-muted-foreground font-mono mt-0.5">
								{displayEmail}
							</p>
						)}
					</div>

					{/* Menu items */}
					<div className="py-1">
						{username && (
							<Link
								href={`/profile/${username}`}
								onClick={() => setOpen(false)}
								className="
									flex w-full items-center gap-2.5 px-4 py-2.5
									text-xs font-mono tracking-widest uppercase
									text-muted-foreground
									transition-colors hover:bg-muted/60 hover:text-foreground
									group
								"
							>
								<User className="h-3 w-3 shrink-0 text-primary/60 group-hover:text-primary transition-colors" />
								Profile
							</Link>
						)}

						<div className="my-1 border-t border-border/40" />

						<button
							onClick={() => { clearAuth(); setOpen(false); }}
							className="
								flex w-full items-center gap-2.5 px-4 py-2.5
								text-xs font-mono tracking-widest uppercase
								text-muted-foreground
								transition-colors hover:bg-destructive/10 hover:text-destructive
								group
							"
						>
							<LogOut className="h-3 w-3 shrink-0 text-muted-foreground/60 group-hover:text-destructive transition-colors" />
							Sign out
						</button>
					</div>
				</div>
			)}
		</div>
	);
}

// ── Navbar ───────────────────────────────────────────────────────────────────

export function Navbar(props: ComponentProps<"header">) {
	const auth = useAuth();
	const isAuthed = Boolean(auth.token);
	const displayName = auth.user?.name || auth.user?.email || "Account";
	const displayEmail = auth.user?.email;
	const username = auth.user?.username;
	const isAdmin = auth.user?.role?.toLowerCase?.() === "admin";
	const isManager = isAdmin || auth.user?.role?.toLowerCase?.() === "manager";
	const initial = useMemo(
		() => (auth.user?.name || auth.user?.email || "?").slice(0, 1).toUpperCase(),
		[auth.user?.email, auth.user?.name],
	);

	const navLinks: NavLink[] = useMemo(() => {
		const base: NavLink[] = [
			{ href: "/problems", label: "Problems" },
			{ href: "/contests", label: "Contests" },
			{ href: "/submissions", label: "Submissions" },
			{ href: "/blog", label: "Blog" },
		];
		if (isManager) {
			base.push({ href: "/manager/problems", label: "Manager" });
		}
		if (isAdmin) {
			base.push({ href: "/admin/problems", label: "Admin" });
		}
		return base;
	}, [isAdmin, isManager]);

	return (
		<>
			{/* Dropdown animation */}
			<style>{`
				@keyframes dropdown-appear {
					from { opacity: 0; transform: translateY(-6px) scale(0.98); }
					to   { opacity: 1; transform: translateY(0)   scale(1); }
				}
			`}</style>

			<header
				{...props}
				className={`sticky top-0 z-50 w-full border-b border-border/60 bg-background/90 backdrop-blur supports-[backdrop-filter]:bg-background/75 ${props.className ?? ""}`}
			>
				<div className="mx-auto flex h-14 max-w-6xl items-center justify-between px-4 sm:px-6">

					{/* Brand */}
					<Link href="/" className="flex items-center gap-2 group">
						<span className="font-display text-2xl leading-none text-primary tracking-wide group-hover:opacity-80 transition-opacity">
							JJUDGE
						</span>
						<span className="hidden sm:inline-flex text-[10px] font-mono text-muted-foreground/60 border border-border/60 px-1.5 py-0.5 tracking-widest">
							OJ
						</span>
					</Link>

					{/* Desktop nav */}
					<div className="hidden flex-1 justify-center md:flex">
						<NavigationMenu>
							<NavigationMenuList className="rounded-none shadow-none border-none gap-0">
								{navLinks.map((link) => (
									<NavigationMenuItem key={link.href}>
										<NavigationMenuLink asChild>
											<Link href={link.href} className={desktopLinkStyles}>
												{link.label}
											</Link>
										</NavigationMenuLink>
									</NavigationMenuItem>
								))}
							</NavigationMenuList>
							<NavigationMenuViewport />
						</NavigationMenu>
					</div>

					{/* Desktop auth */}
					{isAuthed ? (
						<div className="hidden items-center gap-3 md:flex">
							<UserMenu
								initial={initial}
								displayName={displayName}
								displayEmail={displayEmail}
								username={username}
							/>
						</div>
					) : (
						<div className="hidden items-center gap-2 md:flex">
							<Button variant="ghost" asChild size="sm" className="rounded-none text-xs font-mono tracking-widest uppercase">
								<Link href="/login">Log in</Link>
							</Button>
							<Button asChild size="sm" className="rounded-none text-xs font-mono tracking-widest uppercase bg-primary text-primary-foreground hover:opacity-90">
								<Link href="/register">Register</Link>
							</Button>
						</div>
					)}

					{/* Mobile controls */}
					<div className="flex items-center gap-2 md:hidden">
						<ThemeToggle />
						<Sheet>
							<SheetTrigger asChild>
								<Button variant="ghost" size="icon" className="rounded-none" aria-label="Open navigation">
									<Menu className="h-4 w-4" />
								</Button>
							</SheetTrigger>
							<SheetContent>
								<VisuallyHidden.Root>
									<SheetTitle>Navigation</SheetTitle>
								</VisuallyHidden.Root>
								<div className="mt-10 flex flex-col gap-1">
									{navLinks.map((link) => (
										<SheetClose asChild key={link.href}>
											<Link
												href={link.href}
												className="flex items-center border border-transparent px-4 py-3 text-xs font-mono uppercase tracking-widest text-muted-foreground transition hover:border-border/60 hover:text-primary"
											>
												<span className="text-primary/40 mr-3 text-[10px]">▸</span>
												{link.label}
											</Link>
										</SheetClose>
									))}
								</div>
								<div className="mt-6 flex flex-col gap-3 border-t border-border/50 pt-6">
									{isAuthed ? (
										<>
											<div className="flex items-center gap-3 border border-border/60 bg-muted/40 px-3 py-2">
												<span className="flex h-8 w-8 items-center justify-center border border-primary/40 bg-primary/10 text-xs font-bold uppercase text-primary font-mono">
													{initial}
												</span>
												<div className="leading-tight min-w-0">
													<p className="text-xs font-semibold text-foreground truncate">{displayName}</p>
													{displayEmail && (
														<p className="text-[10px] text-muted-foreground truncate">{displayEmail}</p>
													)}
												</div>
											</div>
											{username && (
												<SheetClose asChild>
													<Link
														href={`/profile/${username}`}
														className="flex items-center gap-2.5 border border-transparent px-4 py-3 text-xs font-mono uppercase tracking-widest text-muted-foreground transition hover:border-border/60 hover:text-primary"
													>
														<span className="text-primary/40 mr-1 text-[10px]">▸</span>
														Profile
													</Link>
												</SheetClose>
											)}
											<SheetClose asChild>
												<Button
													variant="outline"
													className="rounded-none text-xs font-mono tracking-widest uppercase"
													onClick={clearAuth}
												>
													Sign out
												</Button>
											</SheetClose>
										</>
									) : (
										<>
											<SheetClose asChild>
												<Button variant="ghost" asChild className="rounded-none text-xs font-mono tracking-widest uppercase">
													<Link href="/login">Log in</Link>
												</Button>
											</SheetClose>
											<SheetClose asChild>
												<Button asChild className="rounded-none text-xs font-mono tracking-widest uppercase">
													<Link href="/register">Register</Link>
												</Button>
											</SheetClose>
										</>
									)}
								</div>
							</SheetContent>
						</Sheet>
					</div>

					{/* Theme toggle — desktop only */}
					<div className="hidden md:flex ml-2">
						<ThemeToggle />
					</div>
				</div>
			</header>
		</>
	);
}

export default Navbar;
