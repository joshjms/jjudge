"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useMemo, useState } from "react";

import { Button } from "@/components/ui/button";
import { api } from "@/lib/api";
import { setAuth } from "@/lib/auth";

type LoginResponse = {
	token?: string;
	user?: { name?: string; email?: string; username?: string; role?: string | null };
};

function LoginForm() {
	const router = useRouter();
	const searchParams = useSearchParams();
	const [username, setUsername] = useState("");
	const [password, setPassword] = useState("");
	const [loading, setLoading] = useState(false);
	const [error, setError] = useState<string | null>(null);

	const isDisabled = useMemo(
		() => loading || !username || !password,
		[loading, username, password],
	);

	const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
		event.preventDefault();
		setLoading(true);
		setError(null);

		try {
			const response = await api.post<LoginResponse>("/auth/login", { username, password });
			const token = response?.token ?? "dev-token";
			const user =
				response?.user ??
				{
					name: username,
					username,
				};
			setAuth({ token, user });
			const redirectTo =
				searchParams.get("redirect") ?? searchParams.get("returnTo") ?? searchParams.get("from");
			if (redirectTo) {
				router.push(redirectTo);
			} else {
				router.back();
			}
		} catch (err) {
			setError("Sign in failed. Check credentials and try again.");
		} finally {
			setLoading(false);
		}
	};

	return (
		<div className="flex min-h-[80vh] items-center justify-center px-4 py-16 sm:px-6">
			<div className="w-full max-w-md space-y-8 border border-border/70 bg-background/80 p-8 shadow-lg">
				<div className="space-y-3 text-center">
					<p className="text-xs font-semibold uppercase tracking-[0.3em] text-primary">
						Welcome back
					</p>
					<h1 className="text-3xl font-bold leading-tight">Sign in to JJudge</h1>
					<p className="text-sm text-muted-foreground">
						Access your dashboard, submissions, and contests.
					</p>
				</div>

				<form className="space-y-6" onSubmit={handleSubmit}>
					<div className="space-y-2">
						<label className="text-sm font-semibold text-foreground" htmlFor="username">
							Username
						</label>
						<input
							id="username"
							name="username"
							type="text"
							required
							autoComplete="username"
							className="w-full border border-border/70 bg-background px-4 py-3 text-sm outline-none transition focus:border-primary focus:ring-2 focus:ring-primary/30"
							placeholder="yourname"
							value={username}
							onChange={(e) => setUsername(e.target.value)}
						/>
					</div>

					<div className="space-y-2">
						<div className="flex items-center justify-between text-sm font-semibold text-foreground">
							<label htmlFor="password">Password</label>
							<Link
								href="/forgot-password"
								className="text-primary transition hover:text-primary/80"
							>
								Forgot?
							</Link>
						</div>
						<input
							id="password"
							name="password"
							type="password"
							required
							autoComplete="current-password"
							className="w-full border border-border/70 bg-background px-4 py-3 text-sm outline-none transition focus:border-primary focus:ring-2 focus:ring-primary/30"
							placeholder="••••••••"
							value={password}
							onChange={(e) => setPassword(e.target.value)}
						/>
					</div>

					<div className="flex items-center gap-2 text-sm text-muted-foreground">
						<input
							id="remember"
							name="remember"
							type="checkbox"
							className="h-4 w-4 border-border/70 text-primary focus:ring-primary/40"
						/>
						<label htmlFor="remember">Remember me</label>
					</div>

					<Button type="submit" className="w-full rounded-none" disabled={isDisabled}>
						{loading ? "Signing in..." : "Sign in"}
					</Button>

					{error && (
						<p className="border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive">
							{error}
						</p>
					)}
				</form>

				<p className="text-center text-sm text-muted-foreground">
					No account?{" "}
					<Link href="/register" className="font-semibold text-primary transition hover:text-primary/80">
						Create one
					</Link>
				</p>
			</div>
		</div>
	);
}

export default function LoginPage() {
	return (
		<Suspense fallback={null}>
			<LoginForm />
		</Suspense>
	);
}
