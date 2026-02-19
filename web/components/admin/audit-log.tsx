"use client";

import { useEffect, useState } from "react";

type AuditEntry = {
  id: number;
  tournamentId: number;
  userId?: number;
  userLogin?: string;
  entity: string;
  entityId: number;
  action: string;
  payload: unknown;
  createdAt: string;
};

type AuditListResponse = {
  items: AuditEntry[];
  page: number;
  pageSize: number;
  total: number;
  totalPages: number;
};

const PAGE_SIZE = 20;

export function AuditLog({ tournamentId }: { tournamentId: number }) {
  const [data, setData] = useState<AuditListResponse>({ items: [], page: 1, pageSize: PAGE_SIZE, total: 0, totalPages: 1 });
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isOpen, setIsOpen] = useState(false);

  const canPrev = data.page > 1;
  const canNext = data.page < data.totalPages;

  const fetchAudit = async (targetPage: number) => {
    setLoading(true);
    setError(null);
    try {
      const baseUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
      const query = new URLSearchParams({ page: String(targetPage), pageSize: String(PAGE_SIZE) });
      const res = await fetch(`${baseUrl}/api/v1/admin/tournaments/${tournamentId}/audit?${query.toString()}`, {
        credentials: "include",
        cache: "no-store"
      });
      if (!res.ok) {
        const text = await res.text();
        setError(text || "Не удалось загрузить аудит.");
        return;
      }
      const payload = (await res.json()) as AuditListResponse;
      setData(payload);
      setPage(payload.page);
    } catch {
      setError("Не удалось загрузить аудит.");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    setPage(1);
    void fetchAudit(1);
  }, [tournamentId]);

  return (
    <section className="mt-6 rounded-xl border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] p-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h3 className="text-base font-semibold">Аудит</h3>
          <p className="mt-1 text-xs text-[color:var(--text-muted)]">Все изменения администратора по этому турниру.</p>
        </div>
        <button
          type="button"
          onClick={() => setIsOpen((prev) => !prev)}
          className="rounded-md border border-[color:var(--panel-border)] px-3 py-1 text-xs text-[color:var(--foreground)] hover:opacity-80"
        >
          {isOpen ? "Скрыть аудит" : "Показать аудит"}
        </button>
      </div>

      {error ? <p className="mt-2 text-xs text-red-400">{error}</p> : null}
      {loading ? <p className="mt-2 text-sm text-[color:var(--text-muted)]">Загрузка...</p> : null}

      {!isOpen ? null : !loading && data.items.length === 0 ? (
        <p className="mt-2 text-sm text-[color:var(--text-muted)]">Событий аудита пока нет.</p>
      ) : null}

      {isOpen && !loading && data.items.length > 0 ? (
        <div className="mt-3 overflow-x-auto">
          <table className="min-w-full text-left text-xs">
            <thead className="text-[11px] uppercase tracking-wide text-[color:var(--text-muted)]">
              <tr>
                <th className="px-3 py-2">Время</th>
                <th className="px-3 py-2">Пользователь</th>
                <th className="px-3 py-2">Сущность</th>
                <th className="px-3 py-2">Действие</th>
                <th className="px-3 py-2">Данные</th>
              </tr>
            </thead>
            <tbody>
              {data.items.map((entry) => (
                <tr key={entry.id} className="border-t border-[color:var(--panel-border)]/60 align-top">
                  <td className="px-3 py-2 text-[color:var(--text-muted)]">{new Date(entry.createdAt).toLocaleString()}</td>
                  <td className="px-3 py-2 text-[color:var(--text-muted)]">{entry.userLogin ?? entry.userId ?? "-"}</td>
                  <td className="px-3 py-2 text-[color:var(--text-muted)]">
                    {entry.entity} #{entry.entityId}
                  </td>
                  <td className="px-3 py-2 text-[color:var(--text-muted)]">{entry.action}</td>
                  <td className="px-3 py-2 text-[color:var(--text-muted)]">
                    <pre className="max-h-32 overflow-auto rounded-md border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] px-2 py-1 text-[11px] text-[color:var(--foreground)]">
                      {JSON.stringify(entry.payload ?? {}, null, 2)}
                    </pre>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}

      {isOpen ? (
        <div className="mt-4 flex items-center justify-between text-xs text-[color:var(--text-muted)]">
          <button
            type="button"
            disabled={!canPrev}
            onClick={() => void fetchAudit(page - 1)}
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
            onClick={() => void fetchAudit(page + 1)}
            className="rounded-md border border-[color:var(--panel-border)] px-3 py-1 hover:opacity-80 disabled:opacity-50"
          >
            Вперёд
          </button>
        </div>
      ) : null}
    </section>
  );
}
