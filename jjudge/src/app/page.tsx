import Link from "next/link";

import { api } from "@/lib/api";

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
			<h1 className="text-4xl font-bold mb-10">Welcome to JJudge</h1>
			<p>Hi, I&apos;m Josh. I made this as a hobby project / final year project. If you have any suggestions, feel free to email me at <Link href="mailto:joshjms1607@gmail.com" className="underline">joshjms1607@gmail.com</Link>.</p>
			<br />
			<p>Without further ado, have fun with these <Link href="/problems" className="underline">problems</Link>?</p>
		</section>
		</>
	);
}
