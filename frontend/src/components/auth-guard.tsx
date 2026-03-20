"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";

import { useAuth } from "@/lib/auth";

type AuthGuardProps = {
	children: React.ReactNode;
	/** If provided, also checks that the user's role matches. */
	requireRole?: string | string[];
};

export function AuthGuard({ children, requireRole }: AuthGuardProps) {
	const auth = useAuth();
	const router = useRouter();

	useEffect(() => {
		if (!auth.token) {
			router.replace("/login");
			return;
		}
		if (requireRole) {
			const role = auth.user?.role?.toLowerCase() ?? "";
			const allowed = Array.isArray(requireRole)
				? requireRole.map((r) => r.toLowerCase())
				: [requireRole.toLowerCase()];
			if (!allowed.includes(role)) {
				router.replace("/login");
			}
		}
	}, [auth.token, auth.user?.role, requireRole, router]);

	// Render nothing until we confirm auth (avoids a flash of protected content).
	if (!auth.token) return null;
	if (requireRole) {
		const role = auth.user?.role?.toLowerCase() ?? "";
		const allowed = Array.isArray(requireRole)
			? requireRole.map((r) => r.toLowerCase())
			: [requireRole.toLowerCase()];
		if (!allowed.includes(role)) return null;
	}

	return <>{children}</>;
}
