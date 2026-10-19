import { cn } from "@/lib/utils";

function Spinner({ className }: { className?: string }) {
  return (
    <span
      aria-hidden
      className={cn(
        "inline-block size-3 shrink-0 animate-spin rounded-full border-[1.6px] border-primary border-r-transparent",
        className,
      )}
    />
  );
}

export { Spinner };
