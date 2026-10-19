import { ChevronRightIcon, DatabaseIcon } from "lucide-react";
import { breadcrumbs } from "@/lib/format";
import type { Location } from "@/lib/nav";
import { cn } from "@/lib/utils";

export function Breadcrumbs({
  location,
  onNavigate,
}: {
  location: Location;
  onNavigate: (prefix: string) => void;
}) {
  const crumbs = breadcrumbs(location.prefix);
  return (
    <nav className="flex min-w-0 items-center gap-0.5 overflow-x-auto [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
      <button
        type="button"
        onClick={() => onNavigate("")}
        className={cn(
          "flex h-6.5 max-w-44 shrink-0 items-center gap-1.5 rounded-md px-1.75 text-[13px] transition-colors hover:bg-hover",
          crumbs.length === 0 ? "font-semibold text-foreground" : "text-muted-foreground",
        )}
      >
        <DatabaseIcon className="size-3.5 flex-none" />
        <span className="truncate">{location.bucket}</span>
      </button>
      {crumbs.map((crumb, index) => {
        const last = index === crumbs.length - 1;
        return (
          <span key={crumb.prefix} className="flex shrink-0 items-center">
            <ChevronRightIcon className="mx-0.5 size-3 flex-none text-faint" />
            <button
              type="button"
              onClick={() => onNavigate(crumb.prefix)}
              className={cn(
                "h-6.5 max-w-52 truncate rounded-md px-1.75 text-[13px] transition-colors hover:bg-hover",
                last ? "font-semibold text-foreground" : "text-muted-foreground",
              )}
            >
              {crumb.name}
            </button>
          </span>
        );
      })}
    </nav>
  );
}
