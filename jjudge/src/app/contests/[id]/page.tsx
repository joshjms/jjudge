import { notFound } from "next/navigation";

import { api } from "@/lib/api";

import { ContestBody, type Contest } from "./contest-body";

export const dynamic = "force-dynamic";

async function fetchContest(id: string): Promise<Contest | null> {
	try {
		return await api.get<Contest>(`/contests/${id}`, { cache: "no-store" });
	} catch {
		return null;
	}
}

export async function generateMetadata({
	params,
}: {
	params: Promise<{ id: string }>;
}) {
	const { id } = await params;
	const contest = await fetchContest(id);
	if (!contest) return { title: "Contest not found" };
	return { title: `${contest.title} · Contest ${contest.id}` };
}

export default async function ContestPage({
	params,
}: {
	params: Promise<{ id: string }>;
}) {
	const { id } = await params;
	const contest = await fetchContest(id);
	if (!contest) notFound();
	return <ContestBody contest={contest} />;
}
