"use client";

import Link from "next/link";
import { useEffect, useState } from "react";

type ActiveMatch = {
  matchId: number;
  tournamentId: number;
  tournamentName: string;
  game: string;
  round: number;
  indexInRound: number;
  status: string;
  teamAName?: string;
  teamBName?: string;
  scoreA: number;
  scoreB: number;
  startsAt?: string;
};

type ActiveMatchesResponse = {
  items: ActiveMatch[];
};

const POLL_MS = 5000;

export function ActiveMatches() {
  const [items, setItems] = useState<ActiveMatch[]>([]);
  const [game, setGame] = useState("ALL");
  const [tournamentId, setTournamentId] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchMatches = async () => {
    setLoading(true);
    setError(null);
    try {
      const baseUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
      const params = new URLSearchParams();
      if (game !== "ALL") params.set("game", game);
      if (tournamentId.trim()) params.set("tournamentId", tournamentId.trim());
      const url = `${baseUrl}/api/v1/matches/active${params.toString() ? `?${params.toString()}` : ""}`;
      const res = await fetch(url, { cache: "no-store" });
      if (!res.ok) {
        const text = await res.text();
        setError(text || "Не удалось загрузить активные матчи.");
        return;
      }
      const payload = (await res.json()) as ActiveMatchesResponse;
      setItems(payload.items ?? []);
    } catch {
      setError("Не удалось загрузить активные матчи.");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void fetchMatches();
    const timer = setInterval(() => void fetchMatches(), POLL_MS);
    return () => clearInterval(timer);
  }, [game, tournamentId]);

  return (
    <section className="rounded-xl border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] p-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold">Активные матчи</h1>
          <p className="mt-1 text-sm text-[color:var(--text-muted)]">Обновление каждые {POLL_MS / 1000}с.</p>
        </div>
        <div className="flex flex-wrap items-center gap-2 text-xs">
          <label className="text-[color:var(--text-muted)]">
            Игра
            <select
              value={game}
              onChange={(event) => setGame(event.target.value)}
              className="ml-2 rounded-md border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] px-2 py-1 text-xs text-[color:var(--foreground)]"
            >
              <option value="ALL">Все</option>
              <option value="DOTA2">Dota 2</option>
              <option value="CS2">CS2</option>
            </select>
          </label>
          <label className="text-[color:var(--text-muted)]">
            ID турнира
            <input
              value={tournamentId}
              onChange={(event) => setTournamentId(event.target.value)}
              placeholder="например 12"
              className="ml-2 w-24 rounded-md border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] px-2 py-1 text-xs text-[color:var(--foreground)]"
            />
          </label>
          <button
            type="button"
            onClick={() => void fetchMatches()}
            className="rounded-md border border-[color:var(--panel-border)] px-3 py-1 text-xs text-[color:var(--foreground)] hover:opacity-80"
          >
            Обновить
          </button>
        </div>
      </div>

      {error ? <p className="mt-3 text-sm text-red-400">{error}</p> : null}
      {loading ? <p className="mt-3 text-sm text-[color:var(--text-muted)]">Загрузка...</p> : null}

      {!loading && items.length === 0 ? (
        <p className="mt-4 text-sm text-[color:var(--text-muted)]">Сейчас нет активных матчей.</p>
      ) : null}

      {!loading && items.length > 0 ? (
        <div className="mt-4 grid gap-4 md:grid-cols-2">
          {items.map((match) => (
            <article key={match.matchId} data-game={match.game === "CS2" ? "cs2" : "dota2"} className="game-panel p-4">
              <div className="flex items-center justify-between text-xs text-[color:var(--text-muted)]">
                <span>{match.game}</span>
                <span>
                  Раунд {match.round} · Матч {match.indexInRound}
                </span>
              </div>
              <h3 className="mt-2 text-sm font-semibold">{match.tournamentName}</h3>
              <div className="mt-2 text-sm text-[color:var(--foreground)]">
                {match.teamAName ?? "Не определено"} vs {match.teamBName ?? "Не определено"}
              </div>
              <div className="mt-1 text-xs text-[color:var(--text-muted)]">
                Счёт {match.scoreA}:{match.scoreB}
              </div>
              {match.startsAt ? (
                <div className="mt-1 text-[11px] text-[color:var(--text-muted)]">
                  Начало {new Date(match.startsAt).toLocaleString()}
                </div>
              ) : null}
              <div className="mt-3">
                <Link href={`/tournament/${match.tournamentId}`} className="text-xs text-[color:var(--foreground)] underline">
                  Открыть турнир
                </Link>
              </div>
            </article>
          ))}
        </div>
      ) : null}
    </section>
  );
}
