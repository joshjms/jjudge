"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

const links = [
	{ label: "Users", href: "/admin/users" },
	{ label: "Problems", href: "/admin/problems" },
	{ label: "Contests", href: "/admin/contests" },
];

export function AdminSidebar() {
	const pathname = usePathname();

	return (
		<nav className="flex flex-col gap-0.5">
			{links.map(({ label, href }) => {
				const active = pathname === href || pathname.startsWith(href + "/");
				return (
					<Link
						key={href}
						href={href}
						className={`border-l-2 px-4 py-2 text-sm font-medium transition-colors ${
							active
								? "border-primary text-foreground"
								: "border-transparent text-muted-foreground hover:border-border hover:text-foreground"
						}`}
					>
						{label}
					</Link>
				);
			})}
		</nav>
	);
}
