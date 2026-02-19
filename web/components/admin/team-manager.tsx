"use client";

import { useEffect, useMemo, useState } from "react";

type Team = {
  id: number;
  tournamentId: number;
  name: string;
  note?: string;
  status: string;
  statusReason?: string;
  seed?: number;
};

type TeamListResponse = {
  items: Team[];
  page: number;
  pageSize: number;
  total: number;
  totalPages: number;
};

type TeamValidation = {
  tournamentId: number;
  allowOdd: boolean;
  teamCount: number;
  isValid: boolean;
  message: string;
  suggestedAction?: string;
  bracketSize?: number;
};

type ImportCSVError = {
  row: number;
  value: string;
  message: string;
};

type ImportCSVResponse = {
  mode: string;
  total: number;
  created: number;
  duplicates: number;
  errors: ImportCSVError[];
};

const PAGE_SIZE = 10;
const STATUS_OPTIONS = ["ACTIVE", "WITHDRAWN", "DISQUALIFIED"] as const;
const REASON_OPTIONS = ["CHEATING", "TOXIC_BEHAVIOR", "NO_SHOW", "TECH_ISSUES", "OTHER"] as const;

const STATUS_LABELS: Record<string, string> = {
  ACTIVE: "Активна",
  WITHDRAWN: "Снята",
  DISQUALIFIED: "Дисквалифицирована"
};

const REASON_LABELS: Record<string, string> = {
  CHEATING: "Читерство",
  TOXIC_BEHAVIOR: "Токсичное поведение",
  NO_SHOW: "Неявка",
  TECH_ISSUES: "Технические проблемы",
  OTHER: "Другое"
};

