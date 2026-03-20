"use client";

import { useEffect, useState } from "react";
import { useRouter, useParams } from "next/navigation";

import { api, ApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { Button } from "@/components/ui/button";

type BlogPost = {
	id: number;
	title: string;
	slug: string;
	excerpt: string;
	content: string;
	tags: string[];
	published: boolean;
};

type Form = {
	title: string;
	slug: string;
	excerpt: string;
	content: string;
	tags: string;
	published: boolean;
};

export default function EditBlogPostPage() {
	const router = useRouter();
	const params = useParams<{ slug: string }>();
	const auth = useAuth();

	const [form, setForm] = useState<Form>({
		title: "",
		slug: "",
		excerpt: "",
		content: "",
		tags: "",
		published: false,
	});
	const [loading, setLoading] = useState(true);
	const [saving, setSaving] = useState(false);
	const [error, setError] = useState<string | null>(null);

	useEffect(() => {
		if (!auth.token) return;
		api.get<BlogPost>(`/blog/${params.slug}`, {
			headers: { Authorization: `Bearer ${auth.token}` },
		})
			.then((post) => {
				setForm({
					title: post.title,
					slug: post.slug,
					excerpt: post.excerpt ?? "",
					content: post.content,
					tags: post.tags?.join(", ") ?? "",
					published: post.published,
				});
			})
			.catch(() => setError("Failed to load post."))
			.finally(() => setLoading(false));
	}, [params.slug, auth.token]);

	const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
		const { name, value } = e.target;
		setForm((prev) => ({ ...prev, [name]: value }));
	};

	const handleSubmit = async (e: React.FormEvent, published: boolean) => {
		e.preventDefault();
		setSaving(true);
		setError(null);

		const tags = form.tags
			.split(",")
			.map((t) => t.trim())
			.filter(Boolean);

		try {
			const updated = await api.patch<{ slug: string }>(`/blog/${params.slug}`, {
				title: form.title,
				slug: form.slug !== params.slug ? form.slug : undefined,
				excerpt: form.excerpt || null,
				content: form.content,
				tags,
				published,
			}, {
				headers: { Authorization: `Bearer ${auth.token}` },
			});
			router.push(`/blog/${updated.slug}`);
		} catch (err) {
			setError(err instanceof ApiError ? (err.data as { error?: string })?.error ?? "Failed to save." : "Failed to save.");
		} finally {
			setSaving(false);
		}
	};

	if (loading) {
		return (
			<div className="flex flex-col gap-6">
				<p className="font-mono text-sm text-muted-foreground">Loading…</p>
			</div>
		);
	}

	return (
		<div className="flex flex-col gap-6">
			<div>
				<h1 className="font-display text-3xl tracking-wide">EDIT POST</h1>
				<p className="font-mono text-xs text-muted-foreground mt-0.5">/{params.slug}</p>
			</div>

			{error && (
				<p className="border border-destructive/50 bg-destructive/10 px-4 py-3 text-sm text-destructive font-mono">
					{error}
				</p>
			)}

			<form className="flex flex-col gap-5" onSubmit={(e) => handleSubmit(e, form.published)}>
				<BlogField label="Title" required>
					<input
						name="title"
						type="text"
						value={form.title}
						onChange={handleChange}
						required
						className={inputClass}
					/>
				</BlogField>

				<BlogField label="Slug">
					<input
						name="slug"
						type="text"
						value={form.slug}
						onChange={handleChange}
						className={inputClass}
					/>
				</BlogField>

				<BlogField label="Excerpt">
					<input
						name="excerpt"
						type="text"
						value={form.excerpt}
						onChange={handleChange}
						placeholder="A brief description…"
						className={inputClass}
					/>
				</BlogField>

				<BlogField label="Tags" hint="Comma-separated">
					<input
						name="tags"
						type="text"
						value={form.tags}
						onChange={handleChange}
						placeholder="announcement, update"
						className={inputClass}
					/>
				</BlogField>

				<BlogField label="Content" required>
					<textarea
						name="content"
						value={form.content}
						onChange={handleChange}
						rows={20}
						required
						className={`${inputClass} resize-y font-mono text-sm leading-relaxed`}
					/>
				</BlogField>

				<div className="flex gap-3 flex-wrap">
					<Button
						type="submit"
						disabled={saving}
						className="rounded-none px-6 font-mono text-xs tracking-widest uppercase"
						onClick={() => setForm((p) => ({ ...p, published: true }))}
					>
						{saving ? "Saving…" : "Publish"}
					</Button>
					<Button
						type="submit"
						disabled={saving}
						variant="outline"
						className="rounded-none px-6 font-mono text-xs tracking-widest uppercase"
						onClick={() => setForm((p) => ({ ...p, published: false }))}
					>
						Save as draft
					</Button>
					<Button
						type="button"
						variant="ghost"
						className="rounded-none px-6 font-mono text-xs tracking-widest uppercase"
						onClick={() => router.push("/manager/blog")}
					>
						Cancel
					</Button>
				</div>
			</form>
		</div>
	);
}

const inputClass =
	"w-full border border-border bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-primary";

function BlogField({
	label,
	hint,
	required,
	children,
}: {
	label: string;
	hint?: string;
	required?: boolean;
	children: React.ReactNode;
}) {
	return (
		<div className="flex flex-col gap-1.5">
			<div className="flex items-baseline gap-2">
				<label className="font-mono text-xs font-semibold uppercase tracking-widest text-foreground">
					{label}
					{required && <span className="text-primary ml-0.5">*</span>}
				</label>
				{hint && <span className="font-mono text-[10px] text-muted-foreground/60">{hint}</span>}
			</div>
			{children}
		</div>
	);
}
