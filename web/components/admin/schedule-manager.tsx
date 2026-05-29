"use client";

import { useEffect, useMemo, useState } from "react";

type ScheduleItem = {
  id: number;
  tournamentId: number;
  matchId: number;
  position: number;
  matchRound: number;
  matchIndex: number;
  status: string;
  bo: number;
  teamAId?: number;
  teamBId?: number;
  teamAName?: string;
  teamBName?: string;
  scoreA: number;
  scoreB: number;
  winnerTeamId?: number;
  sideMode: string;
  teamASide: string;
  teamBSide: string;
};

type ScheduleListResponse = {
  items: ScheduleItem[];
  page: number;
  pageSize: number;
  total: number;
  totalPages: number;
};

const PAGE_SIZE = 20;
const FORFEIT_REASONS = ["CHEATING", "TOXIC_BEHAVIOR", "NO_SHOW", "TECH_ISSUES", "OTHER"] as const;

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

const FORFEIT_REASON_LABELS: Record<string, string> = {
  CHEATING: "Читерство",
  TOXIC_BEHAVIOR: "Токсичное поведение",
  NO_SHOW: "Неявка",
  TECH_ISSUES: "Технические проблемы",
  OTHER: "Другое"
};

function sideOptions(game: string): [string, string] {
  return game === "CS2" ? ["CT", "T"] : ["Radiant", "Dire"];
}

