"use client";

import { useEffect, useState } from "react";

import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";

type User = {
	id: number;
	username: string;
	email: string;
	name: string;
	role: string;
	created_at: string;
};

type UserListResponse = {
	items: User[];
	page: number;
	limit: number;
	total: number;
};

const formatDate = (value: string) =>
	new Intl.DateTimeFormat(undefined, {
		year: "numeric",
		month: "short",
		day: "2-digit",
	}).format(new Date(value));

export default function AdminUsersPage() {
	const auth = useAuth();
	const [data, setData] = useState<UserListResponse | null>(null);
	const [error, setError] = useState<string | null>(null);

	useEffect(() => {
		if (!auth.token) return;
		api
			.get<UserListResponse>("/auth/users", {
				headers: { Authorization: `Bearer ${auth.token}` },
			})
			.then(setData)
			.catch(() => setError("Failed to load users."));
	}, [auth.token]);

	const users = data?.items ?? [];

	return (
		<div className="flex flex-col gap-6">
			<div className="space-y-1">
				<h1 className="text-2xl font-bold">Users</h1>
				{data && (
					<p className="text-sm text-muted-foreground">{data.total} total</p>
				)}
			</div>

			{error ? (
				<div className="border border-border/70 px-6 py-10 text-center text-sm text-destructive">
					{error}
				</div>
			) : !data ? (
				<div className="border border-border/70 px-6 py-10 text-center text-sm text-muted-foreground">
					Loading...
				</div>
			) : users.length === 0 ? (
				<div className="border border-border/70 px-6 py-10 text-center text-sm text-muted-foreground">
					No users found.
				</div>
			) : (
				<div className="overflow-hidden border border-border/70">
					<table className="min-w-full divide-y divide-border/70 text-sm">
						<thead className="bg-muted/70 text-xs uppercase tracking-wide text-muted-foreground">
							<tr>
								<th className="px-4 py-3 text-left font-semibold">ID</th>
								<th className="px-4 py-3 text-left font-semibold">Username</th>
								<th className="px-4 py-3 text-left font-semibold">Name</th>
								<th className="px-4 py-3 text-left font-semibold">Email</th>
								<th className="px-4 py-3 text-left font-semibold">Role</th>
								<th className="px-4 py-3 text-left font-semibold">Joined</th>
							</tr>
						</thead>
						<tbody className="divide-y divide-border/70">
							{users.map((user) => (
								<tr key={user.id} className="hover:bg-muted/40">
									<td className="px-4 py-3 text-muted-foreground">{user.id}</td>
									<td className="px-4 py-3 font-medium">{user.username}</td>
									<td className="px-4 py-3 text-muted-foreground">{user.name || "—"}</td>
									<td className="px-4 py-3 text-muted-foreground">{user.email}</td>
									<td className="px-4 py-3">
										<span
											className={`border px-2 py-0.5 text-[11px] font-semibold uppercase tracking-wide ${
												user.role === "admin"
													? "border-amber-500/40 bg-amber-500/10 text-amber-700"
													: "border-border/60 text-muted-foreground"
											}`}
										>
											{user.role}
										</span>
									</td>
									<td className="px-4 py-3 text-muted-foreground">
										{formatDate(user.created_at)}
									</td>
								</tr>
							))}
						</tbody>
					</table>
				</div>
			)}
		</div>
	);
}
