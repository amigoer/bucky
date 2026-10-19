import { cva, type VariantProps } from "class-variance-authority";
import type * as React from "react";
import { cn } from "@/lib/utils";

const badgeVariants = cva(
  "inline-flex h-4.5 shrink-0 items-center gap-[3px] whitespace-nowrap rounded-sm px-1.5 text-[10.5px] font-semibold [&>svg]:size-2.5",
  {
    variants: {
      variant: {
        default: "bg-primary-soft text-primary-text",
        secondary: "bg-muted text-muted-foreground",
        warning: "bg-warning-soft text-warning-text",
        destructive: "bg-destructive-soft text-destructive",
        success: "bg-success-soft text-success-text",
      },
    },
    defaultVariants: { variant: "default" },
  },
);

function Badge({
  className,
  variant,
  ...props
}: React.ComponentProps<"span"> & VariantProps<typeof badgeVariants>) {
  return (
    <span data-slot="badge" className={cn(badgeVariants({ variant }), className)} {...props} />
  );
}

export { Badge, badgeVariants };
