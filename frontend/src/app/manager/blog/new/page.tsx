"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

import { api, ApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { Button } from "@/components/ui/button";

type Form = {
	title: string;
	slug: string;
	excerpt: string;
	content: string;
	tags: string;
	published: boolean;
};

export default function NewBlogPostPage() {
	const router = useRouter();
	const auth = useAuth();

	const [form, setForm] = useState<Form>({
		title: "",
		slug: "",
		excerpt: "",
		content: "",
		tags: "",
		published: false,
	});
	const [saving, setSaving] = useState(false);
	const [error, setError] = useState<string | null>(null);

	const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
		const { name, value } = e.target;
		setForm((prev) => ({ ...prev, [name]: value }));
	};

	const handleSubmit = async (e: React.FormEvent, publish: boolean) => {
		e.preventDefault();
		setSaving(true);
		setError(null);

		const tags = form.tags
			.split(",")
			.map((t) => t.trim())
			.filter(Boolean);

		try {
			const post = await api.post<{ slug: string }>("/blog", {
				title: form.title,
				slug: form.slug || undefined,
				excerpt: form.excerpt || undefined,
				content: form.content,
				tags,
				published: publish,
			}, {
				headers: { Authorization: `Bearer ${auth.token}` },
			});
			router.push(`/blog/${post.slug}`);
		} catch (err) {
			setError(err instanceof ApiError ? (err.data as { error?: string })?.error ?? "Failed to create post." : "Failed to create post.");
		} finally {
			setSaving(false);
		}
	};

	return (
		<div className="flex flex-col gap-6">
			<div>
				<h1 className="font-display text-3xl tracking-wide">NEW POST</h1>
				<p className="font-mono text-xs text-muted-foreground mt-0.5">
					Create a new blog post
				</p>
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
						placeholder="Post title"
						required
						className={inputClass}
					/>
				</BlogField>

				<BlogField label="Slug" hint="Auto-generated from title if empty">
					<input
						name="slug"
						type="text"
						value={form.slug}
						onChange={handleChange}
						placeholder="my-post-slug"
						className={inputClass}
					/>
				</BlogField>

				<BlogField label="Excerpt" hint="Short summary shown in listings">
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
						placeholder="Write your post in Markdown…"
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
