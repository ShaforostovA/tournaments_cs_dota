import Link from "next/link";
import { TournamentRealtimeRefresh } from "@/components/realtime/tournament-realtime-refresh";
import { ThemeToggle } from "@/components/theme-toggle";
import { SchedulePanel } from "@/components/tournament/schedule-panel";

type Tournament = {
  id: number;
  game: string;
  name: string;
  description?: string;
  startDate?: string;
  endDate?: string;
  status: string;
  isListed: boolean;
  isBracketPublished: boolean;
};

type BracketMatch = {
  id: number;
  round: number;
  indexInRound: number;
  status: string;
  bo: number;
  disputeRule: string;
  teamAId?: number;
  teamAName?: string;
  teamBId?: number;
  teamBName?: string;
  scoreA: number;
  scoreB: number;
  winnerTeamId?: number;
};

type BracketRound = {
  round: number;
  matches: BracketMatch[];
};

type BracketResponse = {
  tournamentId: number;
  rounds: BracketRound[];
};

type ScheduleItem = {
  position: number;
  matchId: number;
  round: number;
  indexInRound: number;
  status: string;
  teamAName?: string;
  teamBName?: string;
  scoreA: number;
  scoreB: number;
  sideMode: string;
  teamASide: string;
  teamBSide: string;
};

type ScheduleResponse = {
  tournamentId: number;
  totalVisible: number;
  items: ScheduleItem[];
};

async function fetchTournament(id: string) {
  const baseUrl = process.env.API_URL ?? process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
  const res = await fetch(`${baseUrl}/api/v1/tournaments/${id}`, { cache: "no-store" });
  if (!res.ok) return null;
  return (await res.json()) as Tournament;
}

async function fetchBracket(id: string) {
  const baseUrl = process.env.API_URL ?? process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
  const res = await fetch(`${baseUrl}/api/v1/tournaments/${id}/bracket`, { cache: "no-store" });
  if (!res.ok) return null;
  return (await res.json()) as BracketResponse;
}

async function fetchSchedule(id: string) {
  const baseUrl = process.env.API_URL ?? process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
  const res = await fetch(`${baseUrl}/api/v1/tournaments/${id}/schedule`, { cache: "no-store" });
  if (!res.ok) return null;
  return (await res.json()) as ScheduleResponse;
}

function formatDateRange(t: Tournament) {
  if (!t.startDate && !t.endDate) return "Дата проведения не указана";
  if (t.startDate && t.endDate) return `${t.startDate} – ${t.endDate}`;
  if (t.startDate) return `с ${t.startDate}`;
  return `по ${t.endDate}`;
}

function teamNameOrTBD(name?: string) {
  return name && name.trim() ? name : "Не определено";
}

function matchStatusLabel(status: string) {
  if (status === "SCHEDULED") return "Запланирован";
  if (status === "LIVE") return "Идёт";
  if (status === "PAUSED") return "Пауза";
  if (status === "FINISHED") return "Завершён";
  if (status === "CANCELED") return "Отменён";
  return status;
}

