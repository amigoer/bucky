import type * as React from "react";
import { cn } from "@/lib/utils";

function Input({ className, type, ...props }: React.ComponentProps<"input">) {
  return (
    <input
      type={type}
      data-slot="input"
      autoComplete="off"
      spellCheck={false}
      className={cn(
        "h-7.5 w-full min-w-0 rounded-lg bg-field px-2.5 text-[13px] text-foreground shadow-[inset_0_0_0_1px_var(--border)] outline-none transition-shadow placeholder:text-faint",
        "focus-visible:shadow-[inset_0_0_0_1px_var(--primary),0_0_0_3px_var(--primary-soft)] focus-visible:outline-none",
        "aria-invalid:shadow-[inset_0_0_0_1px_var(--destructive),0_0_0_3px_var(--destructive-soft)]",
        "disabled:cursor-not-allowed disabled:opacity-50",
        className,
      )}
      {...props}
    />
  );
}

export { Input };
