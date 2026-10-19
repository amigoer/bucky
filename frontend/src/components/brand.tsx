import logo from "@/assets/logo.png";
import { cn } from "@/lib/utils";

export function Logo({ className }: { className?: string }) {
  return (
    <img
      src={logo}
      alt=""
      draggable={false}
      className={cn(
        "size-5 flex-none rounded-[6px] shadow-[inset_0_0_0_0.5px_rgb(0_0_0/0.12),0_1px_2px_rgb(0_0_0/0.12)]",
        className,
      )}
    />
  );
}

export function Brand({ className }: { className?: string }) {
  return (
    <span className={cn("flex items-center gap-1.75", className)}>
      <Logo />
      <span className="text-sm font-bold tracking-[-0.02em]">Bucky</span>
    </span>
  );
}