export default async function TournamentPage({ params }: { params: { id: string } }) {
  const tournament = await fetchTournament(params.id);
  const tournamentID = Number.parseInt(params.id, 10);
  const gameTheme = tournament?.game === "CS2" ? "cs2" : "dota2";

  if (!tournament) {
    return (
      <main className="min-h-screen page-shell">
        <div className="mx-auto max-w-4xl px-6 py-12">
          <h1 className="text-2xl font-semibold">Турнир не найден</h1>
          <p className="mt-2 text-sm text-[color:var(--text-muted)]">Проверьте ссылку и попробуйте снова.</p>
          <Link className="mt-6 inline-block text-sm underline" href="/">
            На главную
          </Link>
        </div>
      </main>
    );
  }

  const bracket = tournament.isBracketPublished ? await fetchBracket(params.id) : null;
  const schedule = tournament.isBracketPublished ? await fetchSchedule(params.id) : null;

  return (
    <main className="min-h-screen page-shell" data-game={gameTheme}>
      {Number.isNaN(tournamentID) ? null : <TournamentRealtimeRefresh tournamentId={tournamentID} />}
      <div className="mx-auto max-w-6xl px-6 py-12">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <Link className="text-sm text-[color:var(--text-muted)] underline" href="/">
            Назад
          </Link>
          <ThemeToggle />
        </div>

        <div className="mt-6 game-panel p-6">
          <h1 className="text-2xl font-semibold">{tournament.name}</h1>
          <p className="mt-1 text-sm text-[color:var(--text-muted)]">{formatDateRange(tournament)}</p>
          {tournament.description ? <p className="mt-4 text-sm text-[color:var(--text-muted)]">{tournament.description}</p> : null}
        </div>

        {!tournament.isBracketPublished ? (
          <div className="mt-8 game-panel p-6">
            <h2 className="text-lg font-semibold">Сетка ещё не опубликована</h2>
            <p className="mt-2 text-sm text-[color:var(--text-muted)]">Организаторы опубликуют её до или во время матчей.</p>
          </div>
        ) : (
          <div className="mt-8 game-panel p-6">
            <h2 className="text-lg font-semibold">Сетка турнира</h2>
            <div className="mt-4 grid gap-4 lg:grid-cols-[320px_minmax(0,1fr)]">
              <SchedulePanel schedule={schedule} />

              {!bracket || bracket.rounds.length === 0 ? (
                <p className="text-sm text-[color:var(--text-muted)]">Данных по сетке пока нет.</p>
              ) : (
                <div className="overflow-x-auto">
                  <div className="flex min-w-max gap-4">
                    {bracket.rounds.map((round) => (
                      <section key={round.round} className="w-72 game-panel p-3">
                        <h3 className="text-sm font-semibold text-[color:var(--foreground)]">Раунд {round.round}</h3>
                        <div className="mt-3 space-y-3">
                          {round.matches.map((match) => {
                            const isFinished = match.status === "FINISHED";
                            const aWinner = isFinished && match.winnerTeamId !== undefined && match.teamAId === match.winnerTeamId;
                            const bWinner = isFinished && match.winnerTeamId !== undefined && match.teamBId === match.winnerTeamId;
                            const isBye =
                              (match.teamAId !== undefined && match.teamBId === undefined) ||
                              (match.teamBId !== undefined && match.teamAId === undefined);
                            return (
                              <article key={match.id} className="game-panel p-3">
                                <div className="mb-2 flex items-center justify-between text-[11px] text-[color:var(--text-muted)]">
                                  <span>Матч {match.indexInRound}</span>
                                  <span className="flex items-center gap-2">
                                    {isBye ? <span className="game-badge">BYE</span> : null}
                                    <span>{matchStatusLabel(match.status)}</span>
                                  </span>
                                </div>
                                <div
                                  className={`flex items-center justify-between rounded-md px-2 py-1 text-xs ${
                                    aWinner
                                      ? "bg-[color:var(--win-bg)] text-[color:var(--win-text)] ring-1 ring-[color:var(--win-text)]/20"
                                      : "text-[color:var(--foreground)]"
                                  }`}
                                >
                                  <span className="truncate pr-2">{teamNameOrTBD(match.teamAName)}</span>
                                  <span className="font-semibold">{match.scoreA}</span>
                                </div>
                                <div
                                  className={`mt-1 flex items-center justify-between rounded-md px-2 py-1 text-xs ${
                                    bWinner
                                      ? "bg-[color:var(--win-bg)] text-[color:var(--win-text)] ring-1 ring-[color:var(--win-text)]/20"
                                      : "text-[color:var(--foreground)]"
                                  }`}
                                >
                                  <span className="truncate pr-2">{teamNameOrTBD(match.teamBName)}</span>
                                  <span className="font-semibold">{match.scoreB}</span>
                                </div>
                                <div className="mt-2 text-[10px] uppercase tracking-wide text-[color:var(--text-muted)]">
                                  BO{match.bo} · {match.disputeRule}
                                </div>
                              </article>
                            );
                          })}
                        </div>
                      </section>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </main>
  );
}
