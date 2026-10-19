import { cn } from "@/lib/utils";

/** Provider mark; R2 is the only provider in this version. */
export function ConnTile({ size = 20, className }: { size?: number; className?: string }) {
  return (
    <span
      className={cn(
        "grid flex-none place-items-center rounded-[6px] bg-[oklch(0.56_0.08_55)] font-mono font-bold tracking-[-0.02em] text-white",
        className,
      )}
      style={{ width: size, height: size, fontSize: Math.round(size * 0.4) }}
    >
      R2
    </span>
  );
}
