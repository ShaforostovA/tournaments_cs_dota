"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";

import { Button } from "@/components/ui/button";

export function CreateTournamentForm() {
  const router = useRouter();
  const [name, setName] = useState("");
  const [game, setGame] = useState("DOTA2");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    setError(null);

    const cleanName = name.trim();
    if (!cleanName) {
      setError("Введите название турнира.");
      return;
    }

    setLoading(true);
    try {
      const baseUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
      const res = await fetch(`${baseUrl}/api/v1/admin/tournaments`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ name: cleanName, game })
      });

      if (!res.ok) {
        const text = await res.text();
        setError(text || "Не удалось создать турнир.");
        return;
      }

      setName("");
      setGame("DOTA2");
      router.refresh();
    } catch {
      setError("Не удалось создать турнир.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <form onSubmit={submit} className="mt-6 grid gap-3 rounded-xl border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] p-4 md:grid-cols-3">
      <label className="text-xs text-[color:var(--text-muted)]">
        Название турнира
        <input
          value={name}
          onChange={(event) => setName(event.target.value)}
          placeholder="Например, Winter Clash"
          className="mt-1 w-full rounded-md border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] px-2 py-2 text-sm text-[color:var(--foreground)]"
        />
      </label>

      <label className="text-xs text-[color:var(--text-muted)]">
        Игра
        <select
          value={game}
          onChange={(event) => setGame(event.target.value)}
          className="mt-1 w-full rounded-md border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] px-2 py-2 text-sm text-[color:var(--foreground)]"
        >
          <option value="DOTA2">Dota 2</option>
          <option value="CS2">CS2</option>
        </select>
      </label>

      <div className="flex items-end">
        <Button type="submit" disabled={loading} className="w-full">
          {loading ? "Создание..." : "Создать турнир"}
        </Button>
      </div>

      {error ? <p className="text-sm text-red-400 md:col-span-3">{error}</p> : null}
    </form>
  );
}
