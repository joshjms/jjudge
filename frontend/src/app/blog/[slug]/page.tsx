import Link from "next/link";
import { notFound } from "next/navigation";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import remarkMath from "remark-math";
import rehypeKatex from "rehype-katex";
import "katex/dist/katex.min.css";

import { api } from "@/lib/api";

export const dynamic = "force-dynamic";

type BlogAuthor = { username: string; name: string };
type BlogPost = {
	id: number;
	title: string;
	slug: string;
	content: string;
	excerpt: string;
	author: BlogAuthor;
	published: boolean;
	tags: string[];
	created_at: string;
	updated_at: string;
};

async function fetchPost(slug: string): Promise<BlogPost | null> {
	try {
		return await api.get<BlogPost>(`/blog/${slug}`, { cache: "no-store" });
	} catch {
		return null;
	}
}

export async function generateMetadata({ params }: { params: Promise<{ slug: string }> }) {
	const { slug } = await params;
	const post = await fetchPost(slug);
	if (!post) return { title: "Post not found" };
	return {
		title: `${post.title} · Blog`,
		description: post.excerpt || post.content.slice(0, 140),
	};
}

function formatDate(value: string) {
	return new Intl.DateTimeFormat(undefined, {
		year: "numeric",
		month: "long",
		day: "numeric",
	}).format(new Date(value));
}

export default async function BlogPostPage({ params }: { params: Promise<{ slug: string }> }) {
	const { slug } = await params;
	const post = await fetchPost(slug);

	if (!post) notFound();

	return (
		<div className="mx-auto w-full max-w-3xl px-4 py-12 sm:px-6">
			{/* Back */}
			<Link
				href="/blog"
				className="inline-flex items-center gap-1.5 font-mono text-[11px] tracking-widest uppercase text-muted-foreground hover:text-primary transition-colors mb-8"
			>
				<span className="text-primary">←</span> Blog
			</Link>

			{/* Header */}
			<header className="mb-10 flex flex-col gap-3">
				{/* Tags */}
				{post.tags?.length > 0 && (
					<div className="flex flex-wrap gap-2">
						{post.tags.map((tag) => (
							<span
								key={tag}
								className="font-mono text-[9px] tracking-[0.15em] uppercase border border-border/50 px-1.5 py-0.5 text-muted-foreground/60"
							>
								{tag}
							</span>
						))}
					</div>
				)}

				<h1 className="font-display text-4xl sm:text-5xl tracking-wide text-foreground leading-[1.05]">
					{post.title}
				</h1>

				{post.excerpt && (
					<p className="font-mono text-sm text-muted-foreground leading-relaxed">
						{post.excerpt}
					</p>
				)}

				{/* Meta bar */}
				<div className="flex flex-wrap items-center gap-4 border-t border-border/50 pt-4 mt-1">
					<span className="font-mono text-[11px] text-muted-foreground/60 tracking-wider">
						{formatDate(post.created_at)}
					</span>
					<span className="font-mono text-[11px] text-muted-foreground/60">·</span>
					<span className="font-mono text-[11px] text-muted-foreground/60 tracking-wider">
						by{" "}
						<Link
							href={`/profile/${post.author?.username}`}
							className="text-primary/80 hover:text-primary transition-colors"
						>
							{post.author?.name || post.author?.username}
						</Link>
					</span>
					{!post.published && (
						<span className="font-mono text-[9px] tracking-[0.15em] uppercase border border-primary/40 bg-primary/10 text-primary px-1.5 py-0.5">
							DRAFT
						</span>
					)}
				</div>
			</header>

			{/* Content */}
			<div className="markdown-content">
				<ReactMarkdown
					remarkPlugins={[remarkGfm, remarkMath]}
					rehypePlugins={[rehypeKatex]}
					skipHtml
				>
					{post.content}
				</ReactMarkdown>
			</div>
		</div>
	);
}
