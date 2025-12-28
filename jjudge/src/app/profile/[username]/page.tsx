import { notFound } from "next/navigation";

import { api } from "@/lib/api";
import { ProfileEditButton } from "./profile-edit-button";

type UserProfile = {
	username: string;
	name?: string | null;
	bio?: string | null;
	created_at?: string | null;
};

const formatDate = (value?: string | null) => {
	if (!value) return "—";
	return new Intl.DateTimeFormat(undefined, {
		year: "numeric",
		month: "short",
		day: "2-digit",
	}).format(new Date(value));
};

async function fetchUser(username: string) {
	try {
		return await api.get<UserProfile>(`/users/${username}`, { cache: "no-store" });
	} catch {
		return null;
	}
}

export async function generateMetadata({ params }: { params: Promise<{ username: string }> }) {
	const { username } = await params;
	const user = await fetchUser(username);

	return {
		title: user?.name ? `${user.name} (@${user.username})` : `@${username} · Profile`,
		description: `Profile for ${user?.name ?? username}`,
	};
}

export default async function ProfilePage({ params }: { params: Promise<{ username: string }> }) {
	const { username } = await params;
	const user = await fetchUser(username);

	if (!user) {
		notFound();
	}

	return (
		<div className="mx-auto flex w-full max-w-5xl flex-col gap-8 px-4 py-12 sm:px-6">
			<header className="flex flex-col gap-3 border border-border/70 bg-card/70 px-6 py-6 sm:flex-row sm:items-center sm:justify-between">
				<div className="space-y-1">
					<p className="text-xs font-semibold uppercase tracking-[0.25em] text-primary">Profile</p>
					<h1 className="text-3xl font-bold leading-tight sm:text-4xl">
						{user.name ? `${user.name} (@${user.username})` : `@${user.username}`}
					</h1>
					<p className="text-sm text-muted-foreground">
						Joined {formatDate(user.created_at)}
					</p>
				</div>
				<div className="flex flex-col items-start gap-2 text-sm text-muted-foreground sm:items-end">
					<p>
						<span className="font-semibold text-foreground">Username:</span> @{user.username}
					</p>
					<ProfileEditButton username={user.username} />
				</div>
			</header>

			<section className="grid gap-6 sm:grid-cols-3">
				<div className="border border-border/70 bg-background/80 px-4 py-4">
					<p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
						Member since
					</p>
					<p className="text-3xl font-bold text-foreground">{formatDate(user.created_at)}</p>
				</div>
			</section>

			<section className="space-y-3 border border-border/70 bg-background/80 px-6 py-6">
				<h2 className="text-xl font-semibold">About</h2>
				{user.bio ? (
					<p className="text-sm leading-relaxed text-muted-foreground whitespace-pre-line">
						{user.bio}
					</p>
				) : (
					<p className="text-sm text-muted-foreground">No bio provided.</p>
				)}
			</section>
		</div>
	);
}
