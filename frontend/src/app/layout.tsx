import type { Metadata } from "next";
import { Source_Code_Pro, Bebas_Neue } from "next/font/google";
import Script from "next/script";
import "./globals.css";
import Navbar from "@/components/navbar";
import { ThemeProvider } from "@/components/theme-provider";

const sourceCodeProSans = Source_Code_Pro({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const sourceCodeProMono = Source_Code_Pro({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

const bebasNeue = Bebas_Neue({
  weight: "400",
  variable: "--font-bebas-neue",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "JJudge — Online Judge",
  description: "A precision online judge for competitive programmers.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
	const themeScript = `
	(() => {
		const storageKey = "theme";
		const stored = window.localStorage.getItem(storageKey);
		const prefersDark = window.matchMedia("(prefers-color-scheme: dark)").matches;
		const theme = stored === "light" || stored === "dark" ? stored : prefersDark ? "dark" : "light";
		const root = document.documentElement;
		root.classList.toggle("dark", theme === "dark");
		root.style.colorScheme = theme;
	})();
	`;

	return (
		<html lang="en" suppressHydrationWarning>
			<body
				className={`${sourceCodeProSans.variable} ${sourceCodeProMono.variable} ${bebasNeue.variable} antialiased`}
			>
				<Script id="theme-script" strategy="beforeInteractive" dangerouslySetInnerHTML={{ __html: themeScript }} />
				<ThemeProvider>
					<div className="flex min-h-screen flex-col bg-background">
						<Navbar />
						<main className="flex-1">{children}</main>
					</div>
				</ThemeProvider>
			</body>
		</html>
	);
}
