"use client";

import Link from "next/link";
import { ComponentProps, useMemo } from "react";
import { Menu } from "lucide-react";

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
	SheetTrigger,
} from "@/components/ui/sheet";
import { clearAuth, useAuth } from "@/lib/auth";

type NavLink = {
	href: string;
	label: string;
};

const desktopLinkStyles =
	"inline-flex h-10 items-center px-4 text-xs font-mono tracking-widest text-muted-foreground transition-colors duration-150 hover:text-primary data-[active=true]:text-foreground uppercase";

export function Navbar(props: ComponentProps<"header">) {
	const auth = useAuth();
	const isAuthed = Boolean(auth.token);
	const displayName = auth.user?.name || auth.user?.email || "Account";
	const displayEmail = auth.user?.email;
	const isAdmin = auth.user?.role?.toLowerCase?.() === "admin";
	const initial = useMemo(
		() => (auth.user?.name || auth.user?.email || "?").slice(0, 1).toUpperCase(),
		[auth.user?.email, auth.user?.name],
	);

	const navLinks: NavLink[] = useMemo(() => {
		const base: NavLink[] = [
			{ href: "/problems", label: "Problems" },
			{ href: "/contests", label: "Contests" },
			{ href: "/submissions", label: "Submissions" },
		];
		if (isAdmin) {
			base.push({ href: "/admin/problems", label: "Admin" });
		}
		return base;
	}, [isAdmin]);

	const handleLogout = () => {
		clearAuth();
	};

	return (
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
						<span className="flex h-8 w-8 items-center justify-center border border-primary/40 bg-primary/10 text-xs font-bold uppercase text-primary font-mono">
							{initial}
						</span>
						<Button
							variant="outline"
							size="sm"
							className="rounded-none text-xs font-mono tracking-widest uppercase border-border/60 hover:border-primary/60"
							onClick={handleLogout}
						>
							Sign out
						</Button>
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
											<div className="leading-tight">
												<p className="text-xs font-semibold text-foreground">{displayName}</p>
												{displayEmail && (
													<p className="text-[10px] text-muted-foreground">{displayEmail}</p>
												)}
											</div>
										</div>
										<SheetClose asChild>
											<Button variant="outline" className="rounded-none text-xs font-mono tracking-widest uppercase" onClick={handleLogout}>
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
	);
}

export default Navbar;
