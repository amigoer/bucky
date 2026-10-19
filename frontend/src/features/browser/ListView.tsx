import { useVirtualizer } from "@tanstack/react-virtual";
import { ChevronDownIcon, ChevronUpIcon } from "lucide-react";
import { useEffect, useRef } from "react";
import { useTranslation } from "react-i18next";
import { FileTile } from "@/components/file-tile";
import type { SortKey } from "@/lib/entries";
import { formatBytes, formatDate } from "@/lib/format";
import { cn } from "@/lib/utils";
import { EntryContextMenu } from "./EntryContextMenu";
import type { EntryViewProps } from "./types";

const ROW = 36;
const GRID = "grid-cols-[minmax(0,1fr)_92px_150px_80px]";

export function ListView({
  entries,
  selection,
  focusKey,
  onSelect,
  onOpen,
  onLoadMore,
  location,
  readOnly,
  sortKey,
  sortDir,
  onSort,
}: EntryViewProps & {
  sortKey: SortKey;
  sortDir: 1 | -1;
  onSort: (key: SortKey) => void;
}) {
  const { t, i18n } = useTranslation();
  const scrollRef = useRef<HTMLDivElement>(null);
  const virtualizer = useVirtualizer({
    count: entries.length,
    getScrollElement: () => scrollRef.current,
    estimateSize: () => ROW,
    overscan: 16,
  });

  const virtualItems = virtualizer.getVirtualItems();
  const lastVisible = virtualItems.at(-1)?.index ?? 0;
  useEffect(() => {
    if (entries.length > 0 && lastVisible >= entries.length - 8) onLoadMore?.();
  }, [lastVisible, entries.length, onLoadMore]);

  const focusIndex = focusKey ? entries.findIndex((entry) => entry.key === focusKey) : -1;
  useEffect(() => {
    if (focusIndex >= 0) virtualizer.scrollToIndex(focusIndex, { align: "auto" });
  }, [focusIndex, virtualizer]);

  const header = (key: SortKey, label: string, className?: string) => (
    <button
      type="button"
      onClick={() => onSort(key)}
      className={cn(
        "flex h-full min-w-0 items-center gap-1 text-left text-[11px] font-semibold text-faint transition-colors hover:text-muted-foreground",
        sortKey === key && "text-muted-foreground",
        className,
      )}
    >
      <span className="truncate">{label}</span>
      {sortKey === key ? (
        sortDir === 1 ? (
          <ChevronUpIcon className="size-2.75 flex-none" strokeWidth={2.4} />
        ) : (
          <ChevronDownIcon className="size-2.75 flex-none" strokeWidth={2.4} />
        )
      ) : null}
    </button>
  );

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className={cn("grid h-8 flex-none items-center gap-2 px-5", GRID)}>
        {header("name", t("browser.colName"))}
        {header("size", t("browser.colSize"), "justify-end")}
        {header("modified", t("browser.colModified"))}
        <span className="text-[11px] font-semibold text-faint">{t("browser.colKind")}</span>
      </div>
      <div
        ref={scrollRef}
        className="min-h-0 flex-1 overflow-y-auto px-2 pb-2"
        role="listbox"
        aria-multiselectable
      >
        <div className="relative" style={{ height: virtualizer.getTotalSize() }}>
          {virtualItems.map((item) => {
            const entry = entries[item.index];
            const selected = selection.has(entry.key);
            return (
              <EntryContextMenu
                key={entry.key}
                entry={entry}
                location={location}
                readOnly={readOnly}
              >
                <div
                  role="option"
                  aria-selected={selected}
                  tabIndex={-1}
                  onKeyDown={(event) => {
                    if (event.key === "Enter" || event.key === " ") {
                      event.preventDefault();
                      onSelect(entry, {
                        metaKey: event.metaKey,
                        ctrlKey: event.ctrlKey,
                        shiftKey: event.shiftKey,
                      });
                    }
                  }}
                  data-entry-key={entry.key}
                  onClick={(event) =>
                    onSelect(entry, {
                      metaKey: event.metaKey,
                      ctrlKey: event.ctrlKey,
                      shiftKey: event.shiftKey,
                    })
                  }
                  onDoubleClick={() => onOpen(entry)}
                  className={cn(
                    "absolute inset-x-0 top-0 grid items-center gap-2 rounded-lg px-3 transition-colors",
                    GRID,
                    selected
                      ? "bg-primary text-primary-foreground"
                      : "hover:bg-hover active:bg-muted",
                  )}
                  style={{
                    height: ROW - 2,
                    transform: `translateY(${item.start + 1}px)`,
                  }}
                >
                  <span className="flex min-w-0 items-center gap-2.5">
                    <FileTile kind={entry.kind} size={21} selected={selected} />
                    <span className="truncate" title={entry.name}>
                      {entry.name}
                    </span>
                  </span>
                  <span
                    className={cn(
                      "truncate text-right tabular",
                      selected ? "text-white/75" : "text-muted-foreground",
                    )}
                  >
                    {entry.folder ? "—" : formatBytes(entry.size, i18n.language)}
                  </span>
                  <span
                    className={cn("truncate", selected ? "text-white/75" : "text-muted-foreground")}
                  >
                    {entry.folder ? "—" : formatDate(entry.modified, i18n.language)}
                  </span>
                  <span
                    className={cn(
                      "truncate text-[11.5px]",
                      selected ? "text-white/75" : "text-faint",
                    )}
                  >
                    {t(`kinds.${entry.kind}`)}
                  </span>
                </div>
              </EntryContextMenu>
            );
          })}
        </div>
      </div>
    </div>
  );
}
