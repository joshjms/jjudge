import { ManagerSidebar } from "./sidebar";
import { AuthGuard } from "@/components/auth-guard";

export default function ManagerLayout({ children }: { children: React.ReactNode }) {
	return (
		<AuthGuard requireRole={["manager", "admin"]}>
			<div className="mx-auto flex w-full max-w-7xl gap-0 px-4 py-12 sm:px-6">
				{/* Sidebar */}
				<aside className="w-44 shrink-0">
					<p className="mb-3 px-4 text-[11px] font-semibold uppercase tracking-[0.2em] text-muted-foreground">
						MANAGER
					</p>
					<ManagerSidebar />
				</aside>

				{/* Divider */}
				<div className="mx-6 w-px shrink-0 bg-border/60" />

				{/* Content */}
				<div className="min-w-0 flex-1">{children}</div>
			</div>
		</AuthGuard>
	);
}