export function TeamManager({ tournamentId, allowOdd }: { tournamentId: number; allowOdd: boolean }) {
  const [data, setData] = useState<TeamListResponse>({ items: [], page: 1, pageSize: PAGE_SIZE, total: 0, totalPages: 1 });
  const [validation, setValidation] = useState<TeamValidation | null>(null);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(false);
  const [validationLoading, setValidationLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [newName, setNewName] = useState("");
  const [newNote, setNewNote] = useState("");
  const [creating, setCreating] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [editName, setEditName] = useState("");
  const [editNote, setEditNote] = useState("");
  const [savingId, setSavingId] = useState<number | null>(null);
  const [deletingId, setDeletingId] = useState<number | null>(null);
  const [seedingLoading, setSeedingLoading] = useState<false | "generate" | "regenerate">(false);
  const [seedingMessage, setSeedingMessage] = useState<string | null>(null);
  const [reorderList, setReorderList] = useState<Team[]>([]);
  const [reorderSavedSnapshot, setReorderSavedSnapshot] = useState<number[]>([]);
  const [draggingTeamId, setDraggingTeamId] = useState<number | null>(null);
  const [reorderSaving, setReorderSaving] = useState(false);
  const [statusDraft, setStatusDraft] = useState<Record<number, { status: string; reason: string }>>({});
  const [statusSavingId, setStatusSavingId] = useState<number | null>(null);
  const [importFile, setImportFile] = useState<File | null>(null);
  const [importResult, setImportResult] = useState<ImportCSVResponse | null>(null);
  const [importLoading, setImportLoading] = useState<false | "dryRun" | "apply">(false);

  const canPrev = data.page > 1;
  const canNext = data.page < data.totalPages;
  const hasReorderChanges = JSON.stringify(reorderList.map((team) => team.id)) !== JSON.stringify(reorderSavedSnapshot);

  const fetchValidation = async () => {
    setValidationLoading(true);
    try {
      const baseUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
      const res = await fetch(`${baseUrl}/api/v1/admin/tournaments/${tournamentId}/teams/validation`, {
        credentials: "include",
        cache: "no-store"
      });
      if (!res.ok) {
        setValidation(null);
        return;
      }
      setValidation((await res.json()) as TeamValidation);
    } finally {
      setValidationLoading(false);
    }
  };

  const fetchTeams = async (targetPage: number) => {
    setLoading(true);
    setError(null);
    try {
      const baseUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
      const query = new URLSearchParams({ page: String(targetPage), pageSize: String(PAGE_SIZE) });
      const res = await fetch(`${baseUrl}/api/v1/admin/tournaments/${tournamentId}/teams?${query.toString()}`, {
        credentials: "include",
        cache: "no-store"
      });
      if (!res.ok) {
        const text = await res.text();
        setError(text || "Не удалось загрузить команды.");
        return;
      }
      const payload = (await res.json()) as TeamListResponse;
      const safeItems = Array.isArray(payload.items) ? payload.items : [];
      const safePayload: TeamListResponse = {
        items: safeItems,
        page: payload.page ?? 1,
        pageSize: payload.pageSize ?? PAGE_SIZE,
        total: payload.total ?? safeItems.length,
        totalPages: payload.totalPages ?? 1
      };
      setData(safePayload);
      setPage(safePayload.page);
      setStatusDraft((prev) => {
        const next = { ...prev };
        for (const team of safeItems) {
          next[team.id] = {
            status: team.status,
            reason: team.statusReason ?? ""
          };
        }
        return next;
      });
      await fetchValidation();
    } catch {
      setError("Не удалось загрузить команды.");
    } finally {
      setLoading(false);
    }
  };

  const fetchReorderList = async () => {
    try {
      const baseUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
      const query = new URLSearchParams({ page: "1", pageSize: "500" });
      const res = await fetch(`${baseUrl}/api/v1/admin/tournaments/${tournamentId}/teams?${query.toString()}`, {
        credentials: "include",
        cache: "no-store"
      });
      if (!res.ok) {
        return;
      }
      const payload = (await res.json()) as TeamListResponse;
      const safeItems = Array.isArray(payload.items) ? payload.items : [];
      setReorderList(safeItems);
      setReorderSavedSnapshot(safeItems.map((team) => team.id));
    } catch {
      // keep existing list
    }
  };

  useEffect(() => {
    setPage(1);
    void fetchTeams(1);
    void fetchReorderList();
  }, [tournamentId, allowOdd]);

  const currentRows = useMemo(() => data.items, [data.items]);

  const createTeam = async (event: React.FormEvent) => {
    event.preventDefault();
    const name = newName.trim();
    if (!name) {
      setError("Введите название команды.");
      return;
    }

    setCreating(true);
    setError(null);
    try {
      const baseUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
      const res = await fetch(`${baseUrl}/api/v1/admin/tournaments/${tournamentId}/teams`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ name, note: newNote })
      });
      if (!res.ok) {
        const text = await res.text();
        setError(text || "Не удалось создать команду.");
        return;
      }
      setNewName("");
      setNewNote("");
      await fetchTeams(1);
      await fetchReorderList();
    } catch {
      setError("Не удалось создать команду.");
    } finally {
      setCreating(false);
    }
  };

  const startEdit = (team: Team) => {
    setEditingId(team.id);
    setEditName(team.name);
    setEditNote(team.note ?? "");
  };

  const saveEdit = async (teamId: number) => {
    const name = editName.trim();
    if (!name) {
      setError("Введите название команды.");
      return;
    }

    setSavingId(teamId);
    setError(null);
    try {
      const baseUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
      const res = await fetch(`${baseUrl}/api/v1/admin/tournaments/${tournamentId}/teams/${teamId}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ name, note: editNote })
      });
      if (!res.ok) {
        const text = await res.text();
        setError(text || "Не удалось обновить команду.");
        return;
      }
      setEditingId(null);
      await fetchTeams(page);
    } catch {
      setError("Не удалось обновить команду.");
    } finally {
      setSavingId(null);
    }
  };

  const removeTeam = async (teamId: number) => {
    setDeletingId(teamId);
    setError(null);
    try {
      const baseUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
      const res = await fetch(`${baseUrl}/api/v1/admin/tournaments/${tournamentId}/teams/${teamId}`, {
        method: "DELETE",
        credentials: "include"
      });
      if (!res.ok) {
        const text = await res.text();
        setError(text || "Не удалось удалить команду.");
        return;
      }
      const targetPage = data.items.length === 1 && page > 1 ? page - 1 : page;
      await fetchTeams(targetPage);
      await fetchReorderList();
    } catch {
      setError("Не удалось удалить команду.");
    } finally {
      setDeletingId(null);
    }
  };

  const generateSeeding = async (overwrite: boolean) => {
    setSeedingLoading(overwrite ? "regenerate" : "generate");
    setSeedingMessage(null);
    setError(null);
    try {
      const baseUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
      const res = await fetch(`${baseUrl}/api/v1/admin/tournaments/${tournamentId}/teams/seeding/generate`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ overwrite })
      });
      if (!res.ok) {
        const text = await res.text();
        setError(text || "Не удалось сформировать порядок команд.");
        return;
      }
      setSeedingMessage(overwrite ? "Порядок команд перерандомизирован." : "Порядок команд сформирован случайно.");
      await fetchTeams(page);
      await fetchReorderList();
    } catch {
      setError("Не удалось сформировать порядок команд.");
    } finally {
      setSeedingLoading(false);
    }
  };

  const moveInReorderList = (draggedId: number, targetId: number) => {
    if (draggedId === targetId) {
      return;
    }
    setReorderList((prev) => {
      const fromIndex = prev.findIndex((item) => item.id === draggedId);
      const toIndex = prev.findIndex((item) => item.id === targetId);
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
    setSeedingMessage(null);
    try {
      const baseUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
      const res = await fetch(`${baseUrl}/api/v1/admin/tournaments/${tournamentId}/teams/seeding/reorder`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ teamIds: reorderList.map((team) => team.id) })
      });
      if (!res.ok) {
        const text = await res.text();
        setError(text || "Не удалось сохранить порядок команд.");
        return;
      }

      setReorderSavedSnapshot(reorderList.map((team) => team.id));
      setSeedingMessage("Порядок команд сохранён.");
      await fetchTeams(page);
      await fetchReorderList();
    } catch {
      setError("Не удалось сохранить порядок команд.");
    } finally {
      setReorderSaving(false);
    }
  };

  const saveStatus = async (teamId: number) => {
    const draft = statusDraft[teamId];
    if (!draft) return;

    setStatusSavingId(teamId);
    setError(null);
    try {
      const baseUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
      const res = await fetch(`${baseUrl}/api/v1/admin/tournaments/${tournamentId}/teams/${teamId}/status`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({
          status: draft.status,
          reason: draft.status === "ACTIVE" ? "" : draft.reason
        })
      });
      if (!res.ok) {
        const text = await res.text();
        setError(text || "Не удалось обновить статус команды.");
        return;
      }
      await fetchTeams(page);
    } catch {
      setError("Не удалось обновить статус команды.");
    } finally {
      setStatusSavingId(null);
    }
  };

  const runImport = async (dryRun: boolean) => {
    if (!importFile) {
      setError("Нужен CSV-файл.");
      return;
    }

    setImportLoading(dryRun ? "dryRun" : "apply");
    setError(null);
    try {
      const baseUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
      const form = new FormData();
      form.append("file", importFile);
      form.append("dryRun", dryRun ? "true" : "false");
      const res = await fetch(`${baseUrl}/api/v1/admin/tournaments/${tournamentId}/teams/import`, {
        method: "POST",
        credentials: "include",
        body: form
      });
      if (!res.ok) {
        const text = await res.text();
        setError(text || "Не удалось импортировать команды.");
        return;
      }
      const payload = (await res.json()) as ImportCSVResponse;
      setImportResult(payload);
      if (!dryRun) {
        await fetchTeams(1);
        await fetchReorderList();
      }
    } catch {
      setError("Не удалось импортировать команды.");
    } finally {
      setImportLoading(false);
    }
  };

  return (
    <section className="mt-6 rounded-xl border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] p-4">
      <h3 className="text-base font-semibold">Команды</h3>
      <p className="mt-1 text-xs text-[color:var(--text-muted)]">Управление командами турнира.</p>

      <form onSubmit={createTeam} className="mt-4 grid gap-3 md:grid-cols-3">
        <input
          value={newName}
          onChange={(event) => setNewName(event.target.value)}
          placeholder="Название команды"
          className="rounded-md border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] px-2 py-2 text-sm text-[color:var(--foreground)]"
        />
        <input
          value={newNote}
          onChange={(event) => setNewNote(event.target.value)}
          placeholder="Примечание (опционально)"
          className="rounded-md border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] px-2 py-2 text-sm text-[color:var(--foreground)]"
        />
        <button
          type="submit"
          disabled={creating}
          className="rounded-md border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] px-4 py-2 text-sm font-medium hover:opacity-80 disabled:opacity-50"
        >
          {creating ? "Добавление..." : "Добавить команду"}
        </button>
      </form>

      <div className="mt-3 flex gap-2">
        <button
          type="button"
          disabled={seedingLoading !== false || reorderSaving}
          onClick={() => void generateSeeding(false)}
          className="rounded-md border border-[color:var(--panel-border)] px-3 py-1 text-xs hover:opacity-80 disabled:opacity-50"
        >
          {seedingLoading === "generate" ? "Формирование..." : "Случайно распределить порядок"}
        </button>
        <button
          type="button"
          disabled={seedingLoading !== false || reorderSaving}
          onClick={() => void generateSeeding(true)}
          className="rounded-md border border-[color:var(--panel-border)] px-3 py-1 text-xs hover:opacity-80 disabled:opacity-50"
        >
          {seedingLoading === "regenerate" ? "Перемешивание..." : "Перемешать порядок"}
        </button>
      </div>

      <div className="mt-4 rounded-md border border-[color:var(--panel-border)] p-3">
        <p className="text-xs text-[color:var(--text-muted)]">CSV импорт (колонки: name, note)</p>
        <div className="mt-2 flex flex-wrap items-center gap-2">
          <input
            type="file"
            accept=".csv,text/csv"
            onChange={(event) => setImportFile(event.target.files?.[0] ?? null)}
            className="text-xs text-[color:var(--text-muted)]"
          />
          <button
            type="button"
            disabled={importLoading !== false}
            onClick={() => void runImport(true)}
            className="rounded-md border border-[color:var(--panel-border)] px-3 py-1 text-xs hover:opacity-80 disabled:opacity-50"
          >
            {importLoading === "dryRun" ? "Проверка..." : "Проверить"}
          </button>
          <button
            type="button"
            disabled={importLoading !== false}
            onClick={() => void runImport(false)}
            className="rounded-md border border-[color:var(--panel-border)] px-3 py-1 text-xs hover:opacity-80 disabled:opacity-50"
          >
            {importLoading === "apply" ? "Импорт..." : "Импортировать"}
          </button>
        </div>
        {importResult ? (
          <div className="mt-3 text-xs text-[color:var(--text-muted)]">
            <div>
              Режим: {importResult.mode}. Всего строк: {importResult.total}. Создано: {importResult.created}. Дубликаты: {importResult.duplicates}.
            </div>
            {importResult.errors.length > 0 ? (
              <ul className="mt-2 max-h-40 overflow-auto rounded-md border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)]/40 p-2">
                {importResult.errors.map((err, index) => (
                  <li key={`${err.row}-${index}`} className="text-[11px] text-amber-300">
                    Строка {err.row}: {err.value} — {err.message}
                  </li>
                ))}
              </ul>
            ) : null}
          </div>
        ) : null}
      </div>

      <div className="mt-3 rounded-md border border-[color:var(--panel-border)] p-3">
        <div className="flex items-center justify-between">
          <p className="text-xs text-[color:var(--text-muted)]">Порядок команд (перетаскивание)</p>
          <button
            type="button"
            disabled={!hasReorderChanges || reorderSaving || seedingLoading !== false}
            onClick={() => void saveReorder()}
            className="rounded-md border border-[color:var(--panel-border)] px-3 py-1 text-xs hover:opacity-80 disabled:opacity-50"
          >
            {reorderSaving ? "Сохранение..." : "Сохранить порядок"}
          </button>
        </div>
        {reorderList.length === 0 ? (
          <p className="mt-2 text-xs text-[color:var(--text-muted)]">Нет команд для сортировки.</p>
        ) : (
          <ul className="mt-2 space-y-1">
            {reorderList.map((team, index) => (
              <li
                key={team.id}
                draggable
                onDragStart={() => setDraggingTeamId(team.id)}
                onDragOver={(event) => event.preventDefault()}
                onDrop={() => {
                  if (draggingTeamId !== null) {
                    moveInReorderList(draggingTeamId, team.id);
                  }
                  setDraggingTeamId(null);
                }}
                onDragEnd={() => setDraggingTeamId(null)}
                className="cursor-grab rounded-md border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)]/40 px-2 py-1 text-xs text-[color:var(--foreground)]"
              >
                #{index + 1} {team.name}
              </li>
            ))}
          </ul>
        )}
      </div>

      {error ? <p className="mt-2 text-xs text-red-400">{error}</p> : null}
      {seedingMessage ? <p className="mt-2 text-xs text-emerald-300">{seedingMessage}</p> : null}
      <div className={`mt-2 rounded-md border px-3 py-2 text-xs ${validation?.isValid ? "border-emerald-700/40 text-emerald-300" : "border-amber-700/40 text-amber-300"}`}>
        {validationLoading ? (
          <span>Проверка ограничений команд...</span>
        ) : validation ? (
          <span>
            {validation.message}
            {validation.suggestedAction ? ` ${validation.suggestedAction}` : ""}
          </span>
        ) : (
          <span>Проверка недоступна.</span>
        )}
      </div>

      <div className="mt-4 text-xs text-[color:var(--text-muted)]">Всего команд: {data.total}</div>
      {loading ? <p className="mt-2 text-sm text-[color:var(--text-muted)]">Загрузка...</p> : null}

      {!loading && currentRows.length === 0 ? <p className="mt-2 text-sm text-[color:var(--text-muted)]">Команд пока нет.</p> : null}

      {!loading && currentRows.length > 0 ? (
        <div className="mt-3 overflow-x-auto">
          <table className="min-w-full text-left text-sm">
            <thead className="text-xs uppercase tracking-wide text-[color:var(--text-muted)]">
              <tr>
                <th className="px-3 py-2">ID</th>
                <th className="px-3 py-2">Порядок</th>
                <th className="px-3 py-2">Название</th>
                <th className="px-3 py-2">Статус</th>
                <th className="px-3 py-2">Примечание</th>
                <th className="px-3 py-2">Действия</th>
              </tr>
            </thead>
            <tbody>
              {currentRows.map((team) => {
                const isEditing = editingId === team.id;
                const statusState = statusDraft[team.id] ?? { status: team.status, reason: team.statusReason ?? "" };
                const needsReason = statusState.status !== "ACTIVE";
                return (
                  <tr key={team.id} className="border-t border-[color:var(--panel-border)]">
                    <td className="px-3 py-2 text-[color:var(--text-muted)]">{team.id}</td>
                    <td className="px-3 py-2 text-[color:var(--text-muted)]">{team.seed ?? "-"}</td>
                    <td className="px-3 py-2">
                      {isEditing ? (
                        <input
                          value={editName}
                          onChange={(event) => setEditName(event.target.value)}
                          className="w-full rounded-md border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] px-2 py-1 text-sm text-[color:var(--foreground)]"
                        />
                      ) : (
                        <span className="font-medium">{team.name}</span>
                      )}
                    </td>
                    <td className="px-3 py-2">
                      <div className="flex flex-wrap items-center gap-2">
                        <select
                          value={statusState.status}
                          onChange={(event) =>
                            setStatusDraft((prev) => ({
                              ...prev,
                              [team.id]: { ...statusState, status: event.target.value }
                            }))
                          }
                          className="rounded-md border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] px-2 py-1 text-xs text-[color:var(--foreground)]"
                        >
                          {STATUS_OPTIONS.map((status) => (
                            <option key={status} value={status}>
                              {STATUS_LABELS[status] ?? status}
                            </option>
                          ))}
                        </select>
                        <select
                          value={statusState.reason}
                          disabled={!needsReason}
                          onChange={(event) =>
                            setStatusDraft((prev) => ({
                              ...prev,
                              [team.id]: { ...statusState, reason: event.target.value }
                            }))
                          }
                          className="min-w-[140px] rounded-md border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] px-2 py-1 text-xs text-[color:var(--foreground)] disabled:opacity-50"
                        >
                          <option value="">{needsReason ? "Выберите причину" : "Без причины"}</option>
                          {REASON_OPTIONS.map((reason) => (
                            <option key={reason} value={reason}>
                              {REASON_LABELS[reason] ?? reason}
                            </option>
                          ))}
                        </select>
                      </div>
                    </td>
                    <td className="px-3 py-2">
                      {isEditing ? (
                        <input
                          value={editNote}
                          onChange={(event) => setEditNote(event.target.value)}
                          className="w-full rounded-md border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] px-2 py-1 text-sm text-[color:var(--foreground)]"
                        />
                      ) : (
                        <span className="text-[color:var(--text-muted)]">{team.note || "-"}</span>
                      )}
                    </td>
                    <td className="px-3 py-2">
                      <div className="flex flex-wrap gap-2">
                        <button
                          type="button"
                          disabled={
                            statusSavingId === team.id ||
                            (statusState.status !== "ACTIVE" && statusState.reason.trim() === "")
                          }
                          onClick={() => void saveStatus(team.id)}
                          className="rounded-md border border-[color:var(--panel-border)] px-2 py-1 text-xs hover:opacity-80 disabled:opacity-50"
                        >
                          {statusSavingId === team.id ? "Сохранение..." : "Сохранить статус"}
                        </button>
                        {isEditing ? (
                          <>
                            <button
                              type="button"
                              disabled={savingId === team.id}
                              onClick={() => void saveEdit(team.id)}
                              className="rounded-md border border-[color:var(--panel-border)] px-2 py-1 text-xs hover:opacity-80 disabled:opacity-50"
                            >
                              Сохранить
                            </button>
                            <button
                              type="button"
                              onClick={() => setEditingId(null)}
                              className="rounded-md border border-[color:var(--panel-border)] px-2 py-1 text-xs hover:opacity-80"
                            >
                              Отмена
                            </button>
                          </>
                        ) : (
                          <>
                            <button
                              type="button"
                              onClick={() => startEdit(team)}
                              className="rounded-md border border-[color:var(--panel-border)] px-2 py-1 text-xs hover:opacity-80"
                            >
                              Редактировать
                            </button>
                            <button
                              type="button"
                              disabled={deletingId === team.id}
                              onClick={() => void removeTeam(team.id)}
                              className="rounded-md border border-[color:var(--panel-border)] px-2 py-1 text-xs hover:opacity-80 disabled:opacity-50"
                            >
                              Удалить
                            </button>
                          </>
                        )}
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
          onClick={() => void fetchTeams(page - 1)}
          className="rounded-md border border-[color:var(--panel-border)] px-3 py-1 hover:opacity-80 disabled:opacity-50"
        >
          Назад
        </button>
        <span>
          Страница {data.page} из {data.totalPages}
        </span>
        <button
          type="button"
          disabled={!canNext}
          onClick={() => void fetchTeams(page + 1)}
          className="rounded-md border border-[color:var(--panel-border)] px-3 py-1 hover:opacity-80 disabled:opacity-50"
        >
          Вперёд
        </button>
      </div>
    </section>
  );
}
