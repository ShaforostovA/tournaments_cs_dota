import Link from "next/link";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { CreateTournamentForm } from "@/components/admin/create-tournament-form";
import { TournamentEditor } from "@/components/admin/tournament-editor";
import { ThemeToggle } from "@/components/theme-toggle";

type Tournament = {
  id: number;
  game: string;
  name: string;
  description?: string;
  startDate?: string;
  endDate?: string;
  allowOdd: boolean;
  status: string;
  isListed: boolean;
  isBracketPublished: boolean;
  scheduleVisibilityAhead: string;
};

type AdminTournamentListResponse = {
  items: Tournament[];
  page: number;
  pageSize: number;
  total: number;
  totalPages: number;
};

type SearchParams = {
  game?: string | string[];
  status?: string | string[];
  search?: string | string[];
  page?: string | string[];
  pageSize?: string | string[];
  editId?: string | string[];
};

function firstValue(input?: string | string[]): string {
  if (!input) return "";
  return Array.isArray(input) ? input[0] ?? "" : input;
}

function parsePositiveInt(input: string, fallback: number): number {
  const value = Number.parseInt(input, 10);
  if (Number.isNaN(value) || value < 1) {
    return fallback;
  }
  return value;
}

function statusLabel(status: string): string {
  if (status === "DRAFT") return "Черновик";
  if (status === "READY") return "Готов";
  if (status === "RUNNING") return "Идёт";
  if (status === "FINISHED") return "Завершён";
  return status;
}

async function fetchAdminTournaments(
  params: { game: string; status: string; search: string; page: number; pageSize: number },
  session: string
) {
  const baseUrl = process.env.API_URL ?? process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
  const query = new URLSearchParams();

  if (params.game) query.set("game", params.game);
  if (params.status) query.set("status", params.status);
  if (params.search) query.set("search", params.search);
  query.set("page", String(params.page));
  query.set("pageSize", String(params.pageSize));

  const res = await fetch(`${baseUrl}/api/v1/admin/tournaments?${query.toString()}`, {
    cache: "no-store",
    headers: session ? { cookie: `admin_session=${session}` } : undefined
  });

  if (res.status === 401) {
    redirect("/admin/login");
  }
  if (!res.ok) {
    return { items: [], page: params.page, pageSize: params.pageSize, total: 0, totalPages: 1 } satisfies AdminTournamentListResponse;
  }
  return (await res.json()) as AdminTournamentListResponse;
}

async function fetchAdminTournamentByID(id: number, session: string): Promise<Tournament | null> {
  const baseUrl = process.env.API_URL ?? process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
  const res = await fetch(`${baseUrl}/api/v1/admin/tournaments/${id}`, {
    cache: "no-store",
    headers: session ? { cookie: `admin_session=${session}` } : undefined
  });

  if (res.status === 401) {
    redirect("/admin/login");
  }
  if (!res.ok) {
    return null;
  }
  return (await res.json()) as Tournament;
}

function buildAdminHref(filters: {
  game: string;
  status: string;
  search: string;
  pageSize: number;
  page: number;
  editId?: number;
}) {
  const params = new URLSearchParams();
  if (filters.game) params.set("game", filters.game);
  if (filters.status) params.set("status", filters.status);
  if (filters.search) params.set("search", filters.search);
  params.set("pageSize", String(filters.pageSize));
  params.set("page", String(filters.page));
  if (filters.editId && filters.editId > 0) params.set("editId", String(filters.editId));
  return `/admin?${params.toString()}`;
}

function formatDateRange(t: Tournament): string {
  if (!t.startDate && !t.endDate) return "Дата не задана";
  if (t.startDate && t.endDate) return `${t.startDate} – ${t.endDate}`;
  if (t.startDate) return `с ${t.startDate}`;
  return `по ${t.endDate}`;
}

