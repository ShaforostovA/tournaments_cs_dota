"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { TeamManager } from "@/components/admin/team-manager";
import { ScheduleManager } from "@/components/admin/schedule-manager";
import { RoundSettingsManager } from "@/components/admin/round-settings-manager";
import { AuditLog } from "@/components/admin/audit-log";
import { TournamentRealtimeRefresh } from "@/components/realtime/tournament-realtime-refresh";

type Tournament = {
  id: number;
  game: string;
  name: string;
  description?: string;
  startDate?: string;
  endDate?: string;
  allowOdd: boolean;
  status: string;
  isBracketPublished: boolean;
  scheduleVisibilityAhead: string;
};

type EditorState = {
  game: string;
  name: string;
  description: string;
  startDate: string;
  endDate: string;
  allowOdd: boolean;
};

const DEBOUNCE_MS = 700;

const STATUS_LABELS: Record<string, string> = {
  DRAFT: "Черновик",
  READY: "Готов",
  RUNNING: "Идёт",
  FINISHED: "Завершён"
};

function statusLabel(status: string): string {
  return STATUS_LABELS[status] ?? status;
}

function toEditorState(tournament: Tournament): EditorState {
  return {
    game: tournament.game,
    name: tournament.name,
    description: tournament.description ?? "",
    startDate: tournament.startDate ?? "",
    endDate: tournament.endDate ?? "",
    allowOdd: tournament.allowOdd
  };
}

function toPayload(state: EditorState) {
  return {
    game: state.game,
    name: state.name,
    description: state.description,
    startDate: state.startDate,
    endDate: state.endDate,
    allowOdd: state.allowOdd
  };
}

