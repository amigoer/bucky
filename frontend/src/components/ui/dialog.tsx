import { XIcon } from "lucide-react";
import { Dialog as DialogPrimitive } from "radix-ui";
import type * as React from "react";
import { cn } from "@/lib/utils";

function Dialog(props: React.ComponentProps<typeof DialogPrimitive.Root>) {
  return <DialogPrimitive.Root data-slot="dialog" {...props} />;
}

function DialogTrigger(props: React.ComponentProps<typeof DialogPrimitive.Trigger>) {
  return <DialogPrimitive.Trigger data-slot="dialog-trigger" {...props} />;
}

function DialogClose(props: React.ComponentProps<typeof DialogPrimitive.Close>) {
  return <DialogPrimitive.Close data-slot="dialog-close" {...props} />;
}

function DialogOverlay({
  className,
  ...props
}: React.ComponentProps<typeof DialogPrimitive.Overlay>) {
  return (
    <DialogPrimitive.Overlay
      data-slot="dialog-overlay"
      className={cn(
        "fixed inset-0 z-50 bg-scrim data-[state=open]:animate-in data-[state=open]:fade-in-0 data-[state=closed]:animate-out data-[state=closed]:fade-out-0",
        className,
      )}
      {...props}
    />
  );
}

/** Centred panel; `placement="top"` drops it from below the title bar like a sheet. */
function DialogContent({
  className,
  children,
  placement = "center",
  overlayClassName,
  ...props
}: React.ComponentProps<typeof DialogPrimitive.Content> & {
  placement?: "center" | "top" | "sheet";
  overlayClassName?: string;
}) {
  return (
    <DialogPrimitive.Portal>
      <DialogOverlay className={overlayClassName} />
      <DialogPrimitive.Content
        data-slot="dialog-content"
        aria-describedby={undefined}
        className={cn(
          "fixed left-1/2 z-50 flex -translate-x-1/2 flex-col overflow-hidden bg-popover text-popover-foreground shadow-sheet outline-none",
          placement === "center" &&
            "top-1/2 max-h-[calc(100%-60px)] -translate-y-1/2 rounded-2xl data-[state=open]:animate-zoom data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-95",
          placement === "top" &&
            "top-20 max-h-[calc(100%-100px)] rounded-2xl data-[state=open]:animate-zoom data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-95",
          placement === "sheet" &&
            "top-[var(--titlebar-height)] max-h-[calc(100%-var(--titlebar-height)-16px)] rounded-b-2xl data-[state=open]:animate-sheet data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:slide-out-to-top-4",
          className,
        )}
        {...props}
      >
        {children}
      </DialogPrimitive.Content>
    </DialogPrimitive.Portal>
  );
}

function DialogHeader({
  className,
  icon,
  title,
  description,
  children,
  close = true,
  iconClassName,
}: {
  className?: string;
  icon?: React.ReactNode;
  title: React.ReactNode;
  description?: React.ReactNode;
  children?: React.ReactNode;
  close?: boolean;
  iconClassName?: string;
}) {
  return (
    <div className={cn("flex flex-none items-center gap-3 py-3.5 pr-3.5 pl-4.5", className)}>
      {icon ? (
        <span
          className={cn(
            "grid size-8.5 flex-none place-items-center rounded-[10px] bg-primary-soft text-primary-text [&_svg]:size-4",
            iconClassName,
          )}
        >
          {icon}
        </span>
      ) : null}
      <div className="min-w-0 flex-1">
        <DialogPrimitive.Title className="truncate text-[15px] font-[650]">
          {title}
        </DialogPrimitive.Title>
        {description ? (
          <DialogPrimitive.Description className="truncate text-[11.5px] text-muted-foreground">
            {description}
          </DialogPrimitive.Description>
        ) : null}
      </div>
      {children}
      {close ? (
        <DialogPrimitive.Close className="grid size-7 place-items-center rounded-md text-muted-foreground transition-colors hover:bg-hover hover:text-foreground">
          <XIcon className="size-4" />
          <span className="sr-only">Close</span>
        </DialogPrimitive.Close>
      ) : null}
    </div>
  );
}

function DialogFooter({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="dialog-footer"
      className={cn(
        "flex flex-none items-center justify-end gap-2 border-t bg-sidebar px-4.5 py-3",
        className,
      )}
      {...props}
    />
  );
}

const DialogTitle = DialogPrimitive.Title;
const DialogDescription = DialogPrimitive.Description;

export {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogOverlay,
  DialogTitle,
  DialogTrigger,
};
