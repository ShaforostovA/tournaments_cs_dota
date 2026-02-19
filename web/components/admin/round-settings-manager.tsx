"use client";

import { useEffect, useMemo, useState } from "react";

type RoundSetting = {
  round: number;
  bo: number;
  disputeRule: string;
  matches: number;
};

const BO_OPTIONS = [1, 3, 5];

const RULES = ["OVERTIME", "REPLAY", "ADMIN_DECISION", "COIN_TOSS", "FORFEIT"] as const;

function ruleLabel(game: string, rule: string): string {
  const isCS2 = game === "CS2";
  if (rule === "OVERTIME") return isCS2 ? "Овертайм (MR3)" : "Доп. игра";
  if (rule === "REPLAY") return isCS2 ? "Переиграть карту" : "Переиграть матч";
  if (rule === "ADMIN_DECISION") return "Решение администратора";
  if (rule === "COIN_TOSS") return isCS2 ? "Ножи/монета" : "Жеребьёвка монетой";
  if (rule === "FORFEIT") return "Тех. победа";
  return rule;
}

export function RoundSettingsManager({
  tournamentId,
  game,
  refreshKey
}: {
  tournamentId: number;
  game: string;
  refreshKey?: number;
}) {
  const [items, setItems] = useState<RoundSetting[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [message, setMessage] = useState<string | null>(null);
  const [savingRound, setSavingRound] = useState<number | null>(null);

  const formState = useMemo(
    () =>
      items.reduce<Record<number, { bo: number; disputeRule: string }>>((acc, item) => {
        acc[item.round] = { bo: item.bo, disputeRule: item.disputeRule };
        return acc;
      }, {}),
    [items]
  );
  const [draft, setDraft] = useState<Record<number, { bo: number; disputeRule: string }>>({});

  const fetchItems = async () => {
    setLoading(true);
    setError(null);
    try {
      const baseUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
      const res = await fetch(`${baseUrl}/api/v1/admin/tournaments/${tournamentId}/round-settings`, {
        credentials: "include",
        cache: "no-store"
      });
      if (!res.ok) {
        const text = await res.text();
        setError(text || "Не удалось загрузить настройки раундов.");
        return;
      }
      const payload = (await res.json()) as RoundSetting[];
      setItems(payload);
      setDraft(
        payload.reduce<Record<number, { bo: number; disputeRule: string }>>((acc, item) => {
          acc[item.round] = { bo: item.bo, disputeRule: item.disputeRule };
          return acc;
        }, {})
      );
    } catch {
      setError("Не удалось загрузить настройки раундов.");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void fetchItems();
  }, [tournamentId, refreshKey]);

  const saveRound = async (round: number) => {
    const value = draft[round];
    if (!value) return;

    setSavingRound(round);
    setError(null);
    setMessage(null);
    try {
      const baseUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
      const res = await fetch(`${baseUrl}/api/v1/admin/tournaments/${tournamentId}/round-settings/${round}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify(value)
      });
      if (!res.ok) {
        const text = await res.text();
        setError(text || "Не удалось сохранить настройки раундов.");
        return;
      }
      setMessage(`Настройки раунда ${round} сохранены.`);
      await fetchItems();
    } catch {
      setError("Не удалось сохранить настройки раундов.");
    } finally {
      setSavingRound(null);
    }
  };

  return (
    <section className="mt-6 rounded-xl border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] p-4">
      <h3 className="text-base font-semibold">Настройки раундов</h3>
      <p className="mt-1 text-xs text-[color:var(--text-muted)]">Настройте BoN и правила спорных ситуаций для каждого раунда.</p>

      {loading ? <p className="mt-2 text-sm text-[color:var(--text-muted)]">Загрузка...</p> : null}
      {error ? <p className="mt-2 text-xs text-red-400">{error}</p> : null}
      {message ? <p className="mt-2 text-xs text-emerald-300">{message}</p> : null}

      {!loading && items.length === 0 ? <p className="mt-2 text-sm text-[color:var(--text-muted)]">Раундов нет. Сначала сгенерируйте сетку.</p> : null}

      {!loading && items.length > 0 ? (
        <div className="mt-3 overflow-x-auto">
          <table className="min-w-full text-left text-sm">
            <thead className="text-xs uppercase tracking-wide text-[color:var(--text-muted)]">
              <tr>
                <th className="px-3 py-2">Раунд</th>
                <th className="px-3 py-2">Матчи</th>
                <th className="px-3 py-2">BoN</th>
                <th className="px-3 py-2">Правило споров</th>
                <th className="px-3 py-2">Действия</th>
              </tr>
            </thead>
            <tbody>
              {items.map((item) => (
                <tr key={item.round} className="border-t border-[color:var(--panel-border)]">
                  <td className="px-3 py-2 text-[color:var(--text-muted)]">{item.round}</td>
                  <td className="px-3 py-2 text-[color:var(--text-muted)]">{item.matches}</td>
                  <td className="px-3 py-2">
                    <select
                      value={draft[item.round]?.bo ?? item.bo}
                      onChange={(event) =>
                        setDraft((prev) => ({
                          ...prev,
                          [item.round]: {
                            bo: Number.parseInt(event.target.value, 10),
                            disputeRule: prev[item.round]?.disputeRule ?? item.disputeRule
                          }
                        }))
                      }
                      className="rounded-md border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] px-2 py-1 text-sm text-[color:var(--foreground)]"
                    >
                      {BO_OPTIONS.map((bo) => (
                        <option key={bo} value={bo}>
                          BO{bo}
                        </option>
                      ))}
                    </select>
                  </td>
                  <td className="px-3 py-2">
                    <select
                      value={draft[item.round]?.disputeRule ?? item.disputeRule}
                      onChange={(event) =>
                        setDraft((prev) => ({
                          ...prev,
                          [item.round]: {
                            bo: prev[item.round]?.bo ?? item.bo,
                            disputeRule: event.target.value
                          }
                        }))
                      }
                      className="rounded-md border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] px-2 py-1 text-sm text-[color:var(--foreground)]"
                    >
                      {RULES.map((rule) => (
                        <option key={rule} value={rule}>
                          {ruleLabel(game, rule)}
                        </option>
                      ))}
                    </select>
                  </td>
                  <td className="px-3 py-2">
                    <button
                      type="button"
                      disabled={savingRound === item.round}
                      onClick={() => void saveRound(item.round)}
                      className="rounded-md border border-[color:var(--panel-border)] px-2 py-1 text-xs hover:opacity-80 disabled:opacity-50"
                    >
                      {savingRound === item.round ? "Сохранение..." : "Сохранить"}
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}
    </section>
  );
}
