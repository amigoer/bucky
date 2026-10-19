import type * as React from "react";
import { cn } from "@/lib/utils";

function Textarea({ className, ...props }: React.ComponentProps<"textarea">) {
  return (
    <textarea
      data-slot="textarea"
      spellCheck={false}
      className={cn(
        "min-h-16 w-full resize-none rounded-lg bg-field px-2.5 py-2 font-mono text-xs text-foreground shadow-[inset_0_0_0_1px_var(--border)] outline-none placeholder:text-faint focus-visible:shadow-[inset_0_0_0_1px_var(--primary),0_0_0_3px_var(--primary-soft)] focus-visible:outline-none",
        className,
      )}
      {...props}
    />
  );
}

export { Textarea };
