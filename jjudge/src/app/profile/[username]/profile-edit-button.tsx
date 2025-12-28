"use client";

import Link from "next/link";

import { Button } from "@/components/ui/button";
import { useAuth } from "@/lib/auth";

type ProfileEditButtonProps = {
	username: string;
};

export function ProfileEditButton({ username }: ProfileEditButtonProps) {
	const auth = useAuth();
	const isCurrentUser = auth.user?.username && auth.user.username === username;

	if (!isCurrentUser) return null;

	return (
		<Button asChild variant="outline" className="rounded-none px-4 py-2">
			<Link href={`/profile/${username}/edit`}>Edit profile</Link>
		</Button>
	);
}
