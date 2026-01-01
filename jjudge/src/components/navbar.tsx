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
	description?: string;
};

const desktopLinkStyles =
	"inline-flex h-10 items-center px-4 text-sm font-semibold text-muted-foreground transition-colors duration-150 hover:text-primary data-[active=true]:text-foreground";

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
			className={`sticky top-0 z-50 w-full border-b border-border/60 bg-background/80 backdrop-blur supports-[backdrop-filter]:bg-background/60 ${
				props.className ?? ""
			}`}
		>
			<div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-4 sm:px-6">
				<Link href="/" className="flex items-center gap-2 font-semibold">
					<span className="text-lg lowercase tracking-tight text-primary">jjudge</span>
					<span className="bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary">
						&lt;online judge&gt;
					</span>
				</Link>

				<div className="hidden min-w-[320px] flex-1 justify-center md:flex">
					<NavigationMenu>
						<NavigationMenuList className="rounded-none shadow-none border-none">
							{navLinks.map((link) => (
								<NavigationMenuItem key={link.href}>
									<NavigationMenuLink asChild>
										<Link
											href={link.href}
											className={desktopLinkStyles}
											data-active={link.href === "/"}
										>
											{link.label}
										</Link>
									</NavigationMenuLink>
								</NavigationMenuItem>
							))}
						</NavigationMenuList>
						<NavigationMenuViewport />
					</NavigationMenu>
				</div>

				{isAuthed ? (
					<div className="hidden items-center gap-3 md:flex">
						<div className="flex items-center gap-3 px-3 py-1.5">
							<span className="flex h-9 w-9 items-center justify-center rounded-full bg-primary/10 text-sm font-semibold uppercase text-primary">
								{initial}
							</span>
							{/* <div className="leading-tight">
								<p className="text-sm font-semibold text-foreground">{displayName}</p>
								{displayEmail && (
									<p className="text-xs text-muted-foreground">{displayEmail}</p>
								)}
							</div> */}
						</div>
						<Button variant="outline" className="rounded-none" onClick={handleLogout}>
							Sign out
						</Button>
					</div>
				) : (
					<div className="hidden items-center gap-2 md:flex">
						<Button variant="ghost" asChild className="rounded-none">
							<Link href="/login">Log in</Link>
						</Button>
						<Button asChild className="rounded-none">
							<Link href="/register">Get started</Link>
						</Button>
					</div>
				)}

				<div className="flex items-center gap-2 md:hidden">
					<Sheet>
						<SheetTrigger asChild>
							<Button variant="ghost" size="icon" aria-label="Open navigation">
								<Menu className="h-5 w-5" />
							</Button>
						</SheetTrigger>
						<SheetContent>
							<div className="mt-10 flex flex-col gap-6">
								{navLinks.map((link) => (
									<SheetClose asChild key={link.href}>
										<Link
											href={link.href}
											className="flex flex-col rounded-none border border-border/70 px-4 py-3 transition hover:border-primary/60 hover:bg-muted/50"
										>
											<span className="text-base font-semibold">{link.label}</span>
										</Link>
									</SheetClose>
								))}
						<div className="flex flex-col gap-3 border-t border-border/50 pt-6">
							{isAuthed ? (
								<>
									<div className="flex items-center gap-3 rounded-xl border border-border/70 bg-muted/60 px-3 py-2">
										<span className="flex h-9 w-9 items-center justify-center rounded-full bg-primary/10 text-sm font-semibold uppercase text-primary">
													{initial}
												</span>
												<div className="leading-tight">
													<p className="text-sm font-semibold text-foreground">
														{displayName}
													</p>
													{displayEmail && (
														<p className="text-xs text-muted-foreground">{displayEmail}</p>
													)}
												</div>
											</div>
											<SheetClose asChild>
												<Button variant="outline" className="rounded-none" onClick={handleLogout}>
													Sign out
												</Button>
											</SheetClose>
										</>
									) : (
										<>
											<SheetClose asChild>
												<Button variant="ghost" asChild className="rounded-none">
													<Link href="/login">Log in</Link>
												</Button>
											</SheetClose>
											<SheetClose asChild>
												<Button asChild className="rounded-none">
													<Link href="/register">Get started</Link>
												</Button>
											</SheetClose>
										</>
									)}
								</div>
							</div>
						</SheetContent>
					</Sheet>
				</div>
				<div className="hidden md:flex">
					<ThemeToggle />
				</div>
				<div className="md:hidden">
					<ThemeToggle />
				</div>
			</div>
		</header>
	);
}

export default Navbar;
