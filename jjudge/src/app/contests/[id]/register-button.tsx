"use client";

import { useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";

type RegisterButtonProps = {
	contestId: number;
};

export function RegisterButton({ contestId }: RegisterButtonProps) {
	const auth = useAuth();
	const [loading, setLoading] = useState(false);
	const [registered, setRegistered] = useState<boolean | null>(null);
	const [error, setError] = useState<string | null>(null);

	useEffect(() => {
		if (!auth.token) return;
		let cancelled = false;

		api
			.get<{ registered: boolean }>(`/contests/${contestId}/register`, {
				headers: { Authorization: `Bearer ${auth.token}` },
			})
			.then((data) => {
				if (!cancelled) setRegistered(data.registered);
			})
			.catch(() => {
				if (!cancelled) setRegistered(false);
			});

		return () => {
			cancelled = true;
		};
	}, [contestId, auth.token]);

	const handleRegister = async () => {
		if (!auth.token) return;
		setLoading(true);
		setError(null);
		try {
			await api.post(
				`/contests/${contestId}/register`,
				{},
				{ headers: { Authorization: `Bearer ${auth.token}` } },
			);
			setRegistered(true);
		} catch {
			setError("Failed to register. Please try again.");
		} finally {
			setLoading(false);
		}
	};

	const handleUnregister = async () => {
		if (!auth.token) return;
		setLoading(true);
		setError(null);
		try {
			await api.delete(`/contests/${contestId}/register`, {
				headers: { Authorization: `Bearer ${auth.token}` },
			});
			setRegistered(false);
		} catch {
			setError("Failed to unregister. Please try again.");
		} finally {
			setLoading(false);
		}
	};

	if (!auth.token) {
		return (
			<p className="text-sm text-muted-foreground">
				<a href="/login" className="text-primary underline">Log in</a> to register for this contest.
			</p>
		);
	}

	if (registered === null) {
		return (
			<div className="h-9 w-24 animate-pulse bg-muted" />
		);
	}

	return (
		<div className="flex flex-col gap-2">
			{registered ? (
				<div className="flex flex-wrap items-center gap-3">
					<span className="border border-emerald-500/40 bg-emerald-500/10 px-3 py-1 text-xs font-semibold text-emerald-700">
						Registered
					</span>
					<button
						onClick={handleUnregister}
						disabled={loading}
						className="text-xs text-muted-foreground underline hover:text-foreground"
					>
						{loading ? "..." : "Unregister"}
					</button>
				</div>
			) : (
				<Button onClick={handleRegister} disabled={loading} className="rounded-none w-fit">
					{loading ? "Registering..." : "Register"}
				</Button>
			)}
			{error && <p className="text-xs text-destructive">{error}</p>}
		</div>
	);
}
