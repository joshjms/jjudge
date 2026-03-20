"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter, useParams } from "next/navigation";
import Image from "next/image";

import { api, ApiError, getApiBaseUrl } from "@/lib/api";
import { useAuth } from "@/lib/auth";

type ProfileForm = {
	bio: string;
	github: string;
	codeforces: string;
	atcoder: string;
	website: string;
};

type UserProfile = {
	username: string;
	bio?: string | null;
	github?: string | null;
	codeforces?: string | null;
	atcoder?: string | null;
	website?: string | null;
	avatar_url?: string | null;
};

export default function EditProfilePage() {
	const params = useParams<{ username: string }>();
	const username = params.username;
	const router = useRouter();
	const auth = useAuth();

	const [form, setForm] = useState<ProfileForm>({
		bio: "",
		github: "",
		codeforces: "",
		atcoder: "",
		website: "",
	});
	const [avatarUrl, setAvatarUrl] = useState<string | null>(null);
	const [avatarPreview, setAvatarPreview] = useState<string | null>(null);
	const [uploadingAvatar, setUploadingAvatar] = useState(false);
	const [avatarError, setAvatarError] = useState<string | null>(null);
	const fileInputRef = useRef<HTMLInputElement>(null);
	const [loading, setLoading] = useState(true);
	const [saving, setSaving] = useState(false);
	const [error, setError] = useState<string | null>(null);

	// Redirect if not the owner
	useEffect(() => {
		if (!auth.user) return;
		if (auth.user.username !== username) {
			router.replace(`/profile/${username}`);
		}
	}, [auth.user, username, router]);

	// Load current profile
	useEffect(() => {
		api.get<UserProfile>(`/users/${username}`)
			.then((user) => {
				setForm({
					bio: user.bio ?? "",
					github: user.github ?? "",
					codeforces: user.codeforces ?? "",
					atcoder: user.atcoder ?? "",
					website: user.website ?? "",
				});
				if (user.avatar_url) {
					setAvatarUrl(`${getApiBaseUrl()}/users/${username}/avatar`);
				}
			})
			.catch(() => setError("Failed to load profile."))
			.finally(() => setLoading(false));
	}, [username]);

	const handleAvatarChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
		const file = e.target.files?.[0];
		if (!file) return;

		// Local preview
		const objectUrl = URL.createObjectURL(file);
		setAvatarPreview(objectUrl);

		setUploadingAvatar(true);
		setAvatarError(null);

		const formData = new FormData();
		formData.append("avatar", file);

		try {
			await api.post(`/users/${username}/avatar`, formData, {
				headers: { Authorization: `Bearer ${auth.token}` },
			});
			// Bust the avatar cache by appending a timestamp
			setAvatarUrl(`${getApiBaseUrl()}/users/${username}/avatar?t=${Date.now()}`);
		} catch (err) {
			if (err instanceof ApiError) {
				setAvatarError((err.data as { error?: string })?.error ?? "Failed to upload avatar.");
			} else {
				setAvatarError("Failed to upload avatar.");
			}
			setAvatarPreview(null);
		} finally {
			setUploadingAvatar(false);
		}
	};

	const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
		setForm((prev) => ({ ...prev, [e.target.name]: e.target.value }));
	};

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		setSaving(true);
		setError(null);

		const payload = {
			bio:        form.bio        || null,
			github:     form.github     || null,
			codeforces: form.codeforces || null,
			atcoder:    form.atcoder    || null,
			website:    form.website    || null,
		};

		try {
			await api.patch(`/users/${username}/profile`, payload, {
				headers: { Authorization: `Bearer ${auth.token}` },
			});
			router.push(`/profile/${username}`);
		} catch (err) {
			if (err instanceof ApiError) {
				setError((err.data as { error?: string })?.error ?? "Failed to save profile.");
			} else {
				setError("Failed to save profile.");
			}
		} finally {
			setSaving(false);
		}
	};

	if (loading) {
		return (
			<div className="mx-auto flex w-full max-w-2xl px-4 py-12 sm:px-6">
				<p className="text-sm text-muted-foreground">Loading…</p>
			</div>
		);
	}

	return (
		<div className="mx-auto flex w-full max-w-2xl flex-col gap-8 px-4 py-12 sm:px-6">
			<header className="border border-border/70 bg-card/70 px-6 py-6">
				<p className="text-xs font-semibold uppercase tracking-[0.25em] text-primary">Profile</p>
				<h1 className="mt-1 font-display text-5xl">EDIT PROFILE</h1>
				<p className="mt-1 font-mono text-xs text-muted-foreground tracking-widest">@{username}</p>
			</header>

			{/* Avatar upload */}
			<div className="border border-border/70 bg-background/80 px-6 py-6 space-y-4">
				<h2 className="text-[10px] font-mono font-semibold uppercase tracking-widest text-muted-foreground">Avatar</h2>

				<div className="flex items-center gap-5">
					{/* Preview */}
					<div className="relative w-20 h-20 border border-border/60 shrink-0 overflow-hidden bg-muted/20 flex items-center justify-center">
						{(avatarPreview || avatarUrl) ? (
							<Image
								src={avatarPreview ?? avatarUrl!}
								alt="Avatar preview"
								width={80}
								height={80}
								className="w-full h-full object-cover"
								unoptimized
							/>
						) : (
							<span className="font-display text-4xl text-muted-foreground/30">
								{(username || "?").slice(0, 1).toUpperCase()}
							</span>
						)}
						{uploadingAvatar && (
							<div className="absolute inset-0 bg-background/70 flex items-center justify-center">
								<span className="font-mono text-[10px] tracking-widest text-muted-foreground animate-pulse">UPLOADING</span>
							</div>
						)}
					</div>

					<div className="flex flex-col gap-2">
						<input
							ref={fileInputRef}
							type="file"
							accept="image/jpeg,image/png,image/gif,image/webp"
							className="hidden"
							onChange={handleAvatarChange}
						/>
						<button
							type="button"
							onClick={() => fileInputRef.current?.click()}
							disabled={uploadingAvatar}
							className="border border-foreground/40 px-4 py-1.5 text-[10px] font-mono tracking-widest hover:border-foreground/80 transition-colors disabled:opacity-50"
						>
							{uploadingAvatar ? "UPLOADING…" : "CHOOSE IMAGE"}
						</button>
						<p className="font-mono text-[10px] text-muted-foreground tracking-wide">
							JPEG, PNG, GIF or WEBP · max 2 MB
						</p>
						{avatarError && (
							<p className="font-mono text-[10px] text-destructive">{avatarError}</p>
						)}
					</div>
				</div>
			</div>

			<form onSubmit={handleSubmit} className="flex flex-col gap-6">
				{error && (
					<p className="border border-destructive/50 bg-destructive/10 px-4 py-3 font-mono text-[11px] tracking-wide text-destructive">
						{error}
					</p>
				)}

				<div className="border border-border/70 bg-background/80 px-6 py-6 space-y-5">
					<h2 className="text-[10px] font-mono font-semibold uppercase tracking-widest text-muted-foreground">About</h2>

					<div className="flex flex-col gap-1.5">
						<label htmlFor="bio" className="text-[10px] font-mono font-semibold uppercase tracking-widest text-muted-foreground">
							Bio
						</label>
						<textarea
							id="bio"
							name="bio"
							rows={4}
							value={form.bio}
							onChange={handleChange}
							placeholder="Tell the world about yourself…"
							className="w-full resize-none border border-border bg-background px-3 py-2 text-sm font-mono placeholder:text-muted-foreground/50 focus:outline-none focus:ring-1 focus:ring-primary"
						/>
					</div>
				</div>

				<div className="border border-border/70 bg-background/80 px-6 py-6 space-y-5">
					<h2 className="text-[10px] font-mono font-semibold uppercase tracking-widest text-muted-foreground">Links</h2>

					{(
						[
							{ name: "github",     label: "GitHub",     placeholder: "username" },
							{ name: "codeforces", label: "Codeforces", placeholder: "handle" },
							{ name: "atcoder",    label: "AtCoder",    placeholder: "handle" },
							{ name: "website",    label: "Website",    placeholder: "https://example.com" },
						] as const
					).map(({ name, label, placeholder }) => (
						<div key={name} className="flex flex-col gap-1.5">
							<label htmlFor={name} className="text-[10px] font-mono font-semibold uppercase tracking-widest text-muted-foreground">
								{label}
							</label>
							<input
								id={name}
								name={name}
								type="text"
								value={form[name]}
								onChange={handleChange}
								placeholder={placeholder}
								className="w-full border border-border bg-background px-3 py-2 text-sm font-mono placeholder:text-muted-foreground/50 focus:outline-none focus:ring-1 focus:ring-primary"
							/>
						</div>
					))}
				</div>

				<div className="flex gap-3">
					<button
						type="submit"
						disabled={saving}
						className="border border-foreground/60 bg-foreground text-background px-6 py-2 text-[10px] font-mono tracking-widest hover:bg-foreground/90 transition-colors disabled:opacity-50"
					>
						{saving ? "SAVING…" : "SAVE CHANGES"}
					</button>
					<button
						type="button"
						className="border border-border/60 px-6 py-2 text-[10px] font-mono tracking-widest hover:border-foreground/40 transition-colors"
						onClick={() => router.push(`/profile/${username}`)}
					>
						CANCEL
					</button>
				</div>
			</form>
		</div>
	);
}
