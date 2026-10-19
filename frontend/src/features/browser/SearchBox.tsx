import { useQueries } from "@tanstack/react-query";
import { DatabaseIcon, FolderIcon, SearchIcon, XIcon } from "lucide-react";
import { type ReactNode, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Tip } from "@/components/ui/tooltip";
import { api } from "@/lib/api";
import { entriesFromPages } from "@/lib/entries";
import type { Location } from "@/lib/nav";
import { useNav } from "@/lib/nav";
import { keys as queryKeys, useListing, useProfiles } from "@/lib/queries";
import { cn } from "@/lib/utils";

interface Target {
  id: string;
  label: string;
  hint: string;
  icon: ReactNode;
  location: Location;
}

/** ⌘F palette: jumps to connections, buckets and folders; objects appear in the browser. */
export function SearchBox({ location }: { location: Location }) {
  const { t } = useTranslation();
  const open = useNav((s) => s.searchOpen);
  const query = useNav((s) => s.query);
  const setSearchOpen = useNav((s) => s.setSearchOpen);
  const setQuery = useNav((s) => s.setQuery);
  const navigate = useNav((s) => s.navigate);
  const containerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const [highlight, setHighlight] = useState(0);

  const q = query.trim().toLowerCase();
  const { data: profiles = [] } = useProfiles();
  const { data: listing } = useListing(location);

  // Bucket discovery only runs for connections that do not pin a bucket.
  const floating = useMemo(() => profiles.filter((p) => !p.bucket), [profiles]);
  const bucketQueries = useQueries({
    queries: floating.map((profile) => ({
      queryKey: queryKeys.buckets(profile.id),
      queryFn: () => api.Buckets(profile.id),
      enabled: open && q.length > 0,
      staleTime: 5 * 60_000,
      retry: false,
    })),
  });

  const targets = useMemo<Target[]>(() => {
    if (!q) return [];
    const out: Target[] = [];
    profiles.forEach((profile) => {
      if (profile.bucket) {
        if (profile.name.toLowerCase().includes(q) || profile.bucket.toLowerCase().includes(q)) {
          out.push({
            id: `connection:${profile.id}`,
            label: profile.name,
            hint: profile.bucket,
            icon: <DatabaseIcon className="size-3.5" />,
            location: { profileId: profile.id, bucket: profile.bucket, prefix: "" },
          });
        }
        return;
      }
      const index = floating.findIndex((p) => p.id === profile.id);
      const buckets = index >= 0 ? (bucketQueries[index]?.data ?? []) : [];
      for (const bucket of buckets) {
        if (bucket.name.toLowerCase().includes(q) || profile.name.toLowerCase().includes(q)) {
          out.push({
            id: `bucket:${profile.id}:${bucket.name}`,
            label: bucket.name,
            hint: profile.name,
            icon: <DatabaseIcon className="size-3.5" />,
            location: { profileId: profile.id, bucket: bucket.name, prefix: "" },
          });
        }
      }
    });
    const folders = entriesFromPages(listing?.pages ?? [], location.prefix).filter(
      (entry) => entry.folder && entry.name.toLowerCase().includes(q),
    );
    for (const folder of folders) {
      out.push({
        id: `folder:${folder.key}`,
        label: folder.name,
        hint: location.prefix || location.bucket,
        icon: <FolderIcon className="size-3.5" />,
        location: { ...location, prefix: folder.key },
      });
    }
    return out.slice(0, 8);
  }, [q, profiles, floating, bucketQueries, listing, location]);

  // biome-ignore lint/correctness/useExhaustiveDependencies: reset the highlighted row when the query changes
  useEffect(() => {
    setHighlight(0);
  }, [q]);

  useEffect(() => {
    if (!open) return;
    const frame = requestAnimationFrame(() => {
      inputRef.current?.focus();
      inputRef.current?.select();
    });
    return () => cancelAnimationFrame(frame);
  }, [open]);

  useEffect(() => {
    if (!open) return;
    const onPointerDown = (event: PointerEvent) => {
      if (!containerRef.current?.contains(event.target as Node)) setSearchOpen(false);
    };
    document.addEventListener("pointerdown", onPointerDown);
    return () => document.removeEventListener("pointerdown", onPointerDown);
  }, [open, setSearchOpen]);

  const close = () => {
    setSearchOpen(false);
    setQuery("");
  };

  const activate = (target: Target) => {
    navigate(target.location);
    close();
  };

  if (!open && !query) {
    return (
      <Tip label={t("browser.searchTip")}>
        <Button
          variant="ghost"
          size="icon"
          aria-label={t("browser.searchTip")}
          onClick={() => setSearchOpen(true)}
        >
          <SearchIcon className="size-4" />
        </Button>
      </Tip>
    );
  }

  return (
    <div ref={containerRef} className="relative w-[260px] flex-none animate-fade">
      <div className="relative flex items-center">
        <SearchIcon className="pointer-events-none absolute left-2 size-3.5 text-faint" />
        <input
          ref={inputRef}
          value={query}
          placeholder={t("browser.searchPlaceholder")}
          onChange={(event) => setQuery(event.target.value)}
          onFocus={() => setSearchOpen(true)}
          onKeyDown={(event) => {
            if (event.key === "Escape") {
              event.preventDefault();
              event.stopPropagation();
              close();
            } else if (event.key === "ArrowDown") {
              event.preventDefault();
              setHighlight((h) => Math.min(targets.length - 1, h + 1));
            } else if (event.key === "ArrowUp") {
              event.preventDefault();
              setHighlight((h) => Math.max(0, h - 1));
            } else if (event.key === "Enter" && targets[highlight]) {
              event.preventDefault();
              activate(targets[highlight]);
            }
          }}
          className="h-7.5 w-full rounded-lg bg-field pr-7 pl-7.5 text-[13px] outline-none shadow-[inset_0_0_0_1px_var(--border)] placeholder:text-faint focus-visible:shadow-[inset_0_0_0_1px_var(--primary),0_0_0_2.5px_var(--primary-soft)]"
        />
        {query ? (
          <button
            type="button"
            onClick={close}
            aria-label={t("search.clear")}
            className="absolute right-1.5 grid size-4.5 place-items-center rounded-full text-faint hover:bg-muted hover:text-foreground"
          >
            <XIcon className="size-3" />
          </button>
        ) : null}
      </div>
      {open ? (
        <div className="absolute top-9.5 right-0 z-50 w-[340px] rounded-xl bg-popover p-1.5 text-[13px] shadow-pop animate-fade">
          {!q ? (
            <p className="px-2 py-1.5 text-[11.5px] leading-relaxed text-muted-foreground">
              {t("search.hint")}
            </p>
          ) : (
            <>
              <div className="px-2 pt-1 pb-1 text-[11px] font-semibold text-faint">
                {t("search.jumpTo")}
              </div>
              {targets.length === 0 ? (
                <p className="px-2 py-1.5 text-[11.5px] text-muted-foreground">
                  {t("search.noJump")}
                </p>
              ) : (
                targets.map((target, index) => (
                  <button
                    type="button"
                    key={target.id}
                    onMouseEnter={() => setHighlight(index)}
                    onClick={() => activate(target)}
                    className={cn(
                      "flex h-7.5 w-full items-center gap-2 rounded-md px-2 text-left",
                      index === highlight
                        ? "bg-primary text-primary-foreground"
                        : "text-foreground",
                    )}
                  >
                    <span className={index === highlight ? "opacity-90" : "text-muted-foreground"}>
                      {target.icon}
                    </span>
                    <span className="min-w-0 flex-1 truncate">{target.label}</span>
                    <span
                      className={cn(
                        "max-w-[45%] truncate text-[11px]",
                        index === highlight ? "text-white/70" : "text-faint",
                      )}
                    >
                      {target.hint}
                    </span>
                  </button>
                ))
              )}
              <div className="mt-1 flex items-center gap-2 border-t px-2 pt-1.5 pb-1 text-[10.5px] text-faint">
                <span>{t("search.keySelect")}</span>
                <span>{t("search.keyOpen")}</span>
                <span className="ml-auto">{t("search.keyClose")}</span>
              </div>
            </>
          )}
        </div>
      ) : null}
    </div>
  );
}