export default async function AdminHomePage({ searchParams }: { searchParams?: SearchParams }) {
  const game = firstValue(searchParams?.game);
  const status = firstValue(searchParams?.status);
  const search = firstValue(searchParams?.search);
  const page = parsePositiveInt(firstValue(searchParams?.page), 1);
  const pageSize = parsePositiveInt(firstValue(searchParams?.pageSize), 10);
  const editId = parsePositiveInt(firstValue(searchParams?.editId), 0);

  const session = cookies().get("admin_session")?.value ?? "";
  const data = await fetchAdminTournaments({ game, status, search, page, pageSize }, session);
  const safeData: AdminTournamentListResponse = {
    items: Array.isArray(data.items) ? data.items : [],
    page: Number.isFinite(data.page) ? data.page : page,
    pageSize: Number.isFinite(data.pageSize) ? data.pageSize : pageSize,
    total: Number.isFinite(data.total) ? data.total : 0,
    totalPages: Number.isFinite(data.totalPages) ? data.totalPages : 1
  };
  const selectedTournament = editId > 0 ? await fetchAdminTournamentByID(editId, session) : null;
  const prevPage = Math.max(1, safeData.page - 1);
  const nextPage = Math.min(safeData.totalPages, safeData.page + 1);

  return (
    <main className="min-h-screen page-shell">
      <div className="mx-auto max-w-6xl px-6 py-12">
        <div className="flex items-center justify-between gap-3">
          <h1 className="text-2xl font-semibold">Админ‑панель</h1>
          <div className="flex items-center gap-3">
            <ThemeToggle />
            <Link className="text-sm text-[color:var(--text-muted)] underline" href="/admin/login">
              Выйти
            </Link>
          </div>
        </div>

        <CreateTournamentForm />

        {selectedTournament ? (
          <div className="mt-6">
            <div className="mb-2 flex items-center justify-between">
              <h2 className="text-sm text-[color:var(--text-muted)]">Редактирование турнира</h2>
              <Link
                className="text-xs text-[color:var(--text-muted)] underline"
                href={buildAdminHref({ game, status, search, pageSize: data.pageSize, page: data.page })}
              >
                Закрыть редактор
              </Link>
            </div>
            <TournamentEditor tournament={selectedTournament} />
          </div>
        ) : null}

        <form method="get" className="mt-6 grid gap-3 rounded-xl border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] p-4 md:grid-cols-4">
          <label className="text-xs text-[color:var(--text-muted)]">
            Игра
            <select
              name="game"
              defaultValue={game}
              className="mt-1 w-full rounded-md border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] px-2 py-2 text-sm text-[color:var(--foreground)]"
            >
              <option value="">Все</option>
              <option value="DOTA2">Dota 2</option>
              <option value="CS2">CS2</option>
            </select>
          </label>

          <label className="text-xs text-[color:var(--text-muted)]">
            Статус
            <select
              name="status"
              defaultValue={status}
              className="mt-1 w-full rounded-md border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] px-2 py-2 text-sm text-[color:var(--foreground)]"
            >
              <option value="">Все</option>
              <option value="DRAFT">Черновик</option>
              <option value="READY">Готов</option>
              <option value="RUNNING">Идёт</option>
              <option value="FINISHED">Завершён</option>
            </select>
          </label>

          <label className="text-xs text-[color:var(--text-muted)]">
            Поиск
            <input
              name="search"
              defaultValue={search}
              placeholder="Название турнира"
              className="mt-1 w-full rounded-md border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] px-2 py-2 text-sm text-[color:var(--foreground)]"
            />
          </label>

          <label className="text-xs text-[color:var(--text-muted)]">
            Размер страницы
            <select
              name="pageSize"
              defaultValue={String(pageSize)}
              className="mt-1 w-full rounded-md border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] px-2 py-2 text-sm text-[color:var(--foreground)]"
            >
              <option value="10">10</option>
              <option value="20">20</option>
              <option value="50">50</option>
            </select>
          </label>

          <input type="hidden" name="page" value="1" />
          {editId > 0 ? <input type="hidden" name="editId" value={String(editId)} /> : null}
          <div className="md:col-span-4">
            <button
              type="submit"
              className="rounded-md border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] px-4 py-2 text-sm font-medium text-[color:var(--foreground)] hover:opacity-80"
            >
              Применить
            </button>
          </div>
        </form>

        <div className="mt-4 rounded-xl border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] p-4">
          <div className="mb-3 text-sm text-[color:var(--text-muted)]">Всего турниров: {safeData.total}</div>
          {safeData.items.length === 0 ? (
            <p className="text-sm text-[color:var(--text-muted)]">Турниры не найдены.</p>
          ) : (
            <div className="overflow-x-auto">
              <table className="min-w-full text-left text-sm">
                <thead className="text-xs uppercase tracking-wide text-[color:var(--text-muted)]">
                  <tr>
                    <th className="px-3 py-2">ID</th>
                    <th className="px-3 py-2">Название</th>
                    <th className="px-3 py-2">Игра</th>
                    <th className="px-3 py-2">Статус</th>
                    <th className="px-3 py-2">Даты</th>
                    <th className="px-3 py-2">Публикация</th>
                    <th className="px-3 py-2">Действия</th>
                  </tr>
                </thead>
                <tbody>
                  {safeData.items.map((t) => (
                    <tr key={t.id} className="border-t border-[color:var(--panel-border)]/60">
                      <td className="px-3 py-2 text-[color:var(--foreground)]">{t.id}</td>
                      <td className="px-3 py-2 font-medium">{t.name}</td>
                      <td className="px-3 py-2 text-[color:var(--text-muted)]">{t.game}</td>
                      <td className="px-3 py-2 text-[color:var(--text-muted)]">{statusLabel(t.status)}</td>
                      <td className="px-3 py-2 text-[color:var(--text-muted)]">{formatDateRange(t)}</td>
                      <td className="px-3 py-2 text-[color:var(--text-muted)]">{t.isListed ? "Да" : "Нет"}</td>
                      <td className="px-3 py-2">
                        <Link
                          className="text-xs text-[color:var(--foreground)] underline"
                        href={buildAdminHref({ game, status, search, pageSize: safeData.pageSize, page: safeData.page, editId: t.id })}
                      >
                          Редактировать
                        </Link>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          <div className="mt-4 flex items-center justify-between text-sm text-[color:var(--text-muted)]">
            <Link
              className={`rounded-md border px-3 py-1 ${
                safeData.page > 1 ? "border-[color:var(--panel-border)] hover:opacity-80" : "pointer-events-none border-[color:var(--panel-border)] opacity-50"
              }`}
              href={buildAdminHref({ game, status, search, pageSize: safeData.pageSize, page: prevPage, editId })}
            >
              Назад
            </Link>
            <span>
              Страница {safeData.page} из {safeData.totalPages}
            </span>
            <Link
              className={`rounded-md border px-3 py-1 ${
                safeData.page < safeData.totalPages ? "border-[color:var(--panel-border)] hover:opacity-80" : "pointer-events-none border-[color:var(--panel-border)] opacity-50"
              }`}
              href={buildAdminHref({ game, status, search, pageSize: safeData.pageSize, page: nextPage, editId })}
            >
              Вперёд
            </Link>
          </div>
        </div>
      </div>
    </main>
  );
}
