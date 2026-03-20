import Link from "next/link";

import { api } from "@/lib/api";

export const metadata = {
	title: "Blog · JJudge",
	description: "Articles and announcements from the JJudge team.",
};

type BlogAuthor = { username: string; name: string };
type BlogPost = {
	id: number;
	title: string;
	slug: string;
	excerpt: string;
	author: BlogAuthor;
	published: boolean;
	tags: string[];
	created_at: string;
};

async function fetchPosts(): Promise<BlogPost[]> {
	try {
		const res = await api.get<{ items: BlogPost[] }>("/blog", { cache: "no-store" });
		return res.items ?? [];
	} catch {
		return [];
	}
}

function formatDate(value: string) {
	return new Intl.DateTimeFormat(undefined, {
		year: "numeric",
		month: "short",
		day: "2-digit",
	}).format(new Date(value));
}

export default async function BlogPage() {
	const posts = await fetchPosts();

	return (
		<div className="mx-auto w-full max-w-4xl px-4 py-12 sm:px-6">
			{/* Header */}
			<div className="mb-10 border-b border-border/60 pb-6">
				<p className="font-mono text-[10px] tracking-[0.35em] uppercase text-primary mb-2">
					// LOG
				</p>
				<h1 className="font-display text-5xl sm:text-6xl tracking-wide text-foreground">
					BLOG
				</h1>
				<p className="mt-2 font-mono text-sm text-muted-foreground">
					Articles, announcements, and updates.
				</p>
			</div>

			{posts.length === 0 ? (
				<p className="font-mono text-sm text-muted-foreground/50 italic">
					{"// no posts yet"}
				</p>
			) : (
				<div className="flex flex-col gap-0">
					{posts.map((post, i) => (
						<article
							key={post.id}
							className="group border-b border-border/50 py-7 first:border-t"
						>
							<div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
								<div className="flex flex-col gap-2 min-w-0">
									{/* Meta */}
									<div className="flex flex-wrap items-center gap-3">
										<span className="font-mono text-[10px] tracking-widest text-muted-foreground/60 uppercase">
											{formatDate(post.created_at)}
										</span>
										{post.tags?.map((tag) => (
											<span
												key={tag}
												className="font-mono text-[9px] tracking-[0.15em] uppercase border border-border/50 px-1.5 py-0.5 text-muted-foreground/60"
											>
												{tag}
											</span>
										))}
										{!post.published && (
											<span className="font-mono text-[9px] tracking-[0.15em] uppercase border border-primary/40 bg-primary/10 text-primary px-1.5 py-0.5">
												DRAFT
											</span>
										)}
									</div>

									{/* Title */}
									<Link href={`/blog/${post.slug}`}>
										<h2 className="font-display text-2xl sm:text-3xl tracking-wide text-foreground transition-colors group-hover:text-primary leading-tight">
											{post.title}
										</h2>
									</Link>

									{/* Excerpt */}
									{post.excerpt && (
										<p className="font-mono text-sm text-muted-foreground leading-relaxed line-clamp-2">
											{post.excerpt}
										</p>
									)}

									{/* Author */}
									<p className="font-mono text-[11px] text-muted-foreground/60 tracking-wider">
										by{" "}
										<Link
											href={`/profile/${post.author?.username}`}
											className="text-primary/80 hover:text-primary transition-colors"
										>
											{post.author?.name || post.author?.username}
										</Link>
									</p>
								</div>

								{/* Index number */}
								<span
									className="font-display text-4xl text-border/50 shrink-0 select-none hidden sm:block"
									aria-hidden
								>
									{String(posts.length - i).padStart(2, "0")}
								</span>
							</div>
						</article>
					))}
				</div>
			)}
		</div>
	);
}
