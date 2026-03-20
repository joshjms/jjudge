"use client";

import Link from "next/link";
import { useEffect, useState } from "react";

import { api, ApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { Button } from "@/components/ui/button";

type BlogPost = {
	id: number;
	title: string;
	slug: string;
	excerpt: string;
	published: boolean;
	tags: string[];
	created_at: string;
	author: { username: string; name: string };
};

function formatDate(value: string) {
	return new Intl.DateTimeFormat(undefined, {
		year: "numeric",
		month: "short",
		day: "2-digit",
	}).format(new Date(value));
}

export default function ManagerBlogPage() {
	const auth = useAuth();
	const [posts, setPosts] = useState<BlogPost[]>([]);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);
	const [deleting, setDeleting] = useState<number | null>(null);

	useEffect(() => {
		api.get<{ items: BlogPost[] }>("/blog", {
			headers: auth.token ? { Authorization: `Bearer ${auth.token}` } : undefined,
		})
			.then((res) => setPosts(res.items ?? []))
			.catch(() => setError("Failed to load posts."))
			.finally(() => setLoading(false));
	}, [auth.token]);

	const handleDelete = async (slug: string, id: number) => {
		if (!confirm("Delete this post?")) return;
		setDeleting(id);
		try {
			await api.delete(`/blog/${slug}`, {
				headers: { Authorization: `Bearer ${auth.token}` },
			});
			setPosts((prev) => prev.filter((p) => p.id !== id));
		} catch (err) {
			alert(err instanceof ApiError ? (err.data as { error?: string })?.error ?? "Failed to delete." : "Failed to delete.");
		} finally {
			setDeleting(null);
		}
	};

	return (
		<div className="flex flex-col gap-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="font-display text-3xl tracking-wide">BLOG</h1>
					<p className="font-mono text-xs text-muted-foreground mt-0.5">
						{posts.length} post{posts.length !== 1 ? "s" : ""}
					</p>
				</div>
				<Button asChild className="rounded-none text-xs font-mono tracking-widest uppercase">
					<Link href="/manager/blog/new">+ New post</Link>
				</Button>
			</div>

			{error && (
				<p className="border border-destructive/50 bg-destructive/10 px-4 py-3 text-sm text-destructive font-mono">
					{error}
				</p>
			)}

			{loading ? (
				<p className="font-mono text-sm text-muted-foreground">Loading…</p>
			) : posts.length === 0 ? (
				<p className="font-mono text-sm text-muted-foreground/50 italic">
					{"// no posts yet"}
				</p>
			) : (
				<div className="flex flex-col gap-0 border border-border/60">
					{posts.map((post) => (
						<div
							key={post.id}
							className="flex items-center gap-4 border-b border-border/50 px-4 py-3 last:border-b-0 hover:bg-muted/30 transition-colors"
						>
							<div className="flex-1 min-w-0">
								<div className="flex items-center gap-2 flex-wrap">
									<Link
										href={`/blog/${post.slug}`}
										className="font-mono text-sm font-semibold text-foreground hover:text-primary transition-colors truncate"
									>
										{post.title}
									</Link>
									{!post.published && (
										<span className="font-mono text-[9px] tracking-[0.15em] uppercase border border-primary/40 bg-primary/10 text-primary px-1.5 py-0.5 shrink-0">
											DRAFT
										</span>
									)}
								</div>
								<p className="font-mono text-[11px] text-muted-foreground/60 mt-0.5">
									{formatDate(post.created_at)} · /{post.slug}
								</p>
							</div>

							<div className="flex items-center gap-2 shrink-0">
								<Link
									href={`/manager/blog/${post.slug}/edit`}
									className="font-mono text-[11px] tracking-widest uppercase text-muted-foreground hover:text-foreground transition-colors border border-border/50 px-2.5 py-1.5 hover:border-border"
								>
									Edit
								</Link>
								<button
									onClick={() => handleDelete(post.slug, post.id)}
									disabled={deleting === post.id}
									className="font-mono text-[11px] tracking-widest uppercase text-muted-foreground hover:text-destructive transition-colors border border-border/50 px-2.5 py-1.5 hover:border-destructive/50 disabled:opacity-50"
								>
									{deleting === post.id ? "…" : "Delete"}
								</button>
							</div>
						</div>
					))}
				</div>
			)}
		</div>
	);
}
