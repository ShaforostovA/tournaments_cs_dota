import { Button } from "@/components/ui/button";

type PaginationProps = {
  page: number;
  totalPages: number;
};

export function Pagination({ page, totalPages }: PaginationProps) {
  const canPrev = page > 1;
  const canNext = page < totalPages;

  return (
    <div className="flex items-center justify-between text-xs text-[color:var(--text-muted)]">
      <Button variant="outline" size="sm" disabled={!canPrev}>
        Назад
      </Button>
      <span>
        Страница {page} из {totalPages}
      </span>
      <Button variant="outline" size="sm" disabled={!canNext}>
        Вперёд
      </Button>
    </div>
  );
}
