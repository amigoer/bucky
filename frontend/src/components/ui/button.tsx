import { cva, type VariantProps } from "class-variance-authority";
import { Slot } from "radix-ui";
import type * as React from "react";
import { cn } from "@/lib/utils";

const buttonVariants = cva(
  "inline-flex shrink-0 items-center justify-center gap-1.5 whitespace-nowrap rounded-md font-medium outline-none transition-[background-color,color,box-shadow,filter,transform,opacity] duration-150 active:scale-[0.97] disabled:pointer-events-none disabled:opacity-45 focus-visible:ring-[3px] focus-visible:ring-primary-soft [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-3.5",
  {
    variants: {
      variant: {
        default:
          "bg-primary font-semibold text-primary-foreground shadow-[0_1px_1px_rgb(0_0_0/0.12),inset_0_1px_0_rgb(255_255_255/0.14)] hover:brightness-106",
        destructive:
          "bg-destructive font-semibold text-white shadow-[0_1px_1px_rgb(0_0_0/0.12)] hover:brightness-106",
        secondary: "bg-secondary text-secondary-foreground shadow-button hover:bg-hover",
        ghost: "text-muted-foreground hover:bg-hover hover:text-foreground",
        soft: "bg-primary-soft font-semibold text-primary-text hover:bg-primary-line",
        "ghost-destructive": "text-destructive hover:bg-destructive-soft",
        link: "h-auto px-0 text-primary-text underline-offset-4 hover:underline active:scale-100",
      },
      size: {
        default: "h-7 px-3 text-[13px]",
        sm: "h-6.5 px-2.5 text-xs",
        lg: "h-7.5 px-3.5 text-[13px]",
        icon: "size-7 p-0",
        "icon-sm": "size-5 rounded-sm p-0",
        "icon-lg": "size-8 rounded-lg p-0",
      },
    },
    defaultVariants: { variant: "default", size: "default" },
  },
);

function Button({
  className,
  variant,
  size,
  asChild = false,
  type = "button",
  ...props
}: React.ComponentProps<"button"> &
  VariantProps<typeof buttonVariants> & {
    asChild?: boolean;
  }) {
  const Comp = asChild ? Slot.Root : "button";
  return (
    <Comp
      data-slot="button"
      type={asChild ? undefined : type}
      className={cn(buttonVariants({ variant, size, className }))}
      {...props}
    />
  );
}

export { Button, buttonVariants };