export function TournamentEditor({ tournament }: { tournament: Tournament }) {
  const router = useRouter();
  const initial = useMemo(() => toEditorState(tournament), [tournament]);
  const [form, setForm] = useState<EditorState>(initial);
  const [lastSaved, setLastSaved] = useState<EditorState>(initial);
  const [currentStatus, setCurrentStatus] = useState(tournament.status);
  const [saveState, setSaveState] = useState<"idle" | "saving" | "saved" | "error">("idle");
  const [error, setError] = useState<string | null>(null);
  const [bracketLoading, setBracketLoading] = useState<false | "generate" | "regenerate">(false);
  const [bracketMessage, setBracketMessage] = useState<string | null>(null);
  const [startLoading, setStartLoading] = useState(false);
  const [startMessage, setStartMessage] = useState<string | null>(null);
  const [visibilityLoading, setVisibilityLoading] = useState(false);
  const [visibilityMessage, setVisibilityMessage] = useState<string | null>(null);
  const [isBracketPublished, setIsBracketPublished] = useState(tournament.isBracketPublished);
  const [scheduleVisibilityAhead, setScheduleVisibilityAhead] = useState(tournament.scheduleVisibilityAhead);
  const [deleteLoading, setDeleteLoading] = useState(false);
  const [roundSettingsRefresh, setRoundSettingsRefresh] = useState(0);

  useEffect(() => {
    setForm(initial);
    setLastSaved(initial);
    setCurrentStatus(tournament.status);
    setSaveState("idle");
    setError(null);
    setBracketLoading(false);
    setBracketMessage(null);
    setStartLoading(false);
    setStartMessage(null);
    setVisibilityLoading(false);
    setVisibilityMessage(null);
    setIsBracketPublished(tournament.isBracketPublished);
    setScheduleVisibilityAhead(tournament.scheduleVisibilityAhead);
  }, [initial, tournament.status, tournament.isBracketPublished, tournament.scheduleVisibilityAhead]);

  const isLocked = currentStatus === "RUNNING" || currentStatus === "FINISHED";
  const canEditDates = currentStatus === "DRAFT" || currentStatus === "READY";
  const hasChanges = JSON.stringify(form) !== JSON.stringify(lastSaved);
  const canDelete = (currentStatus === "DRAFT" || currentStatus === "READY") && !isBracketPublished;

  useEffect(() => {
    if (!hasChanges) {
      return;
    }

    const timer = setTimeout(async () => {
      setSaveState("saving");
      setError(null);

      try {
        const baseUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
        const res = await fetch(`${baseUrl}/api/v1/admin/tournaments/${tournament.id}`, {
          method: "PATCH",
          headers: { "Content-Type": "application/json" },
          credentials: "include",
          body: JSON.stringify(toPayload(form))
        });

        if (!res.ok) {
          const text = await res.text();
          setSaveState("error");
          setError(text || "Не удалось сохранить изменения.");
          return;
        }

        const updated = (await res.json()) as Tournament;
        const updatedState = toEditorState(updated);
        setForm(updatedState);
        setLastSaved(updatedState);
        setCurrentStatus(updated.status);
        setSaveState("saved");
      } catch {
        setSaveState("error");
        setError("Не удалось сохранить изменения.");
      }
    }, DEBOUNCE_MS);

    return () => clearTimeout(timer);
  }, [form, hasChanges, tournament.id]);

  const generateBracket = async (overwrite: boolean) => {
    setBracketLoading(overwrite ? "regenerate" : "generate");
    setBracketMessage(null);
    setStartMessage(null);
    setVisibilityMessage(null);
    setError(null);
    try {
      const baseUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
      const res = await fetch(`${baseUrl}/api/v1/admin/tournaments/${tournament.id}/bracket/generate`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ overwrite })
      });
      if (!res.ok) {
        const text = await res.text();
        setError(text || "Не удалось сгенерировать сетку.");
        return;
      }

      const payload = (await res.json()) as {
        bracketSize: number;
        roundsCount: number;
        totalMatches: number;
        byeSlots: number;
        tournamentNow?: string;
      };
      setBracketMessage(
        `Сетка готова: размер=${payload.bracketSize}, раундов=${payload.roundsCount}, матчей=${payload.totalMatches}, bye=${payload.byeSlots}.`
      );
      if (payload.tournamentNow) {
        setCurrentStatus(payload.tournamentNow);
      }
      setRoundSettingsRefresh((prev) => prev + 1);
    } catch {
      setError("Не удалось сгенерировать сетку.");
    } finally {
      setBracketLoading(false);
    }
  };

  const startTournament = async () => {
    setStartLoading(true);
    setStartMessage(null);
    setVisibilityMessage(null);
    setError(null);
    try {
      const baseUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
      const res = await fetch(`${baseUrl}/api/v1/admin/tournaments/${tournament.id}/start`, {
        method: "POST",
        credentials: "include"
      });
      if (!res.ok) {
        const text = await res.text();
        setError(text || "Не удалось запустить турнир.");
        return;
      }
      const updated = (await res.json()) as Tournament;
      setCurrentStatus(updated.status);
      setIsBracketPublished(updated.isBracketPublished);
      setScheduleVisibilityAhead(updated.scheduleVisibilityAhead);
      setStartMessage("Турнир запущен. Публикация включена, сетка остаётся скрытой.");
    } catch {
      setError("Не удалось запустить турнир.");
    } finally {
      setStartLoading(false);
    }
  };

  const saveVisibility = async () => {
    const rawAhead = scheduleVisibilityAhead.trim().toUpperCase();
    if (rawAhead !== "ALL") {
      const num = Number.parseInt(rawAhead, 10);
      if (Number.isNaN(num) || num < 0 || String(num) !== rawAhead) {
        setError("scheduleVisibilityAhead должен быть ALL или неотрицательным числом.");
        return;
      }
    }

    setVisibilityLoading(true);
    setVisibilityMessage(null);
    setStartMessage(null);
    setError(null);
    try {
      const baseUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
      const res = await fetch(`${baseUrl}/api/v1/admin/tournaments/${tournament.id}/visibility`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({
          isBracketPublished,
          scheduleVisibilityAhead: rawAhead
        })
      });
      if (!res.ok) {
        const text = await res.text();
        setError(text || "Не удалось обновить видимость.");
        return;
      }
      const updated = (await res.json()) as Tournament;
      setIsBracketPublished(updated.isBracketPublished);
      setScheduleVisibilityAhead(updated.scheduleVisibilityAhead);
      setVisibilityMessage("Настройки видимости обновлены.");
    } catch {
      setError("Не удалось обновить видимость.");
    } finally {
      setVisibilityLoading(false);
    }
  };

  const deleteTournament = async () => {
    if (!canDelete || deleteLoading) return;
    if (!confirm("Удалить турнир? Это действие нельзя отменить.")) {
      return;
    }

    setDeleteLoading(true);
    setError(null);
    try {
      const baseUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
      const res = await fetch(`${baseUrl}/api/v1/admin/tournaments/${tournament.id}`, {
        method: "DELETE",
        credentials: "include"
      });
      if (!res.ok) {
        const text = await res.text();
        setError(text || "Не удалось удалить турнир.");
        return;
      }
      router.push("/admin");
      router.refresh();
    } catch {
      setError("Не удалось удалить турнир.");
    } finally {
      setDeleteLoading(false);
    }
  };

  return (
    <section className="rounded-xl border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] p-4">
      <TournamentRealtimeRefresh tournamentId={tournament.id} refreshIntervalMs={1200} />
      <h2 className="text-lg font-semibold">Редактор турнира #{tournament.id}</h2>
      <p className="mt-1 text-xs text-[color:var(--text-muted)]">
        Статус: {statusLabel(currentStatus)}. Автосохранение через {DEBOUNCE_MS}мс.
      </p>

      <div className="mt-4 grid gap-3 md:grid-cols-2">
        <label className="text-xs text-[color:var(--text-muted)]">
          Название
          <input
            value={form.name}
            onChange={(event) => setForm((prev) => ({ ...prev, name: event.target.value }))}
            className="mt-1 w-full rounded-md border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] px-2 py-2 text-sm text-[color:var(--foreground)]"
          />
        </label>

        <label className="text-xs text-[color:var(--text-muted)]">
          Игра
          <select
            value={form.game}
            disabled={isLocked}
            onChange={(event) => setForm((prev) => ({ ...prev, game: event.target.value }))}
            className="mt-1 w-full rounded-md border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] px-2 py-2 text-sm text-[color:var(--foreground)] disabled:opacity-50"
          >
            <option value="DOTA2">Dota 2</option>
            <option value="CS2">CS2</option>
          </select>
        </label>

        <label className="text-xs text-[color:var(--text-muted)] md:col-span-2">
          Описание
          <textarea
            value={form.description}
            onChange={(event) => setForm((prev) => ({ ...prev, description: event.target.value }))}
            rows={3}
            className="mt-1 w-full rounded-md border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] px-2 py-2 text-sm text-[color:var(--foreground)]"
          />
        </label>

        <label className="text-xs text-[color:var(--text-muted)]">
          Дата начала
          <input
            type="date"
            value={form.startDate}
            disabled={!canEditDates}
            onChange={(event) => setForm((prev) => ({ ...prev, startDate: event.target.value }))}
            className="mt-1 w-full rounded-md border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] px-2 py-2 text-sm text-[color:var(--foreground)] disabled:opacity-50"
          />
        </label>

        <label className="text-xs text-[color:var(--text-muted)]">
          Дата окончания
          <input
            type="date"
            value={form.endDate}
            disabled={!canEditDates}
            onChange={(event) => setForm((prev) => ({ ...prev, endDate: event.target.value }))}
            className="mt-1 w-full rounded-md border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] px-2 py-2 text-sm text-[color:var(--foreground)] disabled:opacity-50"
          />
        </label>

        <label className="flex items-center gap-2 text-xs text-[color:var(--text-muted)]">
          <input
            type="checkbox"
            checked={form.allowOdd}
            disabled={isLocked}
            onChange={(event) => setForm((prev) => ({ ...prev, allowOdd: event.target.checked }))}
          />
          Разрешить нечётное число команд
        </label>
      </div>

      <div className="mt-3 text-xs text-[color:var(--text-muted)]">
        {saveState === "saving" ? "Сохранение..." : null}
        {saveState === "saved" ? "Сохранено" : null}
        {saveState === "error" ? "Ошибка сохранения" : null}
        {error ? `: ${error}` : null}
      </div>

      <div className="mt-4 flex flex-wrap gap-2">
        <button
          type="button"
          disabled={bracketLoading !== false || startLoading}
          onClick={() => void generateBracket(false)}
          className="rounded-md border border-[color:var(--panel-border)] px-3 py-1 text-xs text-[color:var(--foreground)] hover:opacity-80 disabled:opacity-50"
        >
          {bracketLoading === "generate" ? "Генерация..." : "Сгенерировать сетку"}
        </button>
        <button
          type="button"
          disabled={bracketLoading !== false || startLoading}
          onClick={() => void generateBracket(true)}
          className="rounded-md border border-[color:var(--panel-border)] px-3 py-1 text-xs text-[color:var(--foreground)] hover:opacity-80 disabled:opacity-50"
        >
          {bracketLoading === "regenerate" ? "Регенерация..." : "Перегенерировать сетку"}
        </button>
      </div>
      {bracketMessage ? <p className="mt-2 text-xs text-emerald-300">{bracketMessage}</p> : null}

      <div className="mt-3">
        <button
          type="button"
          disabled={startLoading || currentStatus === "RUNNING" || currentStatus === "FINISHED"}
          onClick={() => void startTournament()}
          className="rounded-md border border-[color:var(--panel-border)] px-3 py-1 text-xs text-[color:var(--foreground)] hover:opacity-80 disabled:opacity-50"
        >
          {startLoading ? "Запуск..." : "Запустить турнир"}
        </button>
      </div>
      {startMessage ? <p className="mt-2 text-xs text-emerald-300">{startMessage}</p> : null}

      <div className="mt-3">
        <button
          type="button"
          disabled={!canDelete || deleteLoading}
          onClick={() => void deleteTournament()}
          className="rounded-md border border-red-400/60 px-3 py-1 text-xs text-red-600 hover:bg-red-50 disabled:opacity-50"
        >
          {deleteLoading ? "Удаление..." : "Удалить турнир"}
        </button>
        {!canDelete ? (
          <p className="mt-1 text-[11px] text-[color:var(--text-muted)]">
            Удаление возможно только до публикации и запуска.
          </p>
        ) : null}
      </div>

      <div className="mt-4 rounded-md border border-[color:var(--panel-border)] p-3">
        <p className="text-xs font-medium text-[color:var(--foreground)]">Настройки видимости</p>
        <div className="mt-2 grid gap-3 md:grid-cols-2">
          <label className="flex items-center gap-2 text-xs text-[color:var(--text-muted)]">
            <input
              type="checkbox"
              checked={isBracketPublished}
              onChange={(event) => setIsBracketPublished(event.target.checked)}
            />
            Опубликовать сетку
          </label>
          <label className="text-xs text-[color:var(--text-muted)]">
            Видимость расписания вперёд (`0` / `N` / `ALL`)
            <input
              value={scheduleVisibilityAhead}
              onChange={(event) => setScheduleVisibilityAhead(event.target.value)}
              placeholder="0"
              className="mt-1 w-full rounded-md border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] px-2 py-2 text-sm text-[color:var(--foreground)]"
            />
          </label>
        </div>
        <div className="mt-3">
          <button
            type="button"
            disabled={visibilityLoading}
            onClick={() => void saveVisibility()}
            className="rounded-md border border-[color:var(--panel-border)] px-3 py-1 text-xs text-[color:var(--foreground)] hover:opacity-80 disabled:opacity-50"
          >
            {visibilityLoading ? "Сохранение..." : "Сохранить видимость"}
          </button>
        </div>
        {visibilityMessage ? <p className="mt-2 text-xs text-emerald-300">{visibilityMessage}</p> : null}
      </div>

      <TeamManager tournamentId={tournament.id} allowOdd={form.allowOdd} />
      <RoundSettingsManager tournamentId={tournament.id} game={form.game} refreshKey={roundSettingsRefresh} />
      <ScheduleManager tournamentId={tournament.id} game={form.game} />
      <AuditLog tournamentId={tournament.id} />
    </section>
  );
}
