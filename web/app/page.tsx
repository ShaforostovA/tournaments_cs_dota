import Link from "next/link";
import { Card } from "@/components/ui/card";
import { Pagination } from "@/components/pagination";
import { ThemeToggle } from "@/components/theme-toggle";

type Tournament = {
  id: number;
  game: string;
  name: string;
  description?: string;
  startDate?: string;
  endDate?: string;
  status: string;
  isListed: boolean;
};

async function fetchTournaments(game: string) {
  const baseUrl = process.env.API_URL ?? process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
  const url = `${baseUrl}/api/v1/tournaments?game=${encodeURIComponent(game)}`;
  try {
    const res = await fetch(url, { cache: "no-store" });
    if (!res.ok) {
      return [] as Tournament[];
    }
    const payload = (await res.json()) as Tournament[] | null;
    return Array.isArray(payload) ? payload : [];
  } catch {
    return [] as Tournament[];
  }
}

function formatDateRange(t: Tournament) {
  if (!t.startDate && !t.endDate) {
    return "Дата проведения ещё не определена";
  }
  if (t.startDate && t.endDate) {
    return `${t.startDate} – ${t.endDate}`;
  }
  if (t.startDate) {
    return `с ${t.startDate}`;
  }
  return `по ${t.endDate}`;
}

export default async function HomePage() {
  const [dota, cs2] = await Promise.all([
    fetchTournaments("DOTA2"),
    fetchTournaments("CS2")
  ]);

  const dotaVisible = dota.filter((t) => t.isListed && t.status === "RUNNING");
  const cs2Visible = cs2.filter((t) => t.isListed && t.status === "RUNNING");

  return (
    <main className="min-h-screen page-shell">
      <div className="mx-auto max-w-6xl px-6 py-12">
        <header className="mb-10 flex flex-wrap items-end justify-between gap-4">
          <div>
            <p className="text-xs uppercase tracking-[0.4em] text-[color:var(--text-muted)]">Арена</p>
            <h1 className="text-3xl font-semibold tracking-tight">Турниры</h1>
            <p className="mt-2 text-[color:var(--text-muted)]">Публичные сетки по Dota 2 и CS2</p>
          </div>
          <ThemeToggle />
        </header>

        <div className="grid gap-6 md:grid-cols-2">
          <Card data-game="dota2" className="game-card">
            <div className="p-6">
              <div className="flex items-center justify-between">
                <h2 className="text-xl font-semibold">Dota 2</h2>
                <span className="game-badge">Radiant</span>
              </div>
              {dotaVisible.length === 0 ? (
                <p className="mt-2 text-sm text-[color:var(--text-muted)]">Пока нет активных турниров.</p>
              ) : (
                <ul className="mt-4 space-y-3">
                  {dotaVisible.map((t) => (
                    <li key={t.id}>
                      <Link
                        href={`/tournament/${t.id}`}
                        className="block rounded-md border border-[color:var(--game-border)] bg-[color:var(--panel-bg)] p-3 transition hover:-translate-y-0.5 hover:shadow-md"
                      >
                        <div className="flex items-center justify-between gap-2">
                          <div className="text-sm font-semibold">{t.name}</div>
                          <span className="text-xs underline">Открыть</span>
                        </div>
                        <div className="mt-1 text-xs text-[color:var(--text-muted)]">{formatDateRange(t)}</div>
                      </Link>
                    </li>
                  ))}
                </ul>
              )}
              <div className="mt-6">
                <Pagination page={1} totalPages={1} />
              </div>
            </div>
          </Card>

          <Card data-game="cs2" className="game-card">
            <div className="p-6">
              <div className="flex items-center justify-between">
                <h2 className="text-xl font-semibold">CS2</h2>
                <span className="game-badge">Counter</span>
              </div>
              {cs2Visible.length === 0 ? (
                <p className="mt-2 text-sm text-[color:var(--text-muted)]">Пока нет активных турниров.</p>
              ) : (
                <ul className="mt-4 space-y-3">
                  {cs2Visible.map((t) => (
                    <li key={t.id}>
                      <Link
                        href={`/tournament/${t.id}`}
                        className="block rounded-md border border-[color:var(--game-border)] bg-[color:var(--panel-bg)] p-3 transition hover:-translate-y-0.5 hover:shadow-md"
                      >
                        <div className="flex items-center justify-between gap-2">
                          <div className="text-sm font-semibold">{t.name}</div>
                          <span className="text-xs underline">Открыть</span>
                        </div>
                        <div className="mt-1 text-xs text-[color:var(--text-muted)]">{formatDateRange(t)}</div>
                      </Link>
                    </li>
                  ))}
                </ul>
              )}
              <div className="mt-6">
                <Pagination page={1} totalPages={1} />
              </div>
            </div>
          </Card>
        </div>
      </div>
    </main>
  );
}