export function ScheduleManager({ tournamentId, game }: { tournamentId: number; game: string }) {
  const [data, setData] = useState<ScheduleListResponse>({ items: [], page: 1, pageSize: PAGE_SIZE, total: 0, totalPages: 1 });
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [message, setMessage] = useState<string | null>(null);
  const [scheduleLoading, setScheduleLoading] = useState<false | "generate" | "regenerate">(false);
  const [reorderList, setReorderList] = useState<ScheduleItem[]>([]);
  const [savedSnapshot, setSavedSnapshot] = useState<number[]>([]);
  const [draggingMatchId, setDraggingMatchId] = useState<number | null>(null);
  const [reorderSaving, setReorderSaving] = useState(false);
  const [sideDraft, setSideDraft] = useState<Record<number, { mode: "RANDOM" | "MANUAL"; teamASide: string; teamBSide: string }>>({});
  const [savingSidesMatchId, setSavingSidesMatchId] = useState<number | null>(null);
  const [savingStatus, setSavingStatus] = useState<{ matchId: number; action: "START" | "PAUSE" | "RESUME" } | null>(null);
  const [resultDraft, setResultDraft] = useState<Record<number, { scoreA: number; scoreB: number; winnerTeamId?: number }>>({});
  const [savingScoreMatchId, setSavingScoreMatchId] = useState<number | null>(null);
  const [finishingMatchId, setFinishingMatchId] = useState<number | null>(null);
  const [forfeitDraft, setForfeitDraft] = useState<Record<number, { winnerTeamId?: number; reason: string }>>({});
  const [savingForfeitMatchId, setSavingForfeitMatchId] = useState<number | null>(null);
  const [compact, setCompact] = useState(false);

  const canPrev = data.page > 1;
  const canNext = data.page < data.totalPages;
  const hasReorderChanges = JSON.stringify(reorderList.map((item) => item.matchId)) !== JSON.stringify(savedSnapshot);
  const currentRows = useMemo(() => data.items, [data.items]);

  const fetchSchedule = async (targetPage: number) => {
    setLoading(true);
    setError(null);
    try {
      const baseUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
      const query = new URLSearchParams({ page: String(targetPage), pageSize: String(PAGE_SIZE) });
      const res = await fetch(`${baseUrl}/api/v1/admin/tournaments/${tournamentId}/schedule?${query.toString()}`, {
        credentials: "include",
        cache: "no-store"
      });
      if (!res.ok) {
        const text = await res.text();
        setError(text || "Не удалось загрузить расписание.");
        return;
      }
      const payload = (await res.json()) as ScheduleListResponse;
      setData(payload);
      setPage(payload.page);
      setSideDraft((prev) => {
        const next = { ...prev };
        for (const item of payload.items) {
          next[item.matchId] = {
            mode: item.sideMode === "MANUAL" ? "MANUAL" : "RANDOM",
            teamASide: item.teamASide,
            teamBSide: item.teamBSide
          };
        }
        return next;
      });
      setResultDraft((prev) => {
        const next = { ...prev };
        for (const item of payload.items) {
          next[item.matchId] = {
            scoreA: item.scoreA ?? 0,
            scoreB: item.scoreB ?? 0,
            winnerTeamId: item.winnerTeamId
          };
        }
        return next;
      });
      setForfeitDraft((prev) => {
        const next = { ...prev };
        for (const item of payload.items) {
          next[item.matchId] = {
            winnerTeamId: undefined,
            reason: ""
          };
        }
        return next;
      });
    } catch {
      setError("Не удалось загрузить расписание.");
    } finally {
      setLoading(false);
    }
  };

  const fetchReorderList = async () => {
    try {
      const baseUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
      const query = new URLSearchParams({ page: "1", pageSize: "500" });
      const res = await fetch(`${baseUrl}/api/v1/admin/tournaments/${tournamentId}/schedule?${query.toString()}`, {
        credentials: "include",
        cache: "no-store"
      });
      if (!res.ok) {
        setReorderList([]);
        setSavedSnapshot([]);
        return;
      }
      const payload = (await res.json()) as ScheduleListResponse;
      setReorderList(payload.items);
      setSavedSnapshot(payload.items.map((item) => item.matchId));
    } catch {
      setReorderList([]);
      setSavedSnapshot([]);
    }
  };

  useEffect(() => {
    setPage(1);
    void fetchSchedule(1);
    void fetchReorderList();
  }, [tournamentId]);

  const generateSchedule = async (overwrite: boolean) => {
    setScheduleLoading(overwrite ? "regenerate" : "generate");
    setError(null);
    setMessage(null);
    try {
      const baseUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
      const res = await fetch(`${baseUrl}/api/v1/admin/tournaments/${tournamentId}/schedule/generate`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ overwrite })
      });
      if (!res.ok) {
        const text = await res.text();
        setError(text || "Не удалось сгенерировать расписание.");
        return;
      }
      setMessage(overwrite ? "Расписание перегенерировано." : "Расписание сгенерировано.");
      await fetchSchedule(page);
      await fetchReorderList();
    } catch {
      setError("Не удалось сгенерировать расписание.");
    } finally {
      setScheduleLoading(false);
    }
  };

  const moveReorderItem = (draggedMatchId: number, targetMatchId: number) => {
    if (draggedMatchId === targetMatchId) {
      return;
    }
    setReorderList((prev) => {
      const fromIndex = prev.findIndex((item) => item.matchId === draggedMatchId);
      const toIndex = prev.findIndex((item) => item.matchId === targetMatchId);
      if (fromIndex < 0 || toIndex < 0) {
        return prev;
      }
      const next = [...prev];
      const [moved] = next.splice(fromIndex, 1);
      next.splice(toIndex, 0, moved);
      return next;
    });
  };

  const saveReorder = async () => {
    if (!hasReorderChanges) {
      return;
    }

    setReorderSaving(true);
    setError(null);
    setMessage(null);
    try {
      const baseUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
      const res = await fetch(`${baseUrl}/api/v1/admin/tournaments/${tournamentId}/schedule/reorder`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ matchIds: reorderList.map((item) => item.matchId) })
      });
      if (!res.ok) {
        const text = await res.text();
        setError(text || "Не удалось сохранить порядок расписания.");
        return;
      }

      setSavedSnapshot(reorderList.map((item) => item.matchId));
      setMessage("Порядок расписания сохранён.");
      await fetchSchedule(page);
      await fetchReorderList();
    } catch {
      setError("Не удалось сохранить порядок расписания.");
    } finally {
      setReorderSaving(false);
    }
  };

  const saveSides = async (matchId: number) => {
    const draft = sideDraft[matchId];
    if (!draft) return;

    setSavingSidesMatchId(matchId);
    setError(null);
    setMessage(null);
    try {
      const baseUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
      const res = await fetch(`${baseUrl}/api/v1/admin/tournaments/${tournamentId}/matches/${matchId}/sides`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify(draft)
      });
      if (!res.ok) {
        const text = await res.text();
        setError(text || "Не удалось сохранить стороны.");
        return;
      }
      setMessage(`Стороны сохранены для матча ${matchId}.`);
      await fetchSchedule(page);
      await fetchReorderList();
    } catch {
      setError("Не удалось сохранить стороны.");
    } finally {
      setSavingSidesMatchId(null);
    }
  };

  const updateMatchStatus = async (matchId: number, action: "START" | "PAUSE" | "RESUME") => {
    setSavingStatus({ matchId, action });
    setError(null);
    setMessage(null);
    try {
      const baseUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
      const res = await fetch(`${baseUrl}/api/v1/admin/tournaments/${tournamentId}/matches/${matchId}/status`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ action })
      });
      if (!res.ok) {
        const text = await res.text();
        setError(text || "Не удалось обновить статус матча.");
        return;
      }
      setMessage(`Матч ${matchId} обновлён.`);
      await fetchSchedule(page);
      await fetchReorderList();
    } catch {
      setError("Не удалось обновить статус матча.");
    } finally {
      setSavingStatus(null);
    }
  };

  const saveScore = async (matchId: number) => {
    const draft = resultDraft[matchId];
    if (!draft) return;

    setSavingScoreMatchId(matchId);
    setError(null);
    setMessage(null);
    try {
      const baseUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
      const res = await fetch(`${baseUrl}/api/v1/admin/tournaments/${tournamentId}/matches/${matchId}/score`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({
          scoreA: draft.scoreA,
          scoreB: draft.scoreB
        })
      });
      if (!res.ok) {
        const text = await res.text();
        setError(text || "Не удалось сохранить счёт.");
        return;
      }
      setMessage(`Счёт сохранён для матча ${matchId}. Матч на паузе.`);
      await fetchSchedule(page);
      await fetchReorderList();
    } catch {
      setError("Не удалось сохранить счёт.");
    } finally {
      setSavingScoreMatchId(null);
    }
  };

  const finishMatch = async (matchId: number) => {
    const draft = resultDraft[matchId];
    if (!draft) return;

    setFinishingMatchId(matchId);
    setError(null);
    setMessage(null);
    try {
      const baseUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
      const res = await fetch(`${baseUrl}/api/v1/admin/tournaments/${tournamentId}/matches/${matchId}/result`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({
          scoreA: draft.scoreA,
          scoreB: draft.scoreB,
          winnerTeamId: draft.winnerTeamId ?? 0
        })
      });
      if (!res.ok) {
        const text = await res.text();
        setError(text || "Не удалось завершить матч.");
        return;
      }
      setMessage(`Матч ${matchId} завершён.`);
      await fetchSchedule(page);
      await fetchReorderList();
    } catch {
      setError("Не удалось завершить матч.");
    } finally {
      setFinishingMatchId(null);
    }
  };

  const saveForfeit = async (matchId: number) => {
    const draft = forfeitDraft[matchId];
    if (!draft || !draft.winnerTeamId || !draft.reason) return;

    setSavingForfeitMatchId(matchId);
    setError(null);
    setMessage(null);
    try {
      const baseUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
      const res = await fetch(`${baseUrl}/api/v1/admin/tournaments/${tournamentId}/matches/${matchId}/forfeit`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({
          winnerTeamId: draft.winnerTeamId,
          reason: draft.reason
        })
      });
      if (!res.ok) {
        const text = await res.text();
        setError(text || "Не удалось применить тех. победу.");
        return;
      }
      setMessage(`Матчу ${matchId} присвоена тех. победа.`);
      await fetchSchedule(page);
      await fetchReorderList();
    } catch {
      setError("Не удалось применить тех. победу.");
    } finally {
      setSavingForfeitMatchId(null);
    }
  };

  const requiredWins = (bo: number) => {
    if (!bo || bo < 1 || bo % 2 === 0) return 1;
    return Math.floor(bo / 2) + 1;
  };

  const validateScoreDraft = (
    item: ScheduleItem,
    draft: { scoreA: number; scoreB: number; winnerTeamId?: number }
  ): string => {
    if (!item.teamAId || !item.teamBId) return "Матч должен иметь обе команды.";
    if (item.status === "CANCELED") return "Матч отменён.";
    if (item.status === "FINISHED") return "Матч уже завершён.";
    if (draft.scoreA < 0 || draft.scoreB < 0) return "Счёт не может быть отрицательным.";
    const wins = requiredWins(item.bo);
    if (draft.scoreA > wins || draft.scoreB > wins) return `Счёт не может превышать ${wins} для BO${item.bo}.`;
    return "";
  };

  const validateFinishDraft = (
    item: ScheduleItem,
    draft: { scoreA: number; scoreB: number; winnerTeamId?: number }
  ): string => {
    const base = validateScoreDraft(item, draft);
    if (base) return base;
    if (draft.scoreA === draft.scoreB) return "Счёт не может быть равным.";
    const wins = requiredWins(item.bo);
    const maxScore = Math.max(draft.scoreA, draft.scoreB);
    if (maxScore !== wins) return `Чтобы завершить матч, победитель должен иметь ${wins} побед(ы) для BO${item.bo}.`;
    if (!draft.winnerTeamId) return "Выберите победителя.";
    const expectedWinner = draft.scoreA > draft.scoreB ? item.teamAId : item.teamBId;
    if (draft.winnerTeamId !== expectedWinner) return "Победитель должен соответствовать большему счёту.";
    return "";
  };

  const statusReason = (status: string, action: "START" | "PAUSE" | "RESUME") => {
    if (action === "START") {
      if (status === "LIVE") return "Матч уже идёт.";
      if (status === "PAUSED") return "Матч на паузе; используйте «Продолжить».";
      if (status === "FINISHED") return "Матч уже завершён.";
      if (status === "CANCELED") return "Матч отменён.";
      if (status !== "SCHEDULED") return "Матч должен быть запланирован для старта.";
    }
    if (action === "PAUSE") {
      if (status !== "LIVE") return "Матч должен быть LIVE для паузы.";
    }
    if (action === "RESUME") {
      if (status !== "PAUSED") return "Матч должен быть на паузе для продолжения.";
    }
    return "";
  };

  const [firstSide, secondSide] = sideOptions(game);
  const headerCellClass = compact ? "px-2 py-1 text-[10px]" : "px-3 py-2";
  const cellClass = compact ? "px-2 py-1 text-xs" : "px-3 py-2";
  const rowGapClass = compact ? "gap-1" : "gap-2";
  const inputClass = compact ? "px-1 py-0.5 text-[10px]" : "px-2 py-1 text-[11px]";

  return (
    <section className="mt-6 rounded-xl border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] p-4">
      <h3 className="text-base font-semibold">Расписание</h3>
      <p className="mt-1 text-xs text-[color:var(--text-muted)]">Порядок матчей не зависит от сетки.</p>

      <div className="mt-3 flex flex-wrap items-center gap-2">
        <button
          type="button"
          disabled={scheduleLoading !== false || reorderSaving}
          onClick={() => void generateSchedule(false)}
          className="rounded-md border border-[color:var(--panel-border)] px-3 py-1 text-xs hover:bg-[color:var(--panel-bg)] disabled:opacity-50"
        >
          {scheduleLoading === "generate" ? "Генерация..." : "Сгенерировать расписание"}
        </button>
        <button
          type="button"
          disabled={scheduleLoading !== false || reorderSaving}
          onClick={() => void generateSchedule(true)}
          className="rounded-md border border-[color:var(--panel-border)] px-3 py-1 text-xs hover:bg-[color:var(--panel-bg)] disabled:opacity-50"
        >
          {scheduleLoading === "regenerate" ? "Перегенерация..." : "Перегенерировать расписание"}
        </button>
        <label className="ml-auto flex items-center gap-2 text-xs text-[color:var(--text-muted)]">
          <input type="checkbox" checked={compact} onChange={(event) => setCompact(event.target.checked)} />
          Компактный режим
        </label>
      </div>

      <div className="mt-3 rounded-md border border-[color:var(--panel-border)] p-3">
        <div className="flex items-center justify-between">
          <p className="text-xs text-[color:var(--text-muted)]">Порядок (перетаскивание)</p>
          <button
            type="button"
            disabled={!hasReorderChanges || reorderSaving || scheduleLoading !== false}
            onClick={() => void saveReorder()}
            className="rounded-md border border-[color:var(--panel-border)] px-3 py-1 text-xs hover:bg-[color:var(--panel-bg)] disabled:opacity-50"
          >
            {reorderSaving ? "Сохранение порядка..." : "Сохранить порядок"}
          </button>
        </div>

        {reorderList.length === 0 ? (
          <p className="mt-2 text-xs text-[color:var(--text-muted)]">Расписания пока нет. Сначала сгенерируйте расписание.</p>
        ) : (
          <ul className="mt-2 space-y-1">
            {reorderList.map((item, index) => (
              <li
                key={item.matchId}
                draggable
                onDragStart={() => setDraggingMatchId(item.matchId)}
                onDragOver={(event) => event.preventDefault()}
                onDrop={() => {
                  if (draggingMatchId !== null) {
                    moveReorderItem(draggingMatchId, item.matchId);
                  }
                  setDraggingMatchId(null);
                }}
                onDragEnd={() => setDraggingMatchId(null)}
                className={`cursor-grab rounded-md border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)]/40 text-[color:var(--foreground)] ${
                  compact ? "px-2 py-0.5 text-[10px]" : "px-2 py-1 text-xs"
                }`}
              >
                #{index + 1} Матч {item.matchId} (R{item.matchRound} / M{item.matchIndex})
              </li>
            ))}
          </ul>
        )}
      </div>

      {error ? <p className="mt-2 text-xs text-red-400">{error}</p> : null}
      {message ? <p className="mt-2 text-xs text-emerald-300">{message}</p> : null}

      <div className="mt-4 text-xs text-[color:var(--text-muted)]">Всего матчей в расписании: {data.total}</div>
      {loading ? <p className="mt-2 text-sm text-[color:var(--text-muted)]">Загрузка...</p> : null}

      {!loading && currentRows.length > 0 ? (
        <div className="mt-3 overflow-x-auto">
          <table className="min-w-full text-left text-sm">
            <thead className="uppercase tracking-wide text-[color:var(--text-muted)]">
              <tr>
                <th className={headerCellClass}>Позиция</th>
                <th className={headerCellClass}>Матч ID</th>
                <th className={headerCellClass}>Раунд</th>
                <th className={headerCellClass}>Индекс</th>
                <th className={headerCellClass}>Статус</th>
                <th className={headerCellClass}>Стороны</th>
                <th className={headerCellClass}>Действия</th>
              </tr>
            </thead>
            <tbody>
              {currentRows.map((item) => {
                const draft = sideDraft[item.matchId] ?? {
                  mode: "RANDOM" as const,
                  teamASide: firstSide,
                  teamBSide: secondSide
                };
                const result = resultDraft[item.matchId] ?? {
                  scoreA: 0,
                  scoreB: 0,
                  winnerTeamId: undefined
                };
                const forfeit = forfeitDraft[item.matchId] ?? {
                  winnerTeamId: undefined,
                  reason: ""
                };
                const canStart = item.status === "SCHEDULED";
                const canPause = item.status === "LIVE";
                const canResume = item.status === "PAUSED";
                const isSavingStatus = savingStatus?.matchId === item.matchId;
                const hasTeams = item.teamAId !== undefined && item.teamBId !== undefined;
                const scoreValidation = validateScoreDraft(item, result);
                const finishValidation = validateFinishDraft(item, result);
                const canSaveScore = scoreValidation === "" && hasTeams;
                const canFinish = finishValidation === "" && hasTeams;
                const canForfeit = hasTeams && item.status !== "FINISHED" && item.status !== "CANCELED";
                return (
                  <tr key={item.id} className="border-t border-[color:var(--panel-border)]">
                    <td className={`${cellClass} text-[color:var(--text-muted)]`}>{item.position}</td>
                    <td className={`${cellClass} text-[color:var(--text-muted)]`}>{item.matchId}</td>
                    <td className={`${cellClass} text-[color:var(--text-muted)]`}>{item.matchRound}</td>
                    <td className={`${cellClass} text-[color:var(--text-muted)]`}>{item.matchIndex}</td>
                    <td className={`${cellClass} text-[color:var(--text-muted)]`}>{MATCH_STATUS_LABELS[item.status] ?? item.status}</td>
                    <td className={`${cellClass} text-[color:var(--text-muted)]`}>
                      {item.teamAName ?? `Команда ${item.teamAId ?? "A"}`} — {item.teamASide} ·{" "}
                      {item.teamBName ?? `Команда ${item.teamBId ?? "B"}`} — {item.teamBSide} (
                      {SIDE_MODE_LABELS[item.sideMode] ?? item.sideMode})
                    </td>
                    <td className={cellClass}>
                      <div className={`flex flex-col ${rowGapClass}`}>
                        <div className={`flex flex-wrap items-center ${rowGapClass}`}>
                          <button
                            type="button"
                            disabled={!canStart || isSavingStatus}
                            onClick={() => void updateMatchStatus(item.matchId, "START")}
                            title={!canStart ? statusReason(item.status, "START") : ""}
                            className={`rounded-md border border-[color:var(--panel-border)] hover:bg-[color:var(--panel-bg)] disabled:opacity-50 ${compact ? "px-1.5 py-0.5 text-[10px]" : "px-2 py-1 text-[11px]"}`}
                          >
                            {isSavingStatus && savingStatus?.action === "START" ? "Запуск..." : "Старт"}
                          </button>
                          <button
                            type="button"
                            disabled={!canPause || isSavingStatus}
                            onClick={() => void updateMatchStatus(item.matchId, "PAUSE")}
                            title={!canPause ? statusReason(item.status, "PAUSE") : ""}
                            className={`rounded-md border border-[color:var(--panel-border)] hover:bg-[color:var(--panel-bg)] disabled:opacity-50 ${compact ? "px-1.5 py-0.5 text-[10px]" : "px-2 py-1 text-[11px]"}`}
                          >
                            {isSavingStatus && savingStatus?.action === "PAUSE" ? "Пауза..." : "Пауза"}
                          </button>
                          <button
                            type="button"
                            disabled={!canResume || isSavingStatus}
                            onClick={() => void updateMatchStatus(item.matchId, "RESUME")}
                            title={!canResume ? statusReason(item.status, "RESUME") : ""}
                            className={`rounded-md border border-[color:var(--panel-border)] hover:bg-[color:var(--panel-bg)] disabled:opacity-50 ${compact ? "px-1.5 py-0.5 text-[10px]" : "px-2 py-1 text-[11px]"}`}
                          >
                            {isSavingStatus && savingStatus?.action === "RESUME" ? "Возобновление..." : "Продолжить"}
                          </button>
                        </div>
                        <div className={`flex flex-wrap items-center text-[color:var(--text-muted)] ${rowGapClass} ${compact ? "text-[10px]" : "text-[11px]"}`}>
                          <span className="text-[color:var(--text-muted)]">Результат</span>
                          <span className="text-[color:var(--text-muted)]">
                            {item.teamAName ?? `Команда ${item.teamAId ?? "A"}`} / {item.teamBName ?? `Команда ${item.teamBId ?? "B"}`}
                          </span>
                          <input
                            type="number"
                            min={0}
                            value={result.scoreA}
                            onChange={(event) =>
                              setResultDraft((prev) => ({
                                ...prev,
                                [item.matchId]: { ...result, scoreA: Number(event.target.value) }
                              }))
                            }
                            className={`w-14 rounded-md border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] text-[color:var(--foreground)] ${inputClass}`}
                          />
                          <span className="text-[color:var(--text-muted)]">:</span>
                          <input
                            type="number"
                            min={0}
                            value={result.scoreB}
                            onChange={(event) =>
                              setResultDraft((prev) => ({
                                ...prev,
                                [item.matchId]: { ...result, scoreB: Number(event.target.value) }
                              }))
                            }
                            className={`w-14 rounded-md border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] text-[color:var(--foreground)] ${inputClass}`}
                          />
                          <select
                            value={result.winnerTeamId ?? ""}
                            onChange={(event) =>
                              setResultDraft((prev) => ({
                                ...prev,
                                [item.matchId]: {
                                  ...result,
                                  winnerTeamId: event.target.value ? Number(event.target.value) : undefined
                                }
                              }))
                            }
                            className={`min-w-[140px] rounded-md border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] text-[color:var(--foreground)] ${inputClass}`}
                          >
                            <option value="">Выберите победителя</option>
                            {item.teamAId !== undefined ? (
                              <option value={item.teamAId}>{item.teamAName ?? `Команда ${item.teamAId}`}</option>
                            ) : null}
                            {item.teamBId !== undefined ? (
                              <option value={item.teamBId}>{item.teamBName ?? `Команда ${item.teamBId}`}</option>
                            ) : null}
                          </select>
                          <button
                            type="button"
                            disabled={!canSaveScore || savingScoreMatchId === item.matchId}
                            onClick={() => void saveScore(item.matchId)}
                            title={!canSaveScore ? scoreValidation : ""}
                            className={`rounded-md border border-[color:var(--panel-border)] hover:bg-[color:var(--panel-bg)] disabled:opacity-50 ${compact ? "px-1.5 py-0.5 text-[10px]" : "px-2 py-1 text-[11px]"}`}
                          >
                            {savingScoreMatchId === item.matchId ? "Сохранение..." : "Сохранить счёт"}
                          </button>
                          <button
                            type="button"
                            disabled={!canFinish || finishingMatchId === item.matchId}
                            onClick={() => void finishMatch(item.matchId)}
                            title={!canFinish ? finishValidation : ""}
                            className={`rounded-md border border-[color:var(--panel-border)] hover:bg-[color:var(--panel-bg)] disabled:opacity-50 ${compact ? "px-1.5 py-0.5 text-[10px]" : "px-2 py-1 text-[11px]"}`}
                          >
                            {finishingMatchId === item.matchId ? "Завершение..." : "Завершить матч"}
                          </button>
                          {scoreValidation ? <span className={`${compact ? "text-[10px]" : "text-[11px]"} text-amber-300`}>{scoreValidation}</span> : null}
                        </div>
                        <div className={`flex flex-wrap items-center text-[color:var(--text-muted)] ${rowGapClass} ${compact ? "text-[10px]" : "text-[11px]"}`}>
                          <span className="text-[color:var(--text-muted)]">Тех. победа</span>
                          <span className="text-[color:var(--text-muted)]">
                            {item.teamAName ?? `Команда ${item.teamAId ?? "A"}`} / {item.teamBName ?? `Команда ${item.teamBId ?? "B"}`}
                          </span>
                          <select
                            value={forfeit.winnerTeamId ?? ""}
                            onChange={(event) =>
                              setForfeitDraft((prev) => ({
                                ...prev,
                                [item.matchId]: {
                                  ...forfeit,
                                  winnerTeamId: event.target.value ? Number(event.target.value) : undefined
                                }
                              }))
                            }
                            className={`min-w-[140px] rounded-md border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] text-[color:var(--foreground)] ${inputClass}`}
                          >
                            <option value="">Выберите победителя</option>
                            {item.teamAId !== undefined ? (
                              <option value={item.teamAId}>{item.teamAName ?? `Команда ${item.teamAId}`}</option>
                            ) : null}
                            {item.teamBId !== undefined ? (
                              <option value={item.teamBId}>{item.teamBName ?? `Команда ${item.teamBId}`}</option>
                            ) : null}
                          </select>
                          <select
                            value={forfeit.reason}
                            onChange={(event) =>
                              setForfeitDraft((prev) => ({
                                ...prev,
                                [item.matchId]: { ...forfeit, reason: event.target.value }
                              }))
                            }
                            className={`min-w-[140px] rounded-md border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] text-[color:var(--foreground)] ${inputClass}`}
                          >
                            <option value="">Выберите причину</option>
                            {FORFEIT_REASONS.map((reason) => (
                              <option key={reason} value={reason}>
                                {FORFEIT_REASON_LABELS[reason] ?? reason}
                              </option>
                            ))}
                          </select>
                          <button
                            type="button"
                            disabled={!canForfeit || !forfeit.winnerTeamId || !forfeit.reason || savingForfeitMatchId === item.matchId}
                            onClick={() => void saveForfeit(item.matchId)}
                            className={`rounded-md border border-[color:var(--panel-border)] hover:bg-[color:var(--panel-bg)] disabled:opacity-50 ${compact ? "px-1.5 py-0.5 text-[10px]" : "px-2 py-1 text-[11px]"}`}
                          >
                            {savingForfeitMatchId === item.matchId ? "Сохранение..." : "Тех. победа"}
                          </button>
                        </div>
                        <div className={`flex flex-wrap items-center ${rowGapClass}`}>
                          <button
                            type="button"
                            onClick={() =>
                              setSideDraft((prev) => ({
                                ...prev,
                                [item.matchId]: { ...draft, mode: "RANDOM" }
                              }))
                            }
                            className={`rounded-md border border-[color:var(--panel-border)] hover:bg-[color:var(--panel-bg)] ${compact ? "px-1.5 py-0.5 text-[10px]" : "px-2 py-1 text-[11px]"}`}
                          >
                            Случайно
                          </button>
                          <button
                            type="button"
                            onClick={() =>
                              setSideDraft((prev) => ({
                                ...prev,
                                [item.matchId]: { ...draft, mode: "MANUAL" }
                              }))
                            }
                            className={`rounded-md border border-[color:var(--panel-border)] hover:bg-[color:var(--panel-bg)] ${compact ? "px-1.5 py-0.5 text-[10px]" : "px-2 py-1 text-[11px]"}`}
                          >
                            Вручную
                          </button>
                          <select
                            value={draft.teamASide}
                            onChange={(event) =>
                              setSideDraft((prev) => ({
                                ...prev,
                                [item.matchId]: { ...draft, teamASide: event.target.value }
                              }))
                            }
                            className={`rounded-md border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] text-[color:var(--foreground)] ${inputClass}`}
                          >
                            <option value={firstSide}>{firstSide}</option>
                            <option value={secondSide}>{secondSide}</option>
                          </select>
                          <span className={`${compact ? "text-[10px]" : "text-[11px]"} text-[color:var(--text-muted)]`}>
                            {item.teamAName ?? `Команда ${item.teamAId ?? "A"}`}
                          </span>
                          <span className={`${compact ? "text-[10px]" : "text-[11px]"} text-[color:var(--text-muted)]`}>/</span>
                          <select
                            value={draft.teamBSide}
                            onChange={(event) =>
                              setSideDraft((prev) => ({
                                ...prev,
                                [item.matchId]: { ...draft, teamBSide: event.target.value }
                              }))
                            }
                            className={`rounded-md border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] text-[color:var(--foreground)] ${inputClass}`}
                          >
                            <option value={firstSide}>{firstSide}</option>
                            <option value={secondSide}>{secondSide}</option>
                          </select>
                          <span className={`${compact ? "text-[10px]" : "text-[11px]"} text-[color:var(--text-muted)]`}>
                            {item.teamBName ?? `Команда ${item.teamBId ?? "B"}`}
                          </span>
                          <button
                            type="button"
                            disabled={savingSidesMatchId === item.matchId}
                            onClick={() => void saveSides(item.matchId)}
                            className={`rounded-md border border-[color:var(--panel-border)] hover:bg-[color:var(--panel-bg)] disabled:opacity-50 ${compact ? "px-1.5 py-0.5 text-[10px]" : "px-2 py-1 text-[11px]"}`}
                          >
                            {savingSidesMatchId === item.matchId ? "Сохранение..." : "Сохранить стороны"}
                          </button>
                        </div>
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      ) : null}

      <div className="mt-4 flex items-center justify-between text-xs text-[color:var(--text-muted)]">
        <button
          type="button"
          disabled={!canPrev}
          onClick={() => void fetchSchedule(page - 1)}
          className="rounded-md border border-[color:var(--panel-border)] px-3 py-1 hover:bg-[color:var(--panel-bg)] disabled:opacity-50"
        >
          Назад
        </button>
        <span>
          Страница {data.page} из {data.totalPages}
        </span>
        <button
          type="button"
          disabled={!canNext}
          onClick={() => void fetchSchedule(page + 1)}
          className="rounded-md border border-[color:var(--panel-border)] px-3 py-1 hover:bg-[color:var(--panel-bg)] disabled:opacity-50"
        >
          Вперёд
        </button>
      </div>
    </section>
  );
}
