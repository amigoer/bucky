import type * as React from "react";
import { cn } from "@/lib/utils";

export function EmptyState({
  icon,
  title,
  description,
  tone = "muted",
  action,
  className,
}: {
  icon: React.ReactNode;
  title: React.ReactNode;
  description?: React.ReactNode;
  tone?: "muted" | "bad" | "warn" | "accent";
  action?: React.ReactNode;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "flex max-w-[380px] flex-col items-center gap-2 px-6 text-center animate-fade",
        className,
      )}
    >
      <span
        className={cn(
          "mb-1 grid size-14 place-items-center rounded-3xl [&_svg]:size-7",
          tone === "muted" && "bg-muted text-muted-foreground",
          tone === "bad" && "bg-destructive-soft text-destructive",
          tone === "warn" && "bg-warning-soft text-warning-text",
          tone === "accent" && "bg-primary-soft text-primary-text",
        )}
      >
        {icon}
      </span>
      <span className="text-[15px] font-semibold">{title}</span>
      {description ? (
        <span className="text-[12.5px] leading-relaxed text-muted-foreground">{description}</span>
      ) : null}
      {action ? <div className="mt-2 flex gap-2">{action}</div> : null}
    </div>
  );
}
