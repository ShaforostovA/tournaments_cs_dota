"use client";

import { useState } from "react";

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

const MATCH_STATUS_LABELS: Record<string, string> = {
  SCHEDULED: "Запланирован",
  LIVE: "Идёт",
  PAUSED: "Пауза",
  FINISHED: "Завершён",
  CANCELED: "Отменён"
};

const SIDE_MODE_LABELS: Record<string, string> = {
  RANDOM: "Случайно",
  MANUAL: "Вручную"
};

function teamNameOrTBD(name?: string) {
  return name && name.trim() ? name : "Не определено";
}

export function SchedulePanel({ schedule }: { schedule: ScheduleResponse | null }) {
  const [compact, setCompact] = useState(false);
  const itemClass = compact ? "px-2 py-1 text-[10px]" : "p-2 text-xs";

  return (
    <aside className="game-panel p-3">
      <div className="flex items-center justify-between gap-2">
        <h3 className="text-sm font-semibold text-[color:var(--foreground)]">Расписание</h3>
        <label className="flex items-center gap-2 text-[11px] text-[color:var(--text-muted)]">
          <input type="checkbox" checked={compact} onChange={(event) => setCompact(event.target.checked)} />
          Компактно
        </label>
      </div>

      {!schedule || schedule.items.length === 0 ? (
        <p className="mt-2 text-xs text-[color:var(--text-muted)]">Нет видимых матчей в расписании.</p>
      ) : (
        <ul className="mt-3 space-y-2">
          {schedule.items.map((item) => (
            <li
              key={item.matchId}
              className={`rounded-md border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] ${itemClass}`}
            >
              <div className="flex items-center justify-between text-[11px] text-[color:var(--text-muted)]">
                <span>#{item.position}</span>
                <span>{MATCH_STATUS_LABELS[item.status] ?? item.status}</span>
              </div>
              <div className="mt-1 text-[color:var(--foreground)]">
                {teamNameOrTBD(item.teamAName)} vs {teamNameOrTBD(item.teamBName)}
              </div>
              <div className="mt-1 text-[color:var(--text-muted)]">
                Счёт: {teamNameOrTBD(item.teamAName)} {item.scoreA} — {item.scoreB} {teamNameOrTBD(item.teamBName)}
              </div>
              <div className="mt-1 text-[color:var(--text-muted)]">
                Стороны ({SIDE_MODE_LABELS[item.sideMode] ?? item.sideMode}): {teamNameOrTBD(item.teamAName)} — {item.teamASide},{" "}
                {teamNameOrTBD(item.teamBName)} — {item.teamBSide}
              </div>
            </li>
          ))}
        </ul>
      )}
    </aside>
  );
}
