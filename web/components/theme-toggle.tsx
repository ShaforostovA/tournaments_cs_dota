"use client";

import { useEffect, useState } from "react";

const STORAGE_KEY = "theme";

export function ThemeToggle() {
  const [theme, setTheme] = useState<"light" | "dark">("dark");

  useEffect(() => {
    const stored = window.localStorage.getItem(STORAGE_KEY);
    const next = stored === "light" ? "light" : "dark";
    setTheme(next);
    document.documentElement.dataset.theme = next;
  }, []);

  const toggle = () => {
    const next = theme === "dark" ? "light" : "dark";
    setTheme(next);
    document.documentElement.dataset.theme = next;
    window.localStorage.setItem(STORAGE_KEY, next);
  };

  return (
    <button
      type="button"
      onClick={toggle}
      className="rounded-full border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] px-3 py-1 text-[11px] uppercase tracking-[0.2em] text-[color:var(--text-muted)] hover:opacity-80"
    >
      {theme === "dark" ? "Тёмная" : "Светлая"}
    </button>
  );
}
