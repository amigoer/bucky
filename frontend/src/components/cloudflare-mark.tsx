import { cn } from "@/lib/utils";

/**
 * Cloudflare's cloud mark. Built from overlapping shapes instead of a traced
 * path so it stays crisp at any size; the light shape sits behind and peeks out
 * along the top and right edge.
 */
export function CloudflareMark({ size = 28, className }: { size?: number; className?: string }) {
  return (
    <svg
      viewBox="0 0 24 24"
      width={size}
      height={size}
      aria-hidden
      className={cn("flex-none", className)}
    >
      <g fill="#FBAD41">
        <circle cx="10.1" cy="11.9" r="3.8" />
        <circle cx="14.4" cy="8.8" r="4.8" />
        <circle cx="18.8" cy="12.2" r="3.4" />
        <rect x="10.1" y="10.4" width="8.7" height="5.3" />
      </g>
      <g fill="#F6821F">
        <circle cx="8.5" cy="13.5" r="3.8" />
        <circle cx="12.8" cy="10.4" r="4.8" />
        <circle cx="17.2" cy="13.8" r="3.4" />
        <rect x="8.5" y="12" width="8.7" height="5.3" />
      </g>
    </svg>
  );
}
