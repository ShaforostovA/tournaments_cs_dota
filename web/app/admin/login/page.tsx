"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";

import { Button } from "@/components/ui/button";

export default function AdminLoginPage() {
  const router = useRouter();
  const [login, setLogin] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    setError(null);
    setLoading(true);

    try {
      const baseUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
      const res = await fetch(`${baseUrl}/api/v1/admin/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ login, password })
      });

      if (!res.ok) {
        setError("Неверный логин или пароль");
        return;
      }

      router.push("/admin");
    } catch (err) {
      setError("Не удалось войти. Попробуйте позже.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <main className="min-h-screen page-shell text-[color:var(--foreground)]">
      <div className="mx-auto flex min-h-screen max-w-md flex-col justify-center px-6">
        <h1 className="text-2xl font-semibold">Вход администратора</h1>
        <p className="mt-2 text-sm text-[color:var(--text-muted)]">Используйте учётные данные администратора.</p>

        <form onSubmit={submit} className="mt-6 space-y-4">
          <label className="block">
            <span className="text-sm text-[color:var(--foreground)]">Логин</span>
            <input
              className="mt-1 w-full rounded-md border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] px-3 py-2 text-sm text-[color:var(--foreground)]"
              value={login}
              onChange={(e) => setLogin(e.target.value)}
              autoComplete="username"
            />
          </label>

          <label className="block">
            <span className="text-sm text-[color:var(--foreground)]">Пароль</span>
            <input
              type="password"
              className="mt-1 w-full rounded-md border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] px-3 py-2 text-sm text-[color:var(--foreground)]"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete="current-password"
            />
          </label>

          {error ? <p className="text-sm text-red-400">{error}</p> : null}

          <Button type="submit" disabled={loading} className="w-full">
            {loading ? "Вход..." : "Войти"}
          </Button>
        </form>
      </div>
    </main>
  );
}
