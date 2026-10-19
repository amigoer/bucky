import { ToggleGroup as ToggleGroupPrimitive } from "radix-ui";
import type * as React from "react";
import { cn } from "@/lib/utils";

export interface SegmentOption<T extends string> {
  value: T;
  label?: React.ReactNode;
  icon?: React.ReactNode;
  title?: string;
  disabled?: boolean;
}

/** macOS-style segmented control with a sliding thumb. */
function Segmented<T extends string>({
  value,
  onValueChange,
  options,
  className,
  itemClassName,
  "aria-label": ariaLabel,
}: {
  value: T;
  onValueChange: (value: T) => void;
  options: SegmentOption<T>[];
  className?: string;
  itemClassName?: string;
  "aria-label"?: string;
}) {
  const index = Math.max(
    0,
    options.findIndex((o) => o.value === value),
  );
  const n = options.length;
  return (
    <ToggleGroupPrimitive.Root
      type="single"
      value={value}
      aria-label={ariaLabel}
      onValueChange={(next) => {
        if (next) onValueChange(next as T);
      }}
      className={cn("relative grid flex-none rounded-lg bg-muted p-0.5", className)}
      style={{ gridTemplateColumns: `repeat(${n}, minmax(0, 1fr))` }}
    >
      <span
        aria-hidden
        className="absolute top-0.5 bottom-0.5 rounded-md bg-thumb shadow-thumb transition-[left] duration-300 ease-mac"
        style={{
          left: `calc(2px + ${index} * (100% - 4px) / ${n})`,
          width: `calc((100% - 4px) / ${n})`,
        }}
      />
      {options.map((o) => (
        <ToggleGroupPrimitive.Item
          key={o.value}
          value={o.value}
          title={o.title}
          disabled={o.disabled}
          className={cn(
            "relative z-1 flex h-6 items-center justify-center gap-1.25 whitespace-nowrap rounded-md px-2.5 text-xs font-medium text-muted-foreground outline-none transition-colors data-[state=on]:font-semibold data-[state=on]:text-foreground disabled:opacity-40 focus-visible:ring-2 focus-visible:ring-primary-soft [&_svg]:size-3.5 [&_svg]:shrink-0",
            itemClassName,
          )}
        >
          {o.icon}
          {o.label}
        </ToggleGroupPrimitive.Item>
      ))}
    </ToggleGroupPrimitive.Root>
  );
}

export { Segmented };
