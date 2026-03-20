"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

const links = [
	{ label: "Problems", href: "/manager/problems" },
	{ label: "Contests", href: "/manager/contests" },
	{ label: "Blog", href: "/manager/blog" },
];

export function ManagerSidebar() {
	const pathname = usePathname();

	return (
		<nav className="flex flex-col gap-0.5">
			{links.map(({ label, href }) => {
				const active = pathname === href || pathname.startsWith(href + "/");
				return (
					<Link
						key={href}
						href={href}
						className={`border-l-2 px-4 py-2 text-[11px] font-mono tracking-widest uppercase transition-colors ${
							active
								? "border-primary text-foreground"
								: "border-transparent text-muted-foreground/60 hover:border-border hover:text-foreground"
						}`}
					>
						{label}
					</Link>
				);
			})}
		</nav>
	);
}
