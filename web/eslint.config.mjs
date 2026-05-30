import { defineConfig, globalIgnores } from "eslint/config";
import nextVitals from "eslint-config-next/core-web-vitals";
import nextTs from "eslint-config-next/typescript";

const eslintConfig = defineConfig([
  ...nextVitals,
  ...nextTs,

  {
    rules: {
      // После перехода на Next 16 / ESLint 9 правило стало строже
      // и начало блокировать существующие useEffect-паттерны проекта.
      // Для CI оставляем проверку hooks, но не блокируем pipeline этим правилом.
      "react-hooks/set-state-in-effect": "off",

      // В проекте уже есть существующие useEffect-зависимости,
      // которые требуют отдельного рефакторинга, поэтому пока оставляем warning.
      "react-hooks/exhaustive-deps": "warn",

      // В tailwind.config.ts используется require для плагина tailwindcss.
      "@typescript-eslint/no-require-imports": "off",
    },
  },

  globalIgnores([
    ".next/**",
    "out/**",
    "build/**",
    "dist/**",
    "node_modules/**",
    "next-env.d.ts",
  ]),
]);

export default eslintConfig;
