import Link from "next/link";
import { ActiveMatches } from "@/components/live/active-matches";
import { ThemeToggle } from "@/components/theme-toggle";

export default function LivePage() {
  return (
    <main className="min-h-screen page-shell">
      <div className="mx-auto max-w-6xl px-6 py-12">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <Link className="text-sm text-[color:var(--text-muted)] underline" href="/">
            Назад
          </Link>
          <ThemeToggle />
        </div>
        <div className="mt-6">
          <ActiveMatches />
        </div>
      </div>
    </main>
  );
}
