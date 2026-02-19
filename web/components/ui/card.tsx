import * as React from "react";

import { cn } from "@/lib/utils";

const Card = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => (
  <div
    ref={ref}
    className={cn(
      "rounded-lg border border-[color:var(--panel-border)] bg-[color:var(--panel-bg)] text-[color:var(--foreground)] shadow-sm",
      className
    )}
    {...props}
  />
));
Card.displayName = "Card";

export { Card };
