import { Switch as SwitchPrimitive } from "radix-ui";
import type * as React from "react";
import { cn } from "@/lib/utils";

function Switch({ className, ...props }: React.ComponentProps<typeof SwitchPrimitive.Root>) {
  return (
    <SwitchPrimitive.Root
      data-slot="switch"
      className={cn(
        "peer relative inline-flex h-4.5 w-7.5 shrink-0 items-center rounded-full bg-border-strong outline-none transition-colors duration-200 data-[state=checked]:bg-primary focus-visible:ring-[3px] focus-visible:ring-primary-soft disabled:opacity-50",
        className,
      )}
      {...props}
    >
      <SwitchPrimitive.Thumb className="pointer-events-none block size-3.5 translate-x-0.5 rounded-full bg-white shadow-[0_1px_3px_rgb(0_0_0/0.25)] transition-transform duration-200 ease-spring data-[state=checked]:translate-x-3.5" />
    </SwitchPrimitive.Root>
  );
}

export { Switch };
