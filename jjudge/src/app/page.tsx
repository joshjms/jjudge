import Link from "next/link";

import { Button } from "@/components/ui/button";
import { api } from "@/lib/api";
import {
	ArrowRight,
	Clock9,
	Code2,
	ShieldCheck,
	Target,
	Trophy,
	Users,
} from "lucide-react";

type Problem = {
	id?: string;
	title?: string;
	slug?: string;
	difficulty?: string;
	tags?: string[];
};

const fetchProblems = async () => {
	try {
		return await api.get<Problem[]>("/problems", { cache: "no-store" });
	} catch {
		return null;
	}
};

export default async function Home() {
	const problems = await fetchProblems();

	return (
		<>
		<section className="px-24 py-10 md:py-16 lg:py-24 mx-auto max-w-6xl">
			<p>No Announcements...</p>
		</section>
		</>
	);
}
