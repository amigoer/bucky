import type * as React from "react";
import { cn } from "@/lib/utils";

function Kbd({ className, ...props }: React.ComponentProps<"kbd">) {
  return (
    <kbd
      data-slot="kbd"
      className={cn(
        "inline-flex h-5 min-w-5.5 items-center justify-center rounded-sm bg-sidebar px-1.5 font-mono text-[11px] text-muted-foreground shadow-[inset_0_0_0_1px_var(--border-strong),0_1px_0_var(--border-strong)]",
        className,
      )}
      {...props}
    />
  );
}

export { Kbd };
