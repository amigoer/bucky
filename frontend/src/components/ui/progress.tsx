import { Progress as ProgressPrimitive } from "radix-ui";
import type * as React from "react";
import { cn } from "@/lib/utils";

function Progress({
  className,
  indicatorClassName,
  value,
  striped = false,
  ...props
}: React.ComponentProps<typeof ProgressPrimitive.Root> & {
  indicatorClassName?: string;
  striped?: boolean;
}) {
  const percent = Math.min(100, Math.max(0, value ?? 0));
  return (
    <ProgressPrimitive.Root
      data-slot="progress"
      value={percent}
      className={cn("relative h-1.25 w-full overflow-hidden rounded-full bg-muted", className)}
      {...props}
    >
      <ProgressPrimitive.Indicator
        className={cn(
          "h-full rounded-full bg-primary transition-[width] duration-500 ease-linear",
          striped && "stripes animate-stripes",
          indicatorClassName,
        )}
        style={{ width: `${percent}%` }}
      />
    </ProgressPrimitive.Root>
  );
}

export { Progress };
