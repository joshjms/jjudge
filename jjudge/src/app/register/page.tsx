"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useMemo, useState } from "react";

import { Button } from "@/components/ui/button";
import { api } from "@/lib/api";
import { setAuth } from "@/lib/auth";

type RegisterResponse = {
	token?: string;
	user?: { name?: string; email?: string; username?: string; role?: string | null };
};

export default function RegisterPage() {
	const router = useRouter();
	const [name, setName] = useState("");
	const [username, setUsername] = useState("");
	const [email, setEmail] = useState("");
	const [password, setPassword] = useState("");
	const [confirm, setConfirm] = useState("");
	const [loading, setLoading] = useState(false);
	const [error, setError] = useState<string | null>(null);

	const isDisabled = useMemo(() => {
		return loading || !email || !password || !username || password !== confirm;
	}, [confirm, email, loading, password, username]);

	const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
		event.preventDefault();
		if (password !== confirm) {
			setError("Passwords do not match.");
			return;
		}
		setLoading(true);
		setError(null);

		try {
			const response = await api.post<RegisterResponse>("/auth/register", {
				name: name || undefined,
				username,
				email,
				password,
			});
			const token = response?.token ?? "dev-token";
			const user =
				response?.user ??
				{
					name: name || username || email.split("@")[0],
					email,
					username,
				};
			setAuth({ token, user });
			router.push("/");
		} catch (err) {
			setError("Registration failed. Check your details and try again.");
		} finally {
			setLoading(false);
		}
	};

	return (
		<div className="flex min-h-[80vh] items-center justify-center px-4 py-16 sm:px-6">
			<div className="w-full max-w-md space-y-8 border border-border/70 bg-background/80 p-8 shadow-lg">
				<div className="space-y-3 text-center">
					<p className="text-xs font-semibold uppercase tracking-[0.3em] text-primary">
						Create account
					</p>
					<h1 className="text-3xl font-bold leading-tight">Join JJudge</h1>
					<p className="text-sm text-muted-foreground">
						Sign up to submit solutions and track your progress.
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
							className="w-full border border-border/70 bg-background px-4 py-3 text-sm outline-none transition focus:border-primary focus:ring-2 focus:ring-primary/30"
							placeholder="adalovelace"
							value={username}
							onChange={(e) => setUsername(e.target.value)}
						/>
					</div>

					<div className="space-y-2">
						<label className="text-sm font-semibold text-foreground" htmlFor="name">
							Name (optional)
						</label>
						<input
							id="name"
							name="name"
							type="text"
							className="w-full border border-border/70 bg-background px-4 py-3 text-sm outline-none transition focus:border-primary focus:ring-2 focus:ring-primary/30"
							placeholder="Ada Lovelace"
							value={name}
							onChange={(e) => setName(e.target.value)}
						/>
					</div>

					<div className="space-y-2">
						<label className="text-sm font-semibold text-foreground" htmlFor="email">
							Email
						</label>
						<input
							id="email"
							name="email"
							type="email"
							required
							autoComplete="email"
							className="w-full border border-border/70 bg-background px-4 py-3 text-sm outline-none transition focus:border-primary focus:ring-2 focus:ring-primary/30"
							placeholder="you@example.com"
							value={email}
							onChange={(e) => setEmail(e.target.value)}
						/>
					</div>

					<div className="space-y-2">
						<label className="text-sm font-semibold text-foreground" htmlFor="password">
							Password
						</label>
						<input
							id="password"
							name="password"
							type="password"
							required
							autoComplete="new-password"
							className="w-full border border-border/70 bg-background px-4 py-3 text-sm outline-none transition focus:border-primary focus:ring-2 focus:ring-primary/30"
							placeholder="••••••••"
							value={password}
							onChange={(e) => setPassword(e.target.value)}
						/>
					</div>

					<div className="space-y-2">
						<label className="text-sm font-semibold text-foreground" htmlFor="confirm">
							Confirm password
						</label>
						<input
							id="confirm"
							name="confirm"
							type="password"
							required
							autoComplete="new-password"
							className="w-full border border-border/70 bg-background px-4 py-3 text-sm outline-none transition focus:border-primary focus:ring-2 focus:ring-primary/30"
							placeholder="••••••••"
							value={confirm}
							onChange={(e) => setConfirm(e.target.value)}
						/>
					</div>

					<Button type="submit" className="w-full rounded-none" disabled={isDisabled}>
						{loading ? "Creating account..." : "Create account"}
					</Button>

					{error && (
						<p className="border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive">
							{error}
						</p>
					)}
				</form>

				<p className="text-center text-sm text-muted-foreground">
					Already have an account?{" "}
					<Link href="/login" className="font-semibold text-primary transition hover:text-primary/80">
						Sign in
					</Link>
				</p>
			</div>
		</div>
	);
}
